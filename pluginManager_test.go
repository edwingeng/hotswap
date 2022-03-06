package hotswap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/edwingeng/hotswap/cli/hotswap/trial/export/importall"
	"github.com/edwingeng/hotswap/internal/hutils"
	"github.com/edwingeng/slog"
)

func init() {
	if err := os.Setenv("hotswap:checkRequiredPluginFuncs", "0"); err != nil {
		panic(err)
	}
}

func nilNewer() interface{} {
	return nil
}

func prepareEnv(t *testing.T, env string) {
	t.Helper()
	if err := os.Setenv("pluginTest", env); err != nil {
		t.Fatal(err)
	}
}

func preparePluginGroup(t *testing.T, buildArgs []string, group string, pluginNames ...string) string {
	t.Helper()
	outputDir := preparePluginGroupImpl(t, buildArgs, group, true, pluginNames...)
	return outputDir
}

func preparePluginGroupImpl(t *testing.T, buildArgs []string, group string, cleanDir bool, pluginNames ...string) string {
	t.Helper()
	group = strings.ReplaceAll(group, ":", "-")
	const playground = "cli/hotswap/trial/playground"
	outputDir := filepath.Join(playground, group)
	if cleanDir {
		if err := hutils.FindDirectory(outputDir, "outputDir"); err == nil {
			if err = os.RemoveAll(outputDir); err != nil {
				t.Fatal(err)
			}
		}
	}

	preparePlugins(t, buildArgs, outputDir, pluginNames...)
	return outputDir
}

func preparePlugins(t *testing.T, buildArgs []string, outputDir string, pluginNames ...string) {
	t.Helper()
	const exe = "cli/hotswap/hotswap"
	const homeDir = "cli/hotswap/trial"
	for _, pName := range pluginNames {
		pluginDir := filepath.Join(homeDir, pName)
		hutils.BuildPlugin(t, exe, pluginDir, outputDir, buildArgs...)
	}
}

func newScavenger() *slog.Scavenger {
	return slog.NewScavenger()
}

func completePluginPaths(outputDir string, names ...string) []string {
	var a []string
	for _, name := range names {
		withExt := name + hutils.FileNameExt
		a = append(a, filepath.Join(outputDir, withExt))
	}
	return a
}

func invariants(t *testing.T, mgr *PluginManager) {
	t.Helper()
	if len(mgr.ordered) != len(mgr.pluginMap) {
		t.Fatal("len(mgr.ordered) != len(mgr.pluginMap)")
	}
}

func TestPluginManager_loadPlugins_conflict(t *testing.T) {
	files := []string{
		"foo/arya",
		"bar/foo/aRya",
		"aryA",
		"snow",
	}
	mgr := newPluginManager(nil, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when plugin names conflict")
	} else if !strings.Contains(err.Error(), "duplicate name") || strings.Count(err.Error(), ",") != 3 {
		t.Fatal("the error returned by loadPlugins is not expected")
	}
}

func TestPluginManager_loadPlugins_one(t *testing.T) {
	pluginNames := []string{"arya"}
	outputDir := preparePluginGroup(t, nil, "one", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(mgr.pluginMap) != len(pluginNames) {
		t.Fatal("len(mgr.pluginMap) != len(pluginNames)")
	}
	invariants(t, mgr)

	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Fatal("loadPlugins should fail when it is called a second time")
	}
	invariants(t, mgr)

	if isNil(mgr.FindPlugin("arya").exported) {
		t.Fatal(`isNil(mgr.FindPlugin("arya").exported)`)
	}
	if mgr.FindPlugin("arya").fImport() != nil {
		t.Fatal(`mgr.FindPlugin("arya").Import() != nil`)
	}

	ret, err := mgr.FindPlugin("arya").InvokeFunc("polish", 1)
	if err != nil || reflect.ValueOf(ret).Kind() != reflect.String || !strings.HasPrefix(ret.(string), "Polished") {
		t.Fatalf("unexpected return value of InvokeFunc(). ret: %v, err: %v", ret, err)
	}

	if !mgr.FindPlugin("arya").reloadable {
		t.Fatal(`!mgr.FindPlugin("arya").reloadable`)
	}

	if _, ok := mgr.Vault.LiveFuncs["live_NotToday"]; !ok {
		t.Fatal("cannot find the live function, live_NotToday")
	}
	if _, ok := mgr.Vault.LiveFuncs["Live_Anyone"]; !ok {
		t.Fatal("cannot find the live function, Live_Anyone")
	}
	if _, ok := mgr.Vault.LiveTypes["Live_AryaKill"]; !ok {
		t.Fatal("cannot find the live type, Live_AryaKill")
	}
	if _, ok := mgr.Vault.DataBag["arya:OnInit:called"]; !ok {
		t.Fatal("OnInit is not invoked")
	}
}

