package handler

import (
	"math/rand"
	"strings"

	"github.com/edwingeng/hotswap/demo/livex/g"
	"github.com/edwingeng/live"
)

var (
	respPool = []string{
		"I'm here",
		"Here I am",
		"Present",
		"Attend",
	}
)

func live_ResponseRollCall(pluginName string, compileTimeString string, jobData live.Data) error {
	name := jobData.String()
	resp := respPool[rand.Intn(len(respPool))]
	g.Logger.Infof("<%s.%s> %s: %s. reloadCounter: %v",
		pluginName, compileTimeString, name, resp, g.PluginManagerSwapper.ReloadCounter())
	return nil
}

type Live_jobFire struct {
	N int
}

func (jf *Live_jobFire) Handle(pluginName string, compileTimeString string) error {
	str := strings.TrimSpace(strings.Repeat("Bang! ", jf.N))
	g.Logger.Infof("<%s.%s> %s. reloadCounter: %v",
		pluginName, compileTimeString, str, g.PluginManagerSwapper.ReloadCounter())
	return nil
}
