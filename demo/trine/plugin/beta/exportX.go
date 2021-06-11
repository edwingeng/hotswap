package beta

import "github.com/edwingeng/hotswap/demo/trine/g"

type exportX struct{}

func (_ exportX) Two(str1 string, v1 g.Vector, str2 string, v2 g.Vector) {
	g.Logger.Infof("<%s.%s> str1: %s, v1: %v; str2: %s, v2: %v. reloadCounter: %v",
		pluginName, CompileTimeString, str1, v1, str2, v2, g.PluginManagerSwapper.ReloadCounter())
	Deps.Alpha.One(str1, v1)
}
