package beta

import (
	"github.com/edwingeng/hotswap/vault"
)

const (
	pluginName = "beta"
)

var (
	CompileTimeString string
)

func OnLoad(data interface{}) error {
	return nil
}

func OnInit(sharedVault *vault.Vault) error {
	return nil
}

func OnFree() {
	// NOP
}

func Export() interface{} {
	return exportX{}
}

func Import() interface{} {
	return &Deps
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	return nil, nil
}

func Reloadable() bool {
	return true
}
