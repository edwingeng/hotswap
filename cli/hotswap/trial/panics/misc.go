package panics

import "github.com/edwingeng/slog"

var (
	pluginLogger slog.Logger
)

func SetLogger(log slog.Logger) {
	pluginLogger = log
}
