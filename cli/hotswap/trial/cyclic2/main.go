package cyclic2

import (
	"os"

	"github.com/edwingeng/hotswap/vault"
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
	if os.Getenv("pluginTest") == "cyclic2:fx3" {
		return &fx3
	} else {
		return &fx2
	}
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	return nil, nil
}

func Reloadable() bool {
	return true
}
