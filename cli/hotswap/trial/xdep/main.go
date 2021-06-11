package xdep

import (
	"os"

	"github.com/edwingeng/hotswap/vault"
)

var (
	fxWhich string
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
	pluginTest := os.Getenv("pluginTest")
	switch pluginTest {
	case "xdep:ignore":
		fxWhich = "&fxIgnore"
		return &fxIgnore
	case "xdep:mini":
		fxWhich = "&fxMini"
		return &fxMini
	case "xdep:import-returns-obj":
		fxWhich = "fx"
		return fx
	case "xdep:none":
		fxWhich = "&struct{}{}"
		return &struct{}{}
	case "xdep:unknown":
		fxWhich = "&fxUnknown"
		return &fxUnknown
	case "xdep:nameless":
		fxWhich = "&fxNameless"
		return &fxNameless
	case "xdep:stark":
		fxWhich = "&fxStark"
		return &fxStark
	default:
		fxWhich = "&fx"
		return &fx
	}
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	switch name {
	case "fxWhich":
		return fxWhich, nil
	}
	return nil, nil
}

func Reloadable() bool {
	if os.Getenv("pluginTest") == "xdep:not-reloadable" {
		return false
	}
	return true
}
