package job

import (
	"github.com/edwingeng/hotswap/demo/livex/g"
	"github.com/edwingeng/hotswap/demo/livex/plugin/guardian/job/handler"
	"github.com/edwingeng/live"
	"math/rand"
	"reflect"
	"time"
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
)

func MakeRollCall(pluginName string, compileTimeString string) {
	name := namePool[rand.Intn(len(namePool))]
	g.Logger.Infof("<%s.%s> %s? reloadCounter: %v",
		pluginName, compileTimeString, name, g.PluginManagerSwapper.ReloadCounter())
	g.Tickque.AddJob("live_ResponseRollCall", live.WrapString(name))
}

func Fire(pluginName string, compileTimeString string) {
	var job handler.Live_jobFire
	job.N = rand.Intn(3) + 1
	g.Logger.Infof("<%s.%s> Fire x %d. reloadCounter: %v",
		pluginName, compileTimeString, job.N, g.PluginManagerSwapper.ReloadCounter())
	addJobIndirect(&job)
}

func addJobIndirect(obj interface{}) {
	typ := reflect.TypeOf(obj)
	if typ.Kind() != reflect.Ptr {
		panic("obj must be a pointer")
	}
	if typ.Elem().Kind() != reflect.Struct {
		panic("obj must be a pointer to a struct")
	}

	go func() {
		time.Sleep(time.Second)
		jobType := typ.Elem().Name()
		jobData := live.MustWrapObject(obj)
		g.Tickque.AddJob(jobType, jobData)
	}()
}
