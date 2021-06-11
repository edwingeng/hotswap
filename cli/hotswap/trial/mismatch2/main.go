package mismatch2

import (
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
	return nil
}

func InvokeFunc() (interface{}, error) {
	return nil, nil
}

func Reloadable() bool {
	return true
}
