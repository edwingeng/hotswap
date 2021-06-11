package g

import (
	"github.com/edwingeng/hotswap"
	"github.com/edwingeng/live"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/tickque"
)

var (
	Logger = slog.NewConsoleLogger()
)

var (
	PluginManagerSwapper *hotswap.PluginManagerSwapper
	LiveHelper           live.Helper
	Tickque              *tickque.Tickque
)

type VaultExtension struct {
	OnJob func(job *tickque.Job) error
}

func NewVaultExtension() interface{} {
	return &VaultExtension{}
}
