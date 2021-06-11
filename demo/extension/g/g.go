package g

import (
	"github.com/edwingeng/hotswap"
	"github.com/edwingeng/slog"
)

var (
	Logger = slog.NewConsoleLogger()
)

var (
	PluginManagerSwapper *hotswap.PluginManagerSwapper
)

type VaultExtension struct {
	Meow func(repeat int)
}

func NewVaultExtension() interface{} {
	return &VaultExtension{}
}
