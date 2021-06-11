package gamma

import (
	"bytes"
	"math/rand"

	"github.com/edwingeng/hotswap/demo/trine/g"
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
		g.Logger.Infof("<%s.%s> === pulse ===. reloadCounter: %v",
			pluginName, CompileTimeString, g.PluginManagerSwapper.ReloadCounter())
		Deps.Beta.Two(randString(), randVector(), randString(), randVector())
	}
	return nil, nil
}

func Reloadable() bool {
	return true
}

func randString() string {
	var buf bytes.Buffer
	n := rand.Intn(5) + 1
	for i := 0; i < n; i++ {
		buf.WriteByte(byte('A' + rand.Intn(26)))
	}
	return buf.String()
}

func randVector() g.Vector {
	return g.Vector{
		X: rand.Intn(100),
		Y: rand.Intn(100),
	}
}
