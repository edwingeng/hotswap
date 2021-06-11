package panics

import (
	"errors"
	"os"

	"github.com/edwingeng/hotswap/vault"
)

func OnLoad(data interface{}) error {
	pluginLogger.Info("<panics> OnLoad")
	str := "panics:OnLoad"
	if os.Getenv("pluginTest") == str {
		panic(str)
	}
	return nil
}

func OnInit(sharedVault *vault.Vault) error {
	pluginLogger.Info("<panics> OnInit")
	str := "panics:OnInit"
	if os.Getenv("pluginTest") == str {
		panic(str)
	}
	return nil
}

func OnFree() {
	pluginLogger.Info("<panics> OnFree")
	str := "panics:OnFree"
	if os.Getenv("pluginTest") == str {
		panic(str)
	}
}

func Export() interface{} {
	pluginLogger.Info("<panics> Export")
	str := "panics:Export"
	if os.Getenv("pluginTest") == str {
		panic(str)
	}
	return nil
}

func Import() interface{} {
	pluginLogger.Info("<panics> Import")
	str := "panics:Import"
	if os.Getenv("pluginTest") == str {
		panic(str)
	}
	return nil
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	pluginLogger.Info("<panics> InvokeFunc")
	str := "panics:InvokeFunc"
	if os.Getenv("pluginTest") == str {
		panic(str)
	}
	return nil, errors.New("<panics> unreasonable error")
}

func Reloadable() bool {
	pluginLogger.Info("<panics> Reloadable")
	str := "panics:Reloadable"
	if os.Getenv("pluginTest") == str {
		panic(str)
	}
	return true
}
