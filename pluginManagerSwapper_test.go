package hotswap

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/edwingeng/hotswap/internal/hutils"
)

func newSwapper(pluginDir string, opts ...Option) *PluginManagerSwapper {
	n := len(opts)
	opts = append(opts[:n:n], WithFreeDelay(time.Second), WithExtensionNewer(nilNewer))
	return NewPluginManagerSwapper(pluginDir, opts...)
}

func TestPluginManagerSwapper_LoadPlugins(t *testing.T) {
	oldMinFreeDelay := minFreeDelay
	minFreeDelay = time.Second
	defer func() {
		minFreeDelay = oldMinFreeDelay
	}()

	log := newScavenger()
	swapper1 := newSwapper("/helloXYZ", WithLogger(log))
	if _, err := swapper1.LoadPlugins(log); err == nil {
		t.Fatal("LoadPlugins should fail when the plugin directory does not exist")
	} else if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := swapper1.Reload(log); err == nil {
		t.Fatal("Reload should fail when it is called before a successful call of LoadPlugins")
	} else if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("unexpected error: %v", err)
	}
	if swapper1.Current() != nil {
		t.Fatal("swapper1.Current() != nil")
	}

	pluginNames := []string{"arya", "snow", "stubborn"}
	outputDir := preparePluginGroup(t, nil, "LoadPlugins", pluginNames...)

	swapper2 := newSwapper(outputDir, WithLogger(log))
	prepareEnv(t, "")
	details1, err := swapper2.LoadPlugins(log)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range details1 {
		if v != "ok" {
			t.Fatalf(`unexpected result. file: %s, result: %s`, k, v)
		}
	}
	if swapper2.Current() == nil {
		t.Fatal("swapper2.Current() == nil")
	}
	currentPlugins1 := swapper2.Current().Plugins()
	if len(currentPlugins1) != 3 {
		t.Fatal("len(currentPlugins1) != 3")
	}
	for _, p := range currentPlugins1 {
		compileTimeString := ""
		if err := p.Lookup("CompileTimeString", &compileTimeString); err != nil {
			t.Fatal(err)
		} else if compileTimeString != "" {
			t.Fatal("unexpected CompileTimeString: " + compileTimeString)
		}
		switch p.Name {
		case "arya", "snow", "stubborn":
		default:
			panic("impossible")
		}
	}

	log.Reset()
	oldMgr := swapper2.Current()
	newPluginNames := []string{"snow"}
	buildArgs := []string{"--", "-ldflags", "-X main.CompileTimeString=stark"}
	preparePluginGroupImpl(t, buildArgs, "LoadPlugins", false, newPluginNames...)
	details2, err := swapper2.Reload(log)
	if err != nil {
		t.Fatal(err)
	} else if _, _, ok := log.FindString("invoking snow.OnFree"); ok {
		t.Fatal("OnFree should not run this early")
	}
	for k, v := range details2 {
		switch name2key(pluginName(k)) {
		case "arya":
			if v != "unchanged" {
				t.Fatalf(`unexpected result. file: %s, result: %s`, k, v)
			}
		case "snow":
			if v != "ok" {
				t.Fatalf(`unexpected result. file: %s, result: %s`, k, v)
			}
		case "stubborn":
			if v != "not reloadable" {
				t.Fatalf(`unexpected result. file: %s, result: %s`, k, v)
			}
		default:
			panic("impossible")
		}
	}
	if swapper2.Current() == oldMgr {
		t.Fatal("swapper2.Current() == oldMgr")
	}
	if swapper2.Current() == nil {
		t.Fatal("swapper2.Current() == nil")
	}
	currentPlugins2 := swapper2.Current().Plugins()
	if len(currentPlugins2) != 3 {
		t.Fatal("len(currentPlugins2) != 3")
	}
	for _, p := range currentPlugins2 {
		compileTimeString := ""
		if err := p.Lookup("CompileTimeString", &compileTimeString); err != nil {
			t.Fatal(err)
		}
		switch p.Name {
		case "arya", "stubborn":
			if compileTimeString != "" {
				t.Fatal("unexpected CompileTimeString: " + compileTimeString)
			}
		case "snow":
			if compileTimeString != "stark" {
				t.Fatal("unexpected CompileTimeString: " + compileTimeString)
			}
		default:
			panic("impossible")
		}
	}

	time.Sleep(time.Millisecond * 1200)
	for _, pName := range pluginNames {
		str := fmt.Sprintf("invoking %s.OnFree", pName)
		switch pName {
		case "arya", "stubborn":
			if _, _, ok := log.FindString(str); ok {
				t.Fatal("unexpected message: " + str)
			}
		case "snow":
			if _, _, ok := log.FindString(str); !ok {
				t.Fatal("snow.OnFree should have been invoked")
			}
		default:
			panic("impossible")
		}
	}
}

