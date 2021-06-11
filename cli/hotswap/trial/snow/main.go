package snow

import (
	"github.com/edwingeng/hotswap/vault"
	"github.com/edwingeng/slog"
)

var (
	snowLog slog.Logger
)

func OnLoad(data interface{}) error {
	snowLog = data.(slog.Logger)
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
	return nil
}

func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
	return nil, nil
}

func Reloadable() bool {
	return true
}

type live_Longclaw struct {
	// Empty
}
