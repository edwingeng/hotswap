package alpha

import "github.com/edwingeng/hotswap/demo/trine/g"

type exportX struct{}

func (_ exportX) One(str1 string, v1 g.Vector) {
	g.Logger.Infof("<%s.%s> str1: %s, v1: %v. reloadCounter: %v",
		pluginName, CompileTimeString, str1, v1, g.PluginManagerSwapper.ReloadCounter())
}