func TestPluginManagerSwapper_ReloadWithCallback(t *testing.T) {
	var counter int
	cb := func(newManager, oldManager *PluginManager) error {
		counter++
		switch counter {
		case 1:
			newManager.Info("Reloaded: 1")
		case 2:
			newManager.Info("Reloaded: 2")
		case 3:
			return fmt.Errorf("%s", "Reloaded: unreasonable error")
		case 4:
			panic("Reloaded: panic")
		default:
			newManager.Infof("Reloaded: %d", counter)
		}
		return nil
	}

	pluginNames := []string{"snow"}
	outputDir := preparePluginGroup(t, nil, "ReloadWithCallback", pluginNames...)

	log := newScavenger()
	swapper := newSwapper(outputDir, WithLogger(log), WithReloadCallback(cb))
	prepareEnv(t, "")
	_, err := swapper.LoadPlugins(log)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := log.FindString("Reloaded: 1"); !ok {
		t.Fatal("ReloadCallback does not work as expected")
	}

	log.Reset()
	preparePluginGroup(t, nil, "ReloadWithCallback", pluginNames...)
	if _, err := swapper.Reload(log); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := log.FindString("Reloaded: 2"); !ok {
		t.Fatal("ReloadCallback does not work as expected")
	}

	log.Reset()
	preparePluginGroup(t, nil, "ReloadWithCallback", pluginNames...)
	if _, err := swapper.Reload(log); err == nil {
		t.Fatal("Reload should fail when ReloadCallback returns an error")
	}
	if _, _, ok := log.FindString("invoking snow.OnFree"); !ok {
		t.Fatal("snow.OnFree should have been invoked")
	}

	log.Reset()
	preparePluginGroup(t, nil, "ReloadWithCallback", pluginNames...)
	if _, err := swapper.Reload(log); err == nil {
		t.Fatal("Reload should fail when ReloadCallback panics")
	}
	if _, _, ok := log.FindString("invoking snow.OnFree"); !ok {
		t.Fatal("snow.OnFree should have been invoked")
	}

	log.Reset()
	preparePluginGroup(t, nil, "ReloadWithCallback", pluginNames...)
	extra := func(newManager, oldManager *PluginManager) error {
		newManager.Info("Reloaded: done")
		return nil
	}
	if _, err := swapper.ReloadWithCallback(log, extra); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seq := []string{
		"Reloaded: 5",
		"Reloaded: done",
	}
	if _, ok := log.FindStringSequence(seq); !ok {
		t.Fatal("cannot find the error message sequence")
	}
}

func TestFormatDetails(t *testing.T) {
	keys := []string{"a", "b" + hutils.FileNameExt, "c" + hutils.FileNameExt}
	vals := []string{"not reloadable", "ok", "unchanged"}
	var expected string
	for i := 0; i < len(keys); i++ {
		expected = fmt.Sprintf("%s, %s: %s", expected,
			strings.TrimSuffix(keys[i], hutils.FileNameExt), vals[i])
	}
	expected = expected[2:]

	for i := 0; i < 100; i++ {
		for j := 0; j < len(keys); j++ {
			idx := rand.Intn(len(keys))
			keys[j], keys[idx] = keys[idx], keys[j]
			vals[j], vals[idx] = vals[idx], vals[j]
		}
		m := make(map[string]string)
		for j := 0; j < len(keys); j++ {
			m[keys[j]] = vals[j]
		}

		actual := Details(m).String()
		if actual != expected {
			t.Fatal("actual != expected")
		}
	}
}
