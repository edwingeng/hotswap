package gimpl

import (
	"bytes"
	"github.com/edwingeng/hotswap/demo/trine/g"
	"github.com/edwingeng/hotswap/demo/trine/plugin/export"
	"math/rand"
)

func Pulse(pluginName, compileTimeString string, beta export.BetaExport) {
	g.Logger.Infof("<%s.%s> === pulse ===. reloadCounter: %v",
		pluginName, compileTimeString, g.PluginManagerSwapper.ReloadCounter())
	beta.Two(randString(), randVector(), randString(), randVector())
}

func randString() string {
	var buf bytes.Buffer
	n := rand.Intn(5) + 1
	for i := 0; i < n; i++ {
		buf.WriteByte(byte('A' + rand.Intn(26)))
	}
	return buf.String()
}

func randVector() g.Vector {
	return g.Vector{
		X: rand.Intn(100),
		Y: rand.Intn(100),
	}
}
