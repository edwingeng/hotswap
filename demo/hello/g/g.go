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
