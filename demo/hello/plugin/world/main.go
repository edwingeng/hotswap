package world

import (
	"github.com/edwingeng/hotswap/demo/hello/g"
	"github.com/edwingeng/hotswap/demo/hello/plugin/world/hum"
	"github.com/edwingeng/hotswap/vault"
)

const (
	pluginName = "world"
)

var (
	CompileTimeString string
)

func OnLoad(data interface{}) error {
	g.Logger.Infof("<%s.%s> OnLoad", pluginName, CompileTimeString)
	return nil
}

func OnInit(sharedVault *vault.Vault) error {
	g.Logger.Infof("<%s.%s> OnInit", pluginName, CompileTimeString)
	return nil
}

func OnFree() {
	g.Logger.Infof("<%s.%s> OnFree", pluginName, CompileTimeString)
}

func Export() interface{} {
	g.Logger.Infof("<%s.%s> Export", pluginName, CompileTimeString)
	return nil
}

func Import() interface{} {
	g.Logger.Infof("<%s.%s> Import", pluginName, CompileTimeString)
	return nil
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	switch name {
	case "hum":
		repeat := params[0].(int)
		hum.Hum(pluginName, CompileTimeString, repeat)
	}
	return nil, nil
}

func Reloadable() bool {
	g.Logger.Infof("<%s.%s> Reloadable", pluginName, CompileTimeString)
	return true
}
