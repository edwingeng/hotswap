package woof

import (
	"strings"

	"github.com/edwingeng/hotswap/demo/slink/g"
	"github.com/edwingeng/live"
)

func live_Woof(pluginName string, compileTimeString string, jobData live.Data) error {
	str := strings.TrimSpace(strings.Repeat("woof ", jobData.ToInt()))
	g.Logger.Infof("<%s.%s> %s. reloadCounter: %v",
		pluginName, compileTimeString, str, g.PluginManagerSwapper.ReloadCounter())
	return nil
}
