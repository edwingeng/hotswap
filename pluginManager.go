package hotswap

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/edwingeng/hotswap/internal/hutils"
	"github.com/edwingeng/hotswap/vault"
	"github.com/edwingeng/slog"
	"github.com/pierrec/xxHash/xxHash32"
)

type PluginManager struct {
	slog.Logger
	dirName string

	pluginMap map[string]*Plugin
	ordered   []*Plugin
	when      time.Time

	vault.Vault

	cbOpen       func(p *Plugin, data interface{})
	panicTrigger func(data interface{})
}

func newPluginManager(log slog.Logger, newExt func() interface{}) *PluginManager {
	now := time.Now().Format(hutils.CompactDateTimeFormat)
	dirName := fmt.Sprintf("%s-%d", now, os.Getpid())
	var ext interface{}
	if newExt != nil {
		ext = newExt()
	}
	v := vault.Vault{
		LiveFuncs: make(map[string]interface{}),
		LiveTypes: make(map[string]func() interface{}),
		DataBag:   make(map[string]interface{}),
		Extension: ext,
	}
	return &PluginManager{
		Logger:       log,
		dirName:      dirName,
		pluginMap:    make(map[string]*Plugin),
		Vault:        v,
		cbOpen:       func(*Plugin, interface{}) {},
		panicTrigger: func(interface{}) {},
	}
}

func (pm *PluginManager) addUnchanged(oldP *Plugin, note string) {
	newP := *oldP
	newP.unchanged = true
	newP.Note = note
	newP.Refs.Inc()
	pm.pluginMap[name2key(oldP.Name)] = &newP
}

func (pm *PluginManager) outputStats1(infoMap fileInfoMap) {
	var a1, a2 []string
	for _, p := range pm.pluginMap {
		switch p.Note {
		case "not reloadable":
			a1 = append(a1, p.Name)
		case "unchanged":
			a2 = append(a2, p.Name)
		default:
			panic("unknown plugin note")
		}
	}
	str1 := strings.Join(a1, ", ")
	str2 := strings.Join(a2, ", ")
	str3 := strings.Join(infoMap.names(), ", ")
	pm.Infof("<hotswap> not reloadable: [%s], unchanged: [%s], to be loaded: [%s]", str1, str2, str3)
}

func (pm *PluginManager) loadPlugins(files []string, oldManager *PluginManager, data interface{}) (errRet error) {
	var curFileInfo *fileInfo
	defer func() {
		if r := recover(); r != nil {
			var pName string
			if curFileInfo != nil {
				pName = "." + curFileInfo.name
			}
			errRet = fmt.Errorf("<hotswap%s> panic: %+v\n%s", pName, r, debug.Stack())
			pm.invokeEveryOnFree()
		} else if errRet != nil {
			pm.invokeEveryOnFree()
		}
	}()

	if len(pm.pluginMap) != 0 {
		return errors.New("never call loadPlugins twice")
	}

	counters := make(map[string]int)
	for _, file := range files {
		counters[name2key(pluginName(file))]++
	}
	for k, v := range counters {
		if v > 1 {
			return fmt.Errorf("duplicate name detected: %s. files: %s", k, strings.Join(files, ", "))
		}
	}

	infoMap, err := buildFileInfoMap(files)
	if err != nil {
		return err
	}

	notReloadable := infoMap.removeNotReloadable(oldManager)
	for k := range notReloadable {
		pm.addUnchanged(oldManager.pluginMap[k], "not reloadable")
	}
	unchanged := infoMap.removeUnchanged(oldManager)
	for k := range unchanged {
		pm.addUnchanged(oldManager.pluginMap[k], "unchanged")
	}
	if infoMap.Len()+len(pm.pluginMap) != len(files) {
		return errors.New("infoMap.Len()+len(pm.pluginMap) != len(files)")
	}

	pm.outputStats1(infoMap)
	pm.when = time.Now()
	for _, name := range infoMap.names() {
		info := infoMap.m[name]
		curFileInfo = info
		if err = pm.loadPlugin(info, data); err != nil {
			return fmt.Errorf("failed to load the plugin %s. err: %w", info.name, err)
		}
	}
	curFileInfo = nil
	pm.panicTrigger(data)

	if err := pm.initDeps(); err != nil {
		return err
	}
	if err := pm.invokeEveryOnLoad(data); err != nil {
		return err
	}
	if err := pm.setupVault(); err != nil {
		return err
	}
	if err := pm.invokeEveryOnInit(); err != nil {
		return err
	}

	pm.panicTrigger(data)
	return nil
}

