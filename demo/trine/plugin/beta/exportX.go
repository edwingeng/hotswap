package beta

import (
	"github.com/edwingeng/hotswap/demo/trine/g"
	"github.com/edwingeng/hotswap/demo/trine/plugin/beta/bimpl"
)

type exportX struct{}

func (_ exportX) Two(str1 string, v1 g.Vector, str2 string, v2 g.Vector) {
	bimpl.Two(str1, v1, str2, v2, pluginName, CompileTimeString, Deps.Alpha)
}
