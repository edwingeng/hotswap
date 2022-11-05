package g

import (
	"github.com/edwingeng/hotswap"
	"github.com/edwingeng/slog"
)

var (
	Logger = slog.NewDevelopmentConfig().MustBuild()
)

var (
	PluginManagerSwapper *hotswap.PluginManagerSwapper
)
