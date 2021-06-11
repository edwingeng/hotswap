package dog

import (
	"fmt"

	"github.com/edwingeng/hotswap/demo/slink/g"
	"github.com/edwingeng/hotswap/demo/slink/plugin/dog/woof/pg"
	"github.com/edwingeng/hotswap/vault"
	"github.com/edwingeng/live"
	"github.com/edwingeng/tickque"
)

const (
	pluginName = "dog"
)

var (
	CompileTimeString = "default"
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
	return nil, nil
}

func Reloadable() bool {
	return true
}

type JobHandler1 = func(pluginName string, compileTimeString string, jobData live.Data) error

func OnJob(job *tickque.Job) error {
	if v, ok := pg.SharedVault.LiveFuncs[job.Type]; ok {
		return v.(JobHandler1)(pluginName, CompileTimeString, job.Data)
	}

	return fmt.Errorf("unknown job: %s", job.Type)
}
