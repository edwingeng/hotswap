package hotswap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edwingeng/hotswap"
	"github.com/edwingeng/hotswap/cli/hotswap/trial"
	"github.com/edwingeng/hotswap/internal/hutils"
	"github.com/edwingeng/slog"
)

func prepareEnv(t *testing.T, env string) {
	t.Helper()
	if err := os.Setenv("pluginTest", env); err != nil {
		t.Fatal(err)
	}
}

func prepareStaticPlugins(t *testing.T, pluginNames ...string) {
	t.Helper()
	const exe = "cli/hotswap/hotswap"
	const homeDir = "cli/hotswap/trial"
	for _, pName := range pluginNames {
		pluginDir := filepath.Join(homeDir, pName)
		hutils.BuildPlugin(t, exe, pluginDir, homeDir, "--staticLinking")
	}
}

func TestWithStaticPlugins(t *testing.T) {
	pluginNames := []string{"arya", "snow", "stubborn"}
	prepareStaticPlugins(t, pluginNames...)

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
