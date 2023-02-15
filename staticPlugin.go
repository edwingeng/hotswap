package hotswap

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/edwingeng/hotswap/vault"
)

func NewPluginFuncs(
	fExport func() interface{},
	hotswapLiveFuncs func() map[string]interface{},
	hotswapLiveTypes func() map[string]func() interface{},
	fImport func() interface{},
	InvokeFunc func(name string, params ...interface{}) (interface{}, error),
	fOnFree func(),
	fOnInit func(sharedVault *vault.Vault) error,
	fOnLoad func(data interface{}) error,
	fReloadable func() bool,
) PluginFuncs {
	return PluginFuncs{
		fOnLoad:          fOnLoad,
		fOnInit:          fOnInit,
		fOnFree:          fOnFree,
		fExport:          fExport,
		fImport:          fImport,
		InvokeFunc:       InvokeFunc,
		fReloadable:      fReloadable,
		hotswapLiveFuncs: hotswapLiveFuncs,
		hotswapLiveTypes: hotswapLiveTypes,
	}
}

type StaticPlugin struct {
	Name string
	PluginFuncs
}

func (pm *PluginManager) loadStaticPlugins(staticPlugins map[string]*StaticPlugin, data interface{}) (errRet error) {
	var curPlugin *StaticPlugin
	defer func() {
		if r := recover(); r != nil {
			var pName string
			if curPlugin != nil {
				pName = "." + curPlugin.Name
			}
			errRet = fmt.Errorf("<hotswap%s> panic: %+v\n%s", pName, r, debug.Stack())
			pm.invokeEveryOnFree()
		} else if errRet != nil {
			pm.invokeEveryOnFree()
		}
	}()

	if len(pm.pluginMap) != 0 {
		return errors.New("never call loadStaticPlugins twice")
	}

	var a []string
	for k := range staticPlugins {
		a = append(a, k)
	}
	sort.Strings(a)

	pm.when = time.Now()
	for _, name := range a {
		curPlugin = staticPlugins[name]
		if err := pm.loadStaticPlugin(curPlugin); err != nil {
			return fmt.Errorf("failed to load the plugin %s. err: %w", name, err)
		}
	}
	curPlugin = nil

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

	pm.Warn("<hotswap> running under static linking mode")
	return nil
}

func (pm *PluginManager) loadStaticPlugin(sp *StaticPlugin) error {
	p := newPlugin()
	p.Name = sp.Name
	p.When = pm.when
	p.Note = "ok"
	p.PluginFuncs = sp.PluginFuncs

	var a = makePluginFuncItemList(p)
	var missing []string
	for _, v := range a {
		vv := reflect.ValueOf(v.fn)
		if isNil(vv.Elem().Interface()) {
			missing = append(missing, v.symbol)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing functions: %s", strings.Join(missing, ", "))
	}

	var err error
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

func (sw *PluginManagerSwapper) loadStaticPlugins(data interface{}, cbs []ReloadCallback) (Details, error) {
	newManager := newPluginManager(sw.Logger, sw.opts.newExt)
	staticPlugins := sw.staticPlugins
	if len(sw.opts.whitelist) > 0 {
		staticPlugins = make(map[string]*StaticPlugin)
		for _, name := range sw.opts.whitelist {
			if p, ok := sw.staticPlugins[name]; ok {
				staticPlugins[name] = p
			} else {
				return nil, fmt.Errorf("cannot find the static plugin %q", name)
			}
		}
	}

	if err := newManager.loadStaticPlugins(staticPlugins, data); err != nil {
		return nil, err
	}
	if err := invokeReloadCallbacks(cbs, newManager, nil); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for k := range staticPlugins {
		result[k] = "ok"
	}

	sw.current.Store(newManager)
	return result, nil
}