func TestPluginManager_loadPlugins_two(t *testing.T) {
	pluginNames := []string{"arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "two", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, log); err != nil {
		t.Fatal(err)
	}
	if len(mgr.pluginMap) != len(pluginNames) {
		t.Fatal("len(mgr.pluginMap) != len(pluginNames)")
	}
	invariants(t, mgr)
}

func TestPluginManager_loadPlugins_reload(t *testing.T) {
	pluginNames := []string{"arya"}
	outputDir := preparePluginGroup(t, nil, "reload", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err != nil {
		t.Fatal(err)
	}
	invariants(t, mgr)

	buildArgs := []string{"--", "-ldflags", "-X main.CompileTimeString=stark"}
	preparePlugins(t, buildArgs, outputDir, pluginNames...)
	newMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := newMgr.loadPlugins(files, mgr, nil); err != nil {
		t.Fatal(err)
	}
	if len(newMgr.pluginMap) != 1 {
		t.Fatal("there should still be 1 plugin")
	}
	for _, p := range newMgr.pluginMap {
		var compileTimeString string
		if err := p.Lookup("CompileTimeString", &compileTimeString); err != nil {
			t.Fatal(err)
		} else if compileTimeString != "stark" {
			t.Fatal("the plugin was not properly reloaded")
		} else if p.unchanged {
			t.Fatal("p.unchanged should be false")
		}
	}
	invariants(t, newMgr)

	stammerMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := stammerMgr.loadPlugins(files, newMgr, nil); err != nil {
		t.Fatal(err)
	}
	if len(stammerMgr.pluginMap) != 1 {
		t.Fatal("there should still be 1 plugin")
	}
	for _, p := range stammerMgr.pluginMap {
		if p.Note != "unchanged" {
			t.Fatal(`p.Note != "unchanged"`)
		} else if !p.unchanged {
			t.Fatal("p.unchanged should be true")
		}
	}
	invariants(t, stammerMgr)

	preparePlugins(t, nil, outputDir, "snow")
	withBrotherMgr := newPluginManager(log, nilNewer)
	withBrotherFiles := completePluginPaths(outputDir, "arya", "snow")
	prepareEnv(t, "")
	if err := withBrotherMgr.loadPlugins(withBrotherFiles, stammerMgr, log); err != nil {
		t.Fatal(err)
	}
	if len(withBrotherMgr.pluginMap) != 2 {
		t.Fatal("there should be 2 plugins now")
	}
	for _, p := range withBrotherMgr.pluginMap {
		switch p.Name {
		case "arya":
			if p.Note != "unchanged" {
				t.Fatal(`p.Note != "unchanged"`)
			} else if !p.unchanged {
				t.Fatal("p.unchanged should be true")
			}
		case "snow":
			if p.unchanged {
				t.Fatal("p.unchanged should be false")
			}
		default:
			panic("impossible")
		}
	}
	invariants(t, withBrotherMgr)

	aloneMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := aloneMgr.loadPlugins(files, withBrotherMgr, nil); err != nil {
		t.Fatal(err)
	}
	if len(aloneMgr.pluginMap) != 1 {
		t.Fatal("there should be 1 plugin again")
	}
	for _, p := range aloneMgr.pluginMap {
		if p.Note != "unchanged" {
			t.Fatal(`p.Note != "unchanged"`)
		} else if !p.unchanged {
			t.Fatal("p.unchanged should be true")
		}
	}
	invariants(t, aloneMgr)

	arya := aloneMgr.FindPlugin("arya")
	if arya.Refs.Load() != 4 {
		t.Fatal("arya.Refs.Load() != 4")
	}
	aloneMgr.invokeEveryOnFree()
	if arya.Refs.Load() != 3 {
		t.Fatal("arya.Refs.Load() != 3")
	}
}

func TestPluginManager_loadPlugins_stubborn(t *testing.T) {
	pluginNames := []string{"stubborn"}
	outputDir := preparePluginGroup(t, nil, "stubborn", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err != nil {
		t.Fatal(err)
	}
	invariants(t, mgr)
	if mgr.FindPlugin("stubborn").Note != "" {
		t.Fatal(`mgr.FindPlugin("stubborn").Note != ""`)
	}

	buildArgs := []string{"--", "-ldflags", "-X main.CompileTimeString=dogged"}
	preparePlugins(t, buildArgs, outputDir, pluginNames...)
	newMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := newMgr.loadPlugins(files, mgr, nil); err != nil {
		t.Fatal(err)
	}
	if len(newMgr.pluginMap) != 1 {
		t.Fatal("there should still be 1 plugin")
	}
	for _, p := range newMgr.pluginMap {
		if p.Note != "not reloadable" {
			t.Fatal(`p.Note != "not reloadable"`)
		} else if !p.unchanged {
			t.Fatal("p.unchanged should be true")
		} else if p.Refs.Load() != 2 {
			t.Fatal("p.Refs.Load() != 2")
		}
	}
	invariants(t, newMgr)
}

func validatePluginOrder(t *testing.T, mgr *PluginManager, pluginNames ...string) error {
	t.Helper()
	var a []int
outer:
	for _, pName := range pluginNames {
		for i, p := range mgr.ordered {
			if p.Name == pName {
				a = append(a, i)
				continue outer
			}
		}
		return fmt.Errorf("cannot find the plugin %s", pName)
	}
	for i := 0; i < len(a)-1; i++ {
		if a[i] >= a[i+1] {
			return errors.New("mgr.ordered does not work as expected")
		}
	}
	return nil
}

func TestPluginManager_loadPlugins_importall(t *testing.T) {
	pluginNames := []string{"importall", "arya", "snow", "stubborn"}
	outputDir := preparePluginGroup(t, nil, "importall", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, log); err != nil {
		t.Fatal(err)
	}
	if len(mgr.pluginMap) != len(pluginNames) {
		t.Fatal("len(mgr.pluginMap) != len(pluginNames)")
	}
	if err := validatePluginOrder(t, mgr, "arya", "importall"); err != nil {
		t.Fatal(err)
	}
	if err := validatePluginOrder(t, mgr, "snow", "importall"); err != nil {
		t.Fatal(err)
	}
	invariants(t, mgr)

	p := mgr.pluginMap["importall"]
	if p == nil {
		t.Fatal("cannot find 'importall' in mgr.pluginMap")
	}
	if len(p.Deps) != 2 {
		t.Fatal("len(p.Deps) != 2")
	}
	if p.Deps[0] != "arya" && p.Deps[1] != "arya" {
		t.Fatal("arya is not in importall.Deps")
	}
	if p.Deps[0] != "snow" && p.Deps[1] != "snow" {
		t.Fatal("snow is not in importall.Deps")
	}
	exported, ok := p.exported.(importall.Export)
	if !ok {
		t.Fatal("the value returned by importall.Export() is unexpected")
	}
	if err := exported.TestDeps(); err != nil {
		t.Fatal("the import mechanism does not work as expected")
	}

	m := make(map[string]int)
	resetOrderMap := func() {
		m["importall"] = -1
		m["arya"] = -1
		m["snow"] = -1
	}

	resetOrderMap()
	for k := range m {
		str := fmt.Sprintf("invoking %s.OnLoad", k)
		_, i, ok := log.Find(str)
		if !ok {
			t.Fatalf("cannot find %q in the log", str)
		}
		m[k] = i
	}
	if m["importall"] <= m["arya"] {
		t.Fatal(`m["importall"] <= m["arya"]`)
	}
	if m["importall"] <= m["snow"] {
		t.Fatal(`m["importall"] <= m["snow"]`)
	}

	resetOrderMap()
	for k := range m {
		str := fmt.Sprintf("invoking %s.OnInit", k)
		_, i, ok := log.Find(str)
		if !ok {
			t.Fatalf("cannot find %q in the log", str)
		}
		m[k] = i
	}
	if m["importall"] <= m["arya"] {
		t.Fatal(`m["importall"] <= m["arya"]`)
	}
	if m["importall"] <= m["snow"] {
		t.Fatal(`m["importall"] <= m["snow"]`)
	}

	mgr.invokeEveryOnFree()
	resetOrderMap()
	for k := range m {
		str := fmt.Sprintf("invoking %s.OnFree", k)
		_, i, ok := log.Find(str)
		if !ok {
			t.Fatalf("cannot find %q in the log", str)
		}
		m[k] = i
	}
	if m["importall"] >= m["arya"] {
		t.Fatal(`m["importall"] >= m["arya"]`)
	}
	if m["importall"] >= m["snow"] {
		t.Fatal(`m["importall"] >= m["snow"]`)
	}

	log.Reset()
	newMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := newMgr.loadPlugins(files, mgr, log); err != nil {
		t.Fatal(err)
	}
	if len(newMgr.pluginMap) != len(pluginNames) {
		t.Fatal("len(newMgr.pluginMap) != len(pluginNames)")
	}
	s1 := "not reloadable: [stubborn]"
	if _, _, ok := log.Find(s1); !ok {
		t.Fatalf("cannot find %q in the log", s1)
	}
	s2 := "to be loaded: []"
	if _, _, ok := log.Find(s2); !ok {
		t.Fatalf("cannot find %q in the log", s2)
	}
	invariants(t, newMgr)
}

func TestPluginManager_loadPlugins_mismatch1(t *testing.T) {
	pluginNames := []string{"mismatch1", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "mismatch1", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, log); err == nil {
		t.Fatal(errors.New("loadPlugins should fail here"))
	} else if !strings.Contains(err.Error(), "is not assignable") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_mismatch2(t *testing.T) {
	pluginNames := []string{"mismatch2"}
	outputDir := preparePluginGroup(t, nil, "mismatch2", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal(errors.New("loadPlugins should fail here"))
	} else if !strings.Contains(err.Error(), "failed to assign") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_orderPlugins(t *testing.T) {
	pluginNames := []string{"xdep", "importall", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "orderPlugins", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:stark")
	if err := mgr.loadPlugins(files, nil, log); err != nil {
		t.Fatal(err)
	}
	if err := validatePluginOrder(t, mgr, "arya", "importall"); err != nil {
		t.Fatal(err)
	}
	if err := validatePluginOrder(t, mgr, "snow", "importall"); err != nil {
		t.Fatal(err)
	}
	if err := validatePluginOrder(t, mgr, "xdep", "importall"); err != nil {
		t.Fatal(err)
	}
	invariants(t, mgr)
	checkImportValue(t, mgr, "xdep", "&fxStark")
	checkImportValue(t, mgr, "importall", "&fxStark")
}

func TestPluginManager_loadPlugins_xdep_ignore(t *testing.T) {
	pluginNames := []string{"xdep", "importall", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "xdep_ignore", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:ignore")
	if err := mgr.loadPlugins(files, nil, log); err != nil {
		t.Fatal(err)
	}
	if err := validatePluginOrder(t, mgr, "arya", "importall", "xdep"); err != nil {
		t.Fatal(err)
	}
	if err := validatePluginOrder(t, mgr, "snow", "importall", "xdep"); err != nil {
		t.Fatal(err)
	}
	invariants(t, mgr)
	checkImportValue(t, mgr, "xdep", "&fxIgnore")
}

func TestPluginManager_loadPlugins_xdep_mini1(t *testing.T) {
	pluginNames := []string{"xdep", "importall", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "xdep_mini1", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, log); err != nil {
		t.Fatal(err)
	}
	invariants(t, mgr)
	checkImportValue(t, mgr, "xdep", "&fx")

	newPluginNames := []string{"xdep", "arya"}
	preparePlugins(t, nil, outputDir, []string{"xdep"}...)
	newFiles := completePluginPaths(outputDir, newPluginNames...)

	newMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:mini")
	if err := newMgr.loadPlugins(newFiles, mgr, nil); err != nil {
		t.Fatal(err)
	}
	if len(newMgr.pluginMap) != 2 {
		t.Fatal("there should be 2 plugins now")
	}
	for _, p := range newMgr.pluginMap {
		switch p.Name {
		case "xdep":
			if p.unchanged {
				t.Fatal(p.Name + ": p.unchanged should be false")
			}
		case "arya":
			if p.Note != "unchanged" {
				t.Fatal(`p.Note != "unchanged"`)
			} else if !p.unchanged {
				t.Fatal("p.unchanged should be true")
			}
		default:
			panic("impossible")
		}
	}
	invariants(t, newMgr)
	checkImportValue(t, newMgr, "xdep", "&fxMini")
}

func TestPluginManager_loadPlugins_xdep_mini2(t *testing.T) {
	pluginNames := []string{"xdep", "importall", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "xdep_mini2", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, log); err != nil {
		t.Fatal(err)
	}
	invariants(t, mgr)
	checkImportValue(t, mgr, "xdep", "&fx")
	checkImportValue(t, mgr, "importall", "&fx")

	newPluginNames := []string{"xdep", "importall", "arya"}
	preparePlugins(t, nil, outputDir, []string{"importall"}...)
	newFiles := completePluginPaths(outputDir, newPluginNames...)

	newMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:mini")
	if err := newMgr.loadPlugins(newFiles, mgr, nil); err == nil {
		t.Fatal("loadPlugins should fail when a plugin does not change while one of its dependencies changes")
	} else if !strings.Contains(err.Error(), "was rebuilt while") {
		t.Fatal("unexpected error: " + err.Error())
	}

	preparePlugins(t, nil, outputDir, newPluginNames...)
	properMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:mini")
	if err := properMgr.loadPlugins(newFiles, mgr, nil); err != nil {
		t.Fatal(err)
	}
	if len(properMgr.pluginMap) != 3 {
		t.Fatal("there should be 3 plugins now")
	}
	invariants(t, properMgr)

	depNoneNames := []string{"xdep"}
	preparePlugins(t, nil, outputDir, depNoneNames...)
	depNoneFiles := completePluginPaths(outputDir, depNoneNames...)

	depNoneMgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:none")
	if err := depNoneMgr.loadPlugins(depNoneFiles, properMgr, nil); err != nil {
		t.Fatal(err)
	}
	invariants(t, depNoneMgr)
	checkImportValue(t, depNoneMgr, "xdep", "&struct{}{}")
}

func TestPluginManager_loadPlugins_xdep_e1(t *testing.T) {
	pluginNames := []string{"xdep", "importall"}
	outputDir := preparePluginGroup(t, nil, "xdep_e1", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when a dependency is missing")
	} else if !strings.Contains(err.Error(), "unknown dependency: Arya. plugin: importall") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_xdep_e2(t *testing.T) {
	pluginNames := []string{"xdep", "importall", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "xdep_e2", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:import-returns-obj")
	if err := mgr.loadPlugins(files, nil, log); err == nil {
		t.Fatal("loadPlugins should fail when Import() returns an object")
	} else if !strings.Contains(err.Error(), "Import() must be a pointer") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_xdep_e3(t *testing.T) {
	pluginNames := []string{"xdep", "importall", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "xdep_e3", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:unknown")
	if err := mgr.loadPlugins(files, nil, log); err == nil {
		t.Fatal("loadPlugins should fail when a field of Import() has a typo")
	} else if !strings.Contains(err.Error(), "unknown dependency: Xtypo2. plugin: xdep") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_xdep_e4(t *testing.T) {
	pluginNames := []string{"xdep", "importall", "arya", "snow"}
	outputDir := preparePluginGroup(t, nil, "xdep_e4", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:not-reloadable")
	if err := mgr.loadPlugins(files, nil, log); err == nil {
		t.Fatal("loadPlugins should fail when a dependency of a plugin is not reloadable while the plugin is")
	} else if !strings.Contains(err.Error(), "is NOT reloadable while its dependency") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_xdep_e5(t *testing.T) {
	pluginNames := []string{"xdep", "arya"}
	outputDir := preparePluginGroup(t, nil, "xdep_e5", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "xdep:nameless")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when any field of Import() is anonymous")
	} else if !strings.Contains(err.Error(), "field of the Import() object cannot be anonymous") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_cyclic2(t *testing.T) {
	pluginNames := []string{"cyclic1", "cyclic2"}
	outputDir := preparePluginGroup(t, nil, "cyclic2", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "cyclic dependency detected") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_loadPlugins_cyclic3(t *testing.T) {
	pluginNames := []string{"cyclic1", "cyclic2", "cyclic3"}
	outputDir := preparePluginGroup(t, nil, "cyclic3", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "cyclic2:fx3")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "cyclic dependency detected") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func panics_OnOpen(log slog.Logger) func(*Plugin, interface{}) {
	return func(p *Plugin, data interface{}) {
		var setLogger func(slog.Logger)
		if err := p.Lookup("SetLogger", &setLogger); err != nil {
			panic(err)
		} else {
			setLogger(log)
		}
	}
}

func panics_checkLog(t *testing.T, log *slog.Scavenger, env string, pluginNames ...string) {
	t.Helper()
	var a []int
	x := strings.TrimPrefix(env, "panics:")
	for _, pName := range pluginNames {
		str := fmt.Sprintf("<%s> %s", pName, x)
		_, idx, ok := log.Find(str)
		if !ok {
			t.Fatal("cannot find the following message in log: " + str)
		}
		a = append(a, idx)
	}
	switch len(a) {
	case 1:
		x := "> " + x
		if _, ok := log.FindStringSequence([]string{x, x}); ok {
			t.Fatal("messages are both found")
		}
	case 2:
		if idx1, idx2 := a[0], a[1]; idx1 >= idx2 {
			t.Fatal("idx1 >= idx2")
		}
	default:
		panic("impossible")
	}
}

func panics_checkInvokeEveryOnFree(t *testing.T, log *slog.Scavenger, random bool, pluginNames ...string) {
	t.Helper()
	var a []int
	for _, pName := range pluginNames {
		str := fmt.Sprintf("<hotswap> invoking %s.OnFree", pName)
		_, idx, ok := log.Find(str)
		if !ok {
			t.Fatal("cannot find the following message in log: " + str)
		}
		a = append(a, idx)
	}
	if !random {
		for i := 0; i < len(a)-1; i++ {
			if a[i] >= a[i+1] {
				t.Fatal("a[i] >= a[i+1]")
			}
		}
	}
}

func TestPluginManager_loadPlugins_panics_OnLoad(t *testing.T) {
	pluginNames := []string{"importall", "panics"}
	outputDir := preparePluginGroup(t, nil, "panics_OnLoad", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	mgr.cbOpen = panics_OnOpen(log)
	const env = "panics:OnLoad"
	prepareEnv(t, env)
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when panic happens")
	} else if !strings.Contains(err.Error(), "<hotswap:panics> panic: "+env) {
		t.Fatal("unexpected error: " + err.Error())
	}
	panics_checkLog(t, log, env, "panics")
	panics_checkInvokeEveryOnFree(t, log, false, pluginNames...)
}

func TestPluginManager_loadPlugins_panics_OnInit(t *testing.T) {
	pluginNames := []string{"importall", "panics"}
	outputDir := preparePluginGroup(t, nil, "panics_OnInit", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	mgr.cbOpen = panics_OnOpen(log)
	const env = "panics:OnInit"
	prepareEnv(t, env)
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when panic happens")
	} else if !strings.Contains(err.Error(), "<hotswap:panics> panic: "+env) {
		t.Fatal("unexpected error: " + err.Error())
	}
	panics_checkLog(t, log, env, "panics")
	panics_checkInvokeEveryOnFree(t, log, false, pluginNames...)
}

func TestPluginManager_loadPlugins_panics_OnFree(t *testing.T) {
	pluginNames := []string{"importall", "panics"}
	outputDir := preparePluginGroup(t, nil, "panics_OnFree", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	mgr.cbOpen = panics_OnOpen(log)
	const env = "panics:OnFree"
	prepareEnv(t, env)
	if err := mgr.loadPlugins(files, nil, nil); err != nil {
		t.Fatal(err)
	}
	mgr.invokeEveryOnFree()
	panics_checkLog(t, log, env, pluginNames...)
}

func TestPluginManager_loadPlugins_panics_Export(t *testing.T) {
	pluginNames := []string{"importall", "panics"}
	outputDir := preparePluginGroup(t, nil, "panics_Export", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	mgr.cbOpen = panics_OnOpen(log)
	const env = "panics:Export"
	prepareEnv(t, env)
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when panic happens")
	} else if !strings.Contains(err.Error(), "<hotswap:panics> panic: "+env) {
		t.Fatal("unexpected error: " + err.Error())
	}
	panics_checkLog(t, log, env, pluginNames...)
}

func TestPluginManager_loadPlugins_panics_Import(t *testing.T) {
	pluginNames := []string{"importall", "panics"}
	outputDir := preparePluginGroup(t, nil, "panics_Import", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	mgr.cbOpen = panics_OnOpen(log)
	const env = "panics:Import"
	prepareEnv(t, env)
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when panic happens")
	} else if !strings.Contains(err.Error(), "<hotswap:panics> panic: "+env) {
		t.Fatal("unexpected error: " + err.Error())
	}
	panics_checkLog(t, log, env, pluginNames...)
}

func TestPluginManager_loadPlugins_panics_Reloadable(t *testing.T) {
	pluginNames := []string{"importall", "panics"}
	outputDir := preparePluginGroup(t, nil, "panics_Reloadable", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	mgr.cbOpen = panics_OnOpen(log)
	const env = "panics:Reloadable"
	prepareEnv(t, env)
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail when panic happens")
	} else if !strings.Contains(err.Error(), "<hotswap:panics> panic: "+env) {
		t.Fatal("unexpected error: " + err.Error())
	}
	panics_checkLog(t, log, env, pluginNames...)
}

func checkImportValue(t *testing.T, mgr *PluginManager, name string, expected string) {
	t.Helper()
	p := mgr.FindPlugin(name)
	ret, err := p.InvokeFunc("fxWhich")
	if err != nil {
		t.Fatal(err)
	}
	fxWhich, ok := ret.(string)
	if !ok {
		t.Fatal("failed to get the value of fxWhich")
	}
	if fxWhich != expected {
		t.Fatal("fxWhich != expected", fxWhich, expected)
	}
}

func TestPluginManager_loadPlugins_panics_InvokeFunc(t *testing.T) {
	pluginNames := []string{"importall", "panics"}
	outputDir := preparePluginGroup(t, nil, "panics_InvokeFunc", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	mgr.cbOpen = panics_OnOpen(log)
	const env = "panics:InvokeFunc"
	prepareEnv(t, env)
	if err := mgr.loadPlugins(files, nil, nil); err != nil {
		t.Fatal(err)
	}
	mgr.InvokeEach("foo1")
	panics_checkLog(t, log, env, "panics", "importall")
	checkImportValue(t, mgr, "importall", "&fxPanic")

	log.Reset()
	mgr.InvokeEachBackward("foo2")
	panics_checkLog(t, log, env, pluginNames...)

	log.Reset()
	prepareEnv(t, "")
	mgr.InvokeEach("bar1")
	panics_checkLog(t, log, env, "panics", "importall")

	seq := []string{
		"<panics> unreasonable error",
		"<importall> unreasonable error",
	}
	if _, ok := log.FindStringSequence(seq); !ok {
		t.Fatal("cannot find the error message sequence")
	}

	log.Reset()
	mgr.InvokeEachBackward("bar2")
	panics_checkLog(t, log, env, pluginNames...)

	seq[0], seq[1] = seq[1], seq[0]
	if _, ok := log.FindStringSequence(seq); !ok {
		t.Fatal("cannot find the error message sequence")
	}
}

func TestPluginManager_missing1(t *testing.T) {
	pluginNames := []string{"fns1"}
	outputDir := preparePluginGroup(t, nil, "missing1", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "missing functions:") || strings.Count(err.Error(), ",") != 6 {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_missing2(t *testing.T) {
	pluginNames := []string{"fns2"}
	outputDir := preparePluginGroup(t, nil, "missing2", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "missing functions:") || strings.Count(err.Error(), ",") != 2 {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_duplicateLiveFuncName(t *testing.T) {
	pluginNames := []string{"arya", "shadow1"}
	outputDir := preparePluginGroup(t, nil, "duplicateLiveFuncName", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "duplicate live function name") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_duplicateLiveTypeName(t *testing.T) {
	pluginNames := []string{"snow", "shadow2"}
	outputDir := preparePluginGroup(t, nil, "duplicateLiveTypeName", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	prepareEnv(t, "")
	if err := mgr.loadPlugins(files, nil, log); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "duplicate live type name") {
		t.Fatal("unexpected error: " + err.Error())
	}
}

func TestPluginManager_panicTrigger1(t *testing.T) {
	pluginNames := []string{"xdep", "arya"}
	outputDir := preparePluginGroup(t, nil, "panicTrigger1", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	countdown := 1
	mgr.panicTrigger = func(data interface{}) {
		if countdown--; countdown == 0 {
			panic(100)
		}
	}
	prepareEnv(t, "xdep:mini")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "<hotswap> panic:") {
		t.Fatal("unexpected error: " + err.Error())
	}

	panics_checkInvokeEveryOnFree(t, log, true, pluginNames...)
}

func TestPluginManager_panicTrigger2(t *testing.T) {
	pluginNames := []string{"xdep", "arya"}
	outputDir := preparePluginGroup(t, nil, "panicTrigger2", pluginNames...)
	files := completePluginPaths(outputDir, pluginNames...)

	log := newScavenger()
	mgr := newPluginManager(log, nilNewer)
	countdown := 2
	mgr.panicTrigger = func(data interface{}) {
		if countdown--; countdown == 0 {
			panic(200)
		}
	}
	prepareEnv(t, "xdep:mini")
	if err := mgr.loadPlugins(files, nil, nil); err == nil {
		t.Fatal("loadPlugins should fail here")
	} else if !strings.Contains(err.Error(), "<hotswap> panic:") {
		t.Fatal("unexpected error: " + err.Error())
	}

	panics_checkInvokeEveryOnFree(t, log, false, pluginNames...)
}
