package hum

import (
	"strings"

	"github.com/edwingeng/hotswap/demo/hello/g"
)

func Hum(pluginName string, compileTimeString string, repeat int) {
	str := strings.TrimSpace(strings.Repeat("hum ", repeat))
	g.Logger.Infof("<%s.%s> %s. reloadCounter: %v",
		pluginName, compileTimeString, str, g.PluginManagerSwapper.ReloadCounter())
}
