package job

import (
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/edwingeng/hotswap/demo/livex/g"
	"github.com/edwingeng/live"
)

var (
	namePool = []string{
		"Bronzeslice",
		"Burningwish",
		"Blinddeath",
		"Dualpack",
		"Steellash",
		"Vengefang",
		"Laughingmask",
		"Blackbone",
		"Ironkill",
		"Rabidcleaver",
	}

	respPool = []string{
		"I'm here",
		"Here I am",
		"Present",
		"Attend",
	}
)

func MakeRollCall(pluginName string, compileTimeString string) {
	name := namePool[rand.Intn(len(namePool))]
	g.Logger.Infof("<%s.%s> %s? reloadCounter: %v",
		pluginName, compileTimeString, name, g.PluginManagerSwapper.ReloadCounter())
	g.Tickque.AddJob("live_ResponseRollCall", g.LiveHelper.WrapString(name))
}

func live_ResponseRollCall(pluginName string, compileTimeString string, jobData live.Data) error {
	name := jobData.ToString()
	resp := respPool[rand.Intn(len(respPool))]
	g.Logger.Infof("<%s.%s> %s: %s. reloadCounter: %v",
		pluginName, compileTimeString, name, resp, g.PluginManagerSwapper.ReloadCounter())
	return nil
}

func Fire(pluginName string, compileTimeString string) {
	var job live_jobFire
	job.N = rand.Intn(3) + 1
	g.Logger.Infof("<%s.%s> Fire x %d. reloadCounter: %v",
		pluginName, compileTimeString, job.N, g.PluginManagerSwapper.ReloadCounter())

	go func() {
		time.Sleep(time.Second * 2)
		addJobIndirect(&job)
	}()
}

func addJobIndirect(obj interface{}) {
	typ := reflect.TypeOf(obj)
	if typ.Kind() != reflect.Ptr {
		panic("obj must be a pointer")
	}
	if typ.Elem().Kind() != reflect.Struct {
		panic("obj must be a pointer to a struct")
	}

	jobType := typ.Elem().Name()
	jobData := g.LiveHelper.WrapJSONObj(obj)
	g.Tickque.AddJob(jobType, jobData)
}

type live_jobFire struct {
	N int
}

func (wo *live_jobFire) Handle(pluginName string, compileTimeString string) error {
	str := strings.TrimSpace(strings.Repeat("Bang! ", wo.N))
	g.Logger.Infof("<%s.%s> %s. reloadCounter: %v",
		pluginName, compileTimeString, str, g.PluginManagerSwapper.ReloadCounter())
	return nil
}
