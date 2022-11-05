package aimpl

import "github.com/edwingeng/hotswap/demo/trine/g"

func One(str1 string, v1 g.Vector, pluginName, compileTimeString string) {
	g.Logger.Infof("<%s.%s> str1: %s, v1: %v. reloadCounter: %v",
		pluginName, compileTimeString, str1, v1, g.PluginManagerSwapper.ReloadCounter())
}
