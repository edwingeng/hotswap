package gamma

import (
	"github.com/edwingeng/hotswap/demo/trine/plugin/gamma/gimpl"
	"github.com/edwingeng/hotswap/vault"
)

const (
	pluginName = "gamma"
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
	return nil
}

func Import() interface{} {
	return &Deps
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	switch name {
	case "pulse":
		gimpl.Pulse(pluginName, CompileTimeString, Deps.Beta)
	}
	return nil, nil
}

func Reloadable() bool {
	return true
}
