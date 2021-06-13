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
	"github.com/edwingeng/hotswap/demo/extension/g"
	"github.com/edwingeng/hotswap/internal/hutils"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {
	var pluginDir string
	var pidFile string
	flag.StringVar(&pluginDir, "pluginDir", "", "the directory holding your plugins")
	flag.StringVar(&pidFile, "pidFile", "", "pid file path")
	flag.Parse()

	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		panic(err)
	}
	if err := hutils.FindDirectory(absDir, ""); err != nil {
		panic(err)
	}
	if pidFile == "" {
		panic("no --pidFile")
	}

	pid := fmt.Sprint(os.Getpid())
	if err := ioutil.WriteFile(pidFile, []byte(pid), 0644); err != nil {
		panic(err)
	}

	g.PluginManagerSwapper = hotswap.NewPluginManagerSwapper(absDir,
		hotswap.WithLogger(g.Logger),
		hotswap.WithFreeDelay(time.Second*15),
		hotswap.WithExtensionNewer(g.NewVaultExtension),
	)
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

	// Wait for signals
	chSignal := make(chan os.Signal, 1)
	signal.Notify(chSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

loop:
	for {
		select {
		case sig := <-chSignal:
			g.Logger.Infof("signal received: %v", sig)
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				break loop
			case syscall.SIGUSR1:
				g.Logger.Info("<hotswap> reloading...")
				details, err := swapper.Reload(nil)
				if err != nil {
					panic(err)
				} else if len(details) == 0 {
					g.Logger.Infof("no plugin is found in " + absDir)
				} else {
					g.Logger.Infof("<hotswap> %d plugin(s) loaded. details: [%s]",
						len(details), details)
				}
				heartbeat()
			}
		}
	}

	signal.Reset(syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	g.Logger.Info("THE END")
}

func heartbeat() {
	repeat := rand.Intn(3) + 1
	g.PluginManagerSwapper.Current().Extension.(*g.VaultExtension).Meow(repeat)
}
