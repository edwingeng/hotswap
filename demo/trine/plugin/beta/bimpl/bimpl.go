package bimpl

import (
	"github.com/edwingeng/hotswap/demo/trine/g"
	"github.com/edwingeng/hotswap/demo/trine/plugin/export"
)

func Two(str1 string, v1 g.Vector, str2 string, v2 g.Vector, pluginName, compileTimeString string, alpha export.AlphaExport) {
	g.Logger.Infof("<%s.%s> str1: %s, v1: %v; str2: %s, v2: %v. reloadCounter: %v",
		pluginName, compileTimeString, str1, v1, str2, v2, g.PluginManagerSwapper.ReloadCounter())
	alpha.One(str1, v1)
}
