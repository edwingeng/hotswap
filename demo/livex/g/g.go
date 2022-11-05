package g

import (
	"github.com/edwingeng/hotswap"
	"github.com/edwingeng/live"
	"github.com/edwingeng/slog"
	"github.com/edwingeng/tickque"
)

var (
	Logger = slog.NewDevelopmentConfig().MustBuild()
)

var (
	PluginManagerSwapper *hotswap.PluginManagerSwapper
	Tickque              *tickque.Tickque
)

var (
	LiveConfig live.Config
)

type VaultExtension struct {
	OnJob func(job *tickque.Job) error
}

func NewVaultExtension() interface{} {
	return &VaultExtension{}
}
