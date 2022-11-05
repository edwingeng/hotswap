package guardian

import (
	"fmt"

	"github.com/edwingeng/hotswap/demo/livex/g"
	"github.com/edwingeng/hotswap/demo/livex/plugin/guardian/job"
	"github.com/edwingeng/hotswap/demo/livex/plugin/guardian/pg"
	"github.com/edwingeng/hotswap/vault"
	"github.com/edwingeng/live"
	"github.com/edwingeng/tickque"
)

const (
	pluginName = "guardian"
)

var (
	CompileTimeString string
)

func OnLoad(data interface{}) error {
	return nil
}

func OnInit(sharedVault *vault.Vault) error {
	pg.SharedVault = sharedVault
	sharedVault.Extension.(*g.VaultExtension).OnJob = OnJob
	return nil
}

func OnFree() {
	// NOP
}

func Export() interface{} {
	return nil
}

func Import() interface{} {
	return nil
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	switch name {
	case "MakeRollCall":
		job.MakeRollCall(pluginName, CompileTimeString)
	case "Fire":
		job.Fire(pluginName, CompileTimeString)
	default:
		panic("impossible")
	}
	return nil, nil
}

func Reloadable() bool {
	return true
}

type JobHandler1 = func(pluginName string, compileTimeString string, jobData live.Data) error
type JobHandler2 = interface {
	Handle(pluginName string, compileTimeString string) error
}

func OnJob(job *tickque.Job) error {
	if v, ok := pg.SharedVault.LiveFuncs[job.Type]; ok {
		return v.(JobHandler1)(pluginName, CompileTimeString, job.Data)
	}

	if f, ok := pg.SharedVault.LiveTypes[job.Type]; ok {
		newObj := f()
		job.Data.MustUnwrapObject(newObj)
		return newObj.(JobHandler2).Handle(pluginName, CompileTimeString)
	}

	return fmt.Errorf("unknown job: %s", job.Type)
}
