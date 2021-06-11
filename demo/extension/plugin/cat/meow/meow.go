package meow

import (
	"strings"

	"github.com/edwingeng/hotswap/demo/extension/g"
)

func Meow(pluginName string, compileTimeString string, repeat int) {
	str := strings.TrimSpace(strings.Repeat("meow ", repeat))
	g.Logger.Infof("<%s.%s> %s. reloadCounter: %v",
		pluginName, compileTimeString, str, g.PluginManagerSwapper.ReloadCounter())
}