func (pm *PluginManager) copyPlugin(info *fileInfo) (string, error) {
	tmpDir := filepath.Join(filepath.Dir(info.file), "tmp", pm.dirName)
	if err := os.MkdirAll(tmpDir, 0744); err != nil {
		return "", err
	}

	sum := xxHash32.Checksum(info.fileSha1[:], 0)
	dst := filepath.Join(tmpDir, fmt.Sprintf("%s-%#8x%s", info.name, sum, hutils.FileNameExt))
	return dst, ioutil.WriteFile(dst, info.fileData, 0644)
}

type pluginFuncItem struct {
	symbol string
	fn     interface{}
}

func makePluginFuncItemList(p *Plugin) []pluginFuncItem {
	return []pluginFuncItem{
		{"OnLoad", &p.fOnLoad},
		{"OnInit", &p.fOnInit},
		{"OnFree", &p.fOnFree},
		{"Export", &p.fExport},
		{"Import", &p.fImport},
		{"InvokeFunc", &p.InvokeFunc},
		{"Reloadable", &p.fReloadable},
		{"HotswapLiveFuncs", &p.hotswapLiveFuncs},
		{"HotswapLiveTypes", &p.hotswapLiveTypes},
	}
}

func (pm *PluginManager) loadPlugin(info *fileInfo, data interface{}) error {
	actual, err := pm.copyPlugin(info)
	if err != nil {
		return err
	}

	p := newPlugin()
	p.Name = info.name
	p.File = info.file
	p.FileSha1 = info.fileSha1
	p.When = pm.when
	p.P, err = plugin.Open(actual)
	if err != nil {
		return err
	}
	pm.cbOpen(p, data)

	var a = makePluginFuncItemList(p)
	var missing []string
	for _, v := range a {
		if err := p.Lookup(v.symbol, v.fn); err != nil {
			if err == ErrNotExist {
				missing = append(missing, v.symbol)
			} else {
				return err
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing functions: %s", strings.Join(missing, ", "))
	}

	p.reloadable, err = p.invokeReloadable()
	if err != nil {
		return err
	}
	p.exported, err = p.invokeExport()
	if err != nil {
		return err
	}

	pm.pluginMap[name2key(p.Name)] = p
	return nil
}

func (pm *PluginManager) initDeps() error {
	keys := make([]string, 0, len(pm.pluginMap))
	for k := range pm.pluginMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		p := pm.pluginMap[k]
		if p.unchanged {
			continue
		}
		imp, err := p.invokeImport()
		if err != nil {
			return err
		}
		if isNil(imp) {
			continue
		}
		impTyp := reflect.TypeOf(imp)
		if impTyp.Kind() != reflect.Ptr || impTyp.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("the return value of %s.Import() must be a pointer to a struct", p.Name)
		}
		n := impTyp.Elem().NumField()
		for i := 0; i < n; i++ {
			field := impTyp.Elem().Field(i)
			if ch := field.Name[0]; !unicode.IsUpper(rune(ch)) {
				continue
			}
			if field.Tag.Get("hotswap") == "-" {
				continue
			}
			if field.Anonymous {
				return fmt.Errorf("field of the Import() object cannot be anonymous. field: %s, plugin: %s", field.Name, p.Name)
			}
			dep, ok := pm.pluginMap[name2key(field.Name)]
			if !ok {
				return fmt.Errorf("unknown dependency: %s. plugin: %s", field.Name, p.Name)
			}
			if !p.reloadable && dep.reloadable {
				return fmt.Errorf("%s is NOT reloadable while its dependency, %s, is reloadable", p.Name, dep.Name)
			}
			if isNil(dep.exported) {
				p.Deps = append(p.Deps, dep.Name)
				continue
			}
			exportedVal := reflect.ValueOf(dep.exported)
			if !exportedVal.Type().AssignableTo(field.Type) {
				return fmt.Errorf("the return value %s.Export() is not assignable to %s.Import().%s",
					dep.Name, p.Name, field.Name)
			}
			reflect.ValueOf(imp).Elem().Field(i).Set(exportedVal)
			p.Deps = append(p.Deps, dep.Name)
		}
	}

	for _, p := range pm.pluginMap {
		if len(p.Deps) == 0 {
			continue
		}
		cyclicDeps := pm.checkCyclicDependency(p, make(map[*Plugin]struct{}))
		if len(cyclicDeps) != 0 {
			var names []string
			for _, p := range cyclicDeps {
				names = append(names, p.Name)
			}
			return fmt.Errorf("cyclic dependency detected: %s", strings.Join(names, " -> "))
		}
	}

	for _, p := range pm.pluginMap {
		if p.unchanged {
			for _, depName := range p.Deps {
				if dep := pm.pluginMap[name2key(depName)]; !dep.unchanged {
					return fmt.Errorf("%s was rebuilt while %s was not", dep.Name, p.Name)
				}
			}
		}
	}

	pm.orderPlugins()
	return nil
}

