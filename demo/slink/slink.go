package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/edwingeng/hotswap"
	"github.com/edwingeng/hotswap/demo/slink/g"
	"github.com/edwingeng/hotswap/demo/slink/plugin"
	"github.com/edwingeng/hotswap/internal/hutils"
	"github.com/edwingeng/live"
	"github.com/edwingeng/tickque"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {
	var pluginDir string
	var pidFile string
	var signalFile string
	var staticLinking bool
	flag.StringVar(&pluginDir, "pluginDir", "", "the directory holding your plugins")
	flag.StringVar(&pidFile, "pidFile", "", "pid file path")
	flag.StringVar(&signalFile, "signalFile", "", "once the specified file is found on your disk, reload all plugins")
	flag.BoolVar(&staticLinking, "staticLinking", false, "link plugin statically (not reloadable)")
	flag.Parse()

	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		panic(err)
	}
	if err := hutils.FindDirectory(absDir, ""); err != nil && !staticLinking {
		panic(err)
	}
	if pidFile == "" {
		panic("no --pidFile")
	}
	if signalFile == "" {
		panic("no --signalFile")
	}

	pid := fmt.Sprint(os.Getpid())
	if err := ioutil.WriteFile(pidFile, []byte(pid), 0644); err != nil {
		panic(err)
	}

	opts := []hotswap.Option{
		hotswap.WithLogger(g.Logger),
		hotswap.WithExtensionNewer(g.NewVaultExtension),
	}
	if staticLinking {
		opts = append(opts, hotswap.WithStaticPlugins(plugin.HotswapStaticPlugins))
	}
	g.PluginManagerSwapper = hotswap.NewPluginManagerSwapper(absDir, opts...)
	swapper := g.PluginManagerSwapper
	details, err := swapper.LoadPlugins(nil)
	if err != nil {
		panic(err)
	} else if len(details) == 0 {
		panic("no plugin is found in " + absDir)
	} else {
		g.Logger.Infof("<hotswap> %d plugin(s) loaded. details: [%s]",
			len(details), details)
	}

	go func() {
		heartbeat()
		for range time.Tick(time.Second * 3) {
			heartbeat()
		}
	}()

	go func() {
		for range time.Tick(time.Millisecond * 50) {
			if _, err := os.Stat(signalFile); err != nil {
				continue
			}
			g.Logger.Info("<hotswap> reloading...")
			details, err := swapper.Reload(nil)
			if staticLinking {
				if err != nil {
					g.Logger.Errorf("<hotswap> %s", err)
					err = os.Remove(signalFile)
					if err != nil {
						break
					} else {
						continue
					}
				}
				panic("impossible")
			}
			if err != nil {
				panic(err)
			} else if len(details) == 0 {
				g.Logger.Infof("no plugin is found in " + absDir)
			} else {
				g.Logger.Infof("<hotswap> %d plugin(s) loaded. details: [%s]",
					len(details), details)
			}

			heartbeat()
			if err := os.Remove(signalFile); err != nil {
				g.Logger.Error(err)
				os.Exit(1)
			}
		}
	}()

	// Wait for signals
	chSignal := make(chan os.Signal, 1)
	signal.Notify(chSignal, syscall.SIGINT, syscall.SIGTERM)

loop:
	for {
		select {
		case sig := <-chSignal:
			g.Logger.Infof("signal received: %v", sig)
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				break loop
			}
		}
	}

	signal.Reset(syscall.SIGINT, syscall.SIGTERM)
	g.Logger.Info("THE END")
}

func heartbeat() {
	var job tickque.Job
	job.Type = "live_Woof"
	job.Data = live.WrapInt(rand.Intn(3) + 1)
	err := g.PluginManagerSwapper.Current().Extension.(*g.VaultExtension).OnJob(&job)
	if err != nil {
		g.Logger.Error(err)
	}
}
