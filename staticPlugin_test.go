package hotswap_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/edwingeng/hotswap"
	"github.com/edwingeng/hotswap/cli/hotswap/trial"
	"github.com/edwingeng/slog"
)

func prepareEnv(t *testing.T, env string) {
	t.Helper()
	if err := os.Setenv("pluginTest", env); err != nil {
		t.Fatal(err)
	}
}

func TestWithStaticPlugins(t *testing.T) {
	pluginNames := []string{"arya", "snow", "stubborn"}
	for _, pName := range pluginNames {
		file := filepath.Join("cli/hotswap/trial", fmt.Sprintf("hotswap.staticPluginInit.%s.go", pName))
		if _, err := os.Stat(file); err != nil {
			t.SkipNow()
		}
	}

	log := slog.NewScavenger()
	swapper := hotswap.NewPluginManagerSwapper("",
		hotswap.WithLogger(log),
		hotswap.WithStaticPlugins(trial.HotswapStaticPlugins),
	)
	prepareEnv(t, "")
	details1, err := swapper.LoadPlugins(log)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range details1 {
		if v != "ok" {
			t.Fatalf(`unexpected result. file: %s, result: %s`, k, v)
		}
	}
	if swapper.Current() == nil {
		t.Fatal("swapper.Current() == nil")
	}
	currentPlugins := swapper.Current().Plugins()
	if len(currentPlugins) != 3 {
		t.Fatal("len(currentPlugins) != 3")
	}
	for _, p := range currentPlugins {
		switch p.Name {
		case "arya", "snow", "stubborn":
		default:
			panic("impossible")
		}
	}
	if len(swapper.Current().LiveFuncs) != 2 {
		t.Fatal("len(swapper.Current().LiveFuncs) != 2")
	}
	if len(swapper.Current().LiveTypes) != 2 {
		t.Fatal("len(swapper.Current().LiveTypes) != 2")
	}
}