func (pm *PluginManager) checkCyclicDependency(p *Plugin, visited map[*Plugin]struct{}) []*Plugin {
	me := [1]*Plugin{p}
	if _, ok := visited[p]; ok {
		return me[:]
	}

	visited[p] = struct{}{}
	defer func() {
		delete(visited, p)
	}()

	for _, depName := range p.Deps {
		dep := pm.pluginMap[name2key(depName)]
		ret := pm.checkCyclicDependency(dep, visited)
		if len(ret) > 0 {
			return append(me[:], ret...)
		}
	}

	return nil
}

func (pm *PluginManager) orderPlugins() {
	n := len(pm.pluginMap)
	a := make([]*Plugin, 0, n)
	m := make(map[*Plugin]struct{}, n)
	var keys []string
	for k, p := range pm.pluginMap {
		if len(p.Deps) == 0 {
			a = append(a, p)
			m[p] = struct{}{}
		} else {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)
	for i := 0; i < n+1; i++ {
		if len(a) == n {
			pm.ordered = a
			return
		}
		for _, k := range keys {
			p := pm.pluginMap[k]
			if _, ok := m[p]; ok {
				continue
			}
			var counter int
			for _, depName := range p.Deps {
				dep := pm.pluginMap[name2key(depName)]
				if _, ok := m[dep]; ok {
					counter++
				} else {
					break
				}
			}
			if counter == len(p.Deps) {
				a = append(a, p)
				m[p] = struct{}{}
			}
		}
	}

	panic("something is wrong with PluginManager.orderedPlugins()")
}

func (pm *PluginManager) invokeEveryOnLoad(data interface{}) error {
	invokeImpl := func(p *Plugin) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("<hotswap:%s> panic: %+v\n%s", p.Name, r, debug.Stack())
			}
		}()
		pm.Debugf("<hotswap> invoking %s.OnLoad", p.Name)
		return p.fOnLoad(data)
	}

	for _, p := range pm.ordered {
		if p.unchanged {
			continue
		}
		if err := invokeImpl(p); err != nil {
			return err
		}
	}

	return nil
}

func (pm *PluginManager) invokeEveryOnInit() error {
	invokeImpl := func(p *Plugin) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("<hotswap:%s> panic: %+v\n%s", p.Name, r, debug.Stack())
			}
		}()
		pm.Debugf("<hotswap> invoking %s.OnInit", p.Name)
		return p.fOnInit(&pm.Vault)
	}

	for _, p := range pm.ordered {
		if p.unchanged {
			continue
		}
		if err := invokeImpl(p); err != nil {
			return err
		}
	}

	return nil
}

func (pm *PluginManager) setupVault() error {
	for i, p := range pm.ordered {
		liveFuncs := p.hotswapLiveFuncs()
		if isNil(liveFuncs) {
			return fmt.Errorf("something is wrong with HotswapLiveFuncs(). plugin: %s", p.Name)
		}
		for k, v := range liveFuncs {
			if _, ok := pm.LiveFuncs[k]; !ok {
				pm.LiveFuncs[k] = v
				continue
			}

			var another string
			for j := 0; j < i; j++ {
				m := pm.ordered[j].hotswapLiveFuncs()
				if _, ok := m[k]; ok {
					another = pm.ordered[j].Name
					break
				}
			}
			return fmt.Errorf("duplicate live function name detected: %s. plugins: %s, %s",
				k, another, p.Name)
		}
	}

	for i, p := range pm.ordered {
		liveTypes := p.hotswapLiveTypes()
		if isNil(liveTypes) {
			return fmt.Errorf("something is wrong with HotswapLiveTypes(). plugin: %s", p.Name)
		}
		for k, v := range liveTypes {
			if _, ok := pm.LiveTypes[k]; !ok {
				pm.LiveTypes[k] = v
				continue
			}

			var another string
			for j := 0; j < i; j++ {
				m := pm.ordered[j].hotswapLiveTypes()
				if _, ok := m[k]; ok {
					another = pm.ordered[j].Name
					break
				}
			}
			return fmt.Errorf("duplicate live type name detected: %s. plugins: %s, %s",
				k, another, p.Name)
		}
	}

	return nil
}

