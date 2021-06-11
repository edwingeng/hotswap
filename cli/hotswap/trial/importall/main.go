package importall

import (
	"errors"
	"os"
	"strings"

	"github.com/edwingeng/hotswap/vault"
)

var (
	fxWhich string
)

func OnLoad(data interface{}) error {
	pluginLogger.Info("<importall> OnLoad")
	return nil
}

func OnInit(sharedVault *vault.Vault) error {
	pluginLogger.Info("<importall> OnInit")
	return nil
}

func OnFree() {
	pluginLogger.Info("<importall> OnFree")
}

func Export() interface{} {
	pluginLogger.Info("<importall> Export")
	return exportX{}
}

func Import() interface{} {
	pluginLogger.Info("<importall> Import")
	pluginTest := os.Getenv("pluginTest")
	switch {
	case strings.HasPrefix(pluginTest, "panics:"):
		fxWhich = "&fxPanic"
		return &fxPanic
	case strings.Contains(pluginTest, "mini"):
		fxWhich = "&fxMini"
		return &fxMini
	case strings.Contains(pluginTest, "stark"):
		fxWhich = "&fxStark"
		return &fxStark
	default:
		fxWhich = "&fx"
		return &fx
	}
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	pluginLogger.Info("<importall> InvokeFunc")
	switch name {
	case "fxWhich":
		return fxWhich, nil
	}
	return nil, errors.New("<importall> unreasonable error")
}

func Reloadable() bool {
	pluginLogger.Info("<importall> Reloadable")
	return true
}
