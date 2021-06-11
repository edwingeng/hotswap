package hotswap

import (
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"testing"

	"github.com/edwingeng/hotswap/internal/hutils"
	"github.com/edwingeng/hotswap/vault"
)

func TestPlugin(t *testing.T) {
	const exe = "cli/hotswap/hotswap"
	const pluginDir = "cli/hotswap/trial/arya"
	const outputDir = pluginDir
	hutils.BuildPlugin(t, exe, pluginDir, outputDir)
	so := filepath.Join(outputDir, "arya.so")
	if _, err := os.Stat(so); err != nil {
		t.Fatal(err)
	}

	plug, err := plugin.Open(so)
	if err != nil {
		t.Fatal(err)
	}

	p := newPlugin()
	p.Name = pluginName(so)
	p.P = plug

	m := map[string]interface{}{
		"OnLoad":           &p.fOnLoad,
		"OnInit":           &p.fOnInit,
		"OnFree":           &p.fOnFree,
		"Export":           &p.fExport,
		"Import":           &p.fImport,
		"InvokeFunc":       &p.InvokeFunc,
		"Reloadable":       &p.fReloadable,
		"HotswapLiveFuncs": &p.hotswapLiveFuncs,
		"HotswapLiveTypes": &p.hotswapLiveTypes,
	}

	for k, v := range m {
		if err := p.Lookup(k, v); err != nil {
			t.Fatalf("Lookup() does not work as expected [1]. k: %s, err: %+v", k, err)
		}
		if isNil(v) {
			t.Fatalf("Lookup() does not work as expected [2]. k: %s", k)
		}
	}

	ptr := &PluginFuncs{}
	if err := p.Lookup("", ptr); err == nil {
		t.Fatalf("Lookup() should fail when symName is empty")
	}
	if err := p.Lookup("alpha", nil); err == nil {
		t.Fatalf("Lookup() should fail when out is nil")
	}
	if err := p.Lookup("alpha", ""); err == nil {
		t.Fatalf("Lookup() should fail when out is not a pointer")
	}
	if err := p.Lookup("alpha", ptr); err != ErrNotExist {
		t.Fatalf("Lookup() should return ErrNotExist when symName is not found. err: %v", err)
	}

	var finalBlow1 int
	if err := p.Lookup("FinalBlow1", &finalBlow1); err != nil {
		t.Fatalf(`FinalBlow1. err: %v`, err)
	} else if finalBlow1 != 100 {
		t.Fatalf("finalBlow1 != 100")
	}

	var finalBlow2 *int
	if err := p.Lookup("FinalBlow2", &finalBlow2); err != nil {
		t.Fatalf(`FinalBlow2. err: %v`, err)
	} else if *finalBlow2 != 100 {
		t.Fatalf("*finalBlow2 != 100")
	}

	var finalBlow3 int64
	if err := p.Lookup("FinalBlow3", &finalBlow3); err == nil {
		t.Fatalf("FinalBlow3")
	}

	// Func: OnLoad
	if err := p.fOnLoad(nil); err != nil {
		t.Fatalf("OnLoad() should not return an error. err: %v", err)
	}

	// Func: OnInit
	vault1 := &vault.Vault{
		LiveFuncs: make(map[string]interface{}),
		LiveTypes: make(map[string]func() interface{}),
		DataBag:   make(map[string]interface{}),
	}
	if err := p.fOnInit(vault1); err != nil {
		t.Fatalf("OnInit() should not return an error. err: %v", err)
	}

	// Func: OnFree
	p.fOnFree()

	// Func: Export
	if ret := p.fExport(); isNil(ret) {
		t.Fatalf("Export() should not return nil")
	}

	// Func: Import
	if ret := p.fImport(); ret != nil {
		t.Fatalf("Import() should return nil")
	}

	// Func: InvokeFunc
	ret, err := p.InvokeFunc("polish", 1)
	if err != nil || reflect.ValueOf(ret).Kind() != reflect.String || !strings.HasPrefix(ret.(string), "Polished") {
		t.Fatalf("unexpected return value of InvokeFunc(). ret: %v, err: %v", ret, err)
	}

	// Func: Reloadable
	if reloadable := p.fReloadable(); !reloadable {
		t.Fatalf("Reloadable() should return true")
	}

	// Func: HotswapLiveFuncs
	if p.hotswapLiveFuncs() == nil {
		t.Fatalf("p.hotswapLiveFuncs() == nil")
	}
	if len(p.hotswapLiveFuncs()) != 2 {
		t.Fatalf("len(p.hotswapLiveFuncs()) != 2")
	}
	for k, v := range p.hotswapLiveFuncs() {
		if k != "live_NotToday" && k != "Live_Anyone" {
			t.Fatalf("unexpected live function: %s", k)
		}
		if reflect.TypeOf(v).Kind() != reflect.Func {
			t.Fatalf("reflect.TypeOf(v).Kind() != reflect.Func")
		}
	}

	// Func: HotswapLiveTypes
	if p.hotswapLiveTypes() == nil {
		t.Fatalf("p.hotswapLiveTypes() == nil")
	}
	if len(p.hotswapLiveTypes()) != 1 {
		t.Fatalf("len(p.hotswapLiveTypes()) != 1")
	}
	for k, v := range p.hotswapLiveTypes() {
		if k != "Live_AryaKill" {
			t.Fatalf("unexpected live type: %s", k)
		}
		if reflect.TypeOf(v).Kind() != reflect.Func {
			t.Fatalf("reflect.TypeOf(v).Kind() != reflect.Func")
		}
		if v == nil || v() == nil {
			t.Fatalf("v == nil || v() == nil")
		}
	}
}