func (pm *PluginManager) invokeEveryOnFree() {
	invokeImpl := func(p *Plugin) {
		if v := p.Refs.Dec(); v > 0 {
			return
		}
		defer func() {
			if r := recover(); r != nil {
				pm.Errorf("<hotswap:%s> panic: %+v\n%s", p.Name, r, debug.Stack())
			}
		}()
		pm.Debugf("<hotswap> invoking %s.OnFree", p.Name)
		p.freeOnce.Do(func() {
			p.fOnFree()
		})
	}

	all := pm.Plugins()
	if len(all) > 0 {
		for i := len(all) - 1; i >= 0; i-- {
			invokeImpl(all[i])
		}
	} else {
		for _, p := range pm.pluginMap {
			invokeImpl(p)
		}
	}
}

func (pm *PluginManager) FindPlugin(name string) *Plugin {
	return pm.pluginMap[name2key(name)]
}

func (pm *PluginManager) Plugins() []*Plugin {
	return pm.ordered
}

func (pm *PluginManager) InvokeEach(name string, params ...interface{}) {
	invokeImpl := func(p *Plugin) {
		defer func() {
			if r := recover(); r != nil {
				pm.Errorf("<hotswap:%s> panic: %+v\n%s", p.Name, r, debug.Stack())
			}
		}()
		if _, err := p.InvokeFunc(name, params...); err != nil {
			pm.Error(err)
		}
	}

	all := pm.Plugins()
	for _, p := range all {
		invokeImpl(p)
	}
}

func (pm *PluginManager) InvokeEachBackward(name string, params ...interface{}) {
	invokeImpl := func(p *Plugin) {
		defer func() {
			if r := recover(); r != nil {
				pm.Errorf("<hotswap:%s> panic: %+v\n%s", p.Name, r, debug.Stack())
			}
		}()
		if _, err := p.InvokeFunc(name, params...); err != nil {
			pm.Error(err)
		}
	}

	all := pm.Plugins()
	for i := len(all) - 1; i >= 0; i-- {
		invokeImpl(all[i])
	}
}

type fileInfo struct {
	name     string
	file     string
	fileData []byte
	fileSha1 [sha1.Size]byte
}

type fileInfoMap struct {
	m map[string]*fileInfo
}

func (x fileInfoMap) removeNotReloadable(oldManager *PluginManager) map[string]*fileInfo {
	if oldManager == nil {
		return nil
	}

	notReloadable := make(map[string]*fileInfo)
	for k, info := range x.m {
		if oldP := oldManager.pluginMap[k]; oldP != nil {
			if !oldP.reloadable {
				notReloadable[k] = info
				delete(x.m, k)
			}
		}
	}
	return notReloadable
}

func name2key(name string) string {
	return strings.ToLower(name)
}

func (x fileInfoMap) removeUnchanged(oldManager *PluginManager) map[string]*fileInfo {
	if oldManager == nil {
		return nil
	}

	unchanged := make(map[string]*fileInfo)
	for k, info := range x.m {
		if oldP := oldManager.pluginMap[k]; oldP != nil {
			if info.fileSha1 == oldP.FileSha1 {
				unchanged[k] = info
				delete(x.m, k)
			}
		}
	}

	return unchanged
}

func (x fileInfoMap) names() []string {
	var a []string
	for _, info := range x.m {
		a = append(a, info.name)
	}
	sort.Strings(a)
	return a
}

func (x fileInfoMap) Len() int {
	return len(x.m)
}

func pluginName(file string) string {
	return strings.TrimSuffix(filepath.Base(file), hutils.FileNameExt)
}

func (x fileInfoMap) add(file string, fileData []byte) {
	name := pluginName(file)
	fileSha1 := sha1.Sum(fileData)
	info := &fileInfo{
		name:     name,
		file:     file,
		fileData: fileData,
		fileSha1: fileSha1,
	}
	k := name2key(name)
	x.m[k] = info
}

func buildFileInfoMap(files []string) (fileInfoMap, error) {
	x := fileInfoMap{
		m: make(map[string]*fileInfo),
	}
	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return x, err
		}
		x.add(file, data)
	}
	return x, nil
}
