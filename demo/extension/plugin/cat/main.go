package cat

import (
	"github.com/edwingeng/hotswap/demo/extension/g"
	"github.com/edwingeng/hotswap/demo/extension/plugin/cat/meow"
	"github.com/edwingeng/hotswap/vault"
)

const (
	pluginName = "cat"
)

var (
	CompileTimeString string
)

func OnLoad(data interface{}) error {
	return nil
}

func OnInit(sharedVault *vault.Vault) error {
	sharedVault.Extension.(*g.VaultExtension).Meow = func(repeat int) {
		meow.Meow(pluginName, CompileTimeString, repeat)
	}
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
