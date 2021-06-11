package importall

import "github.com/edwingeng/slog"

var (
	pluginLogger slog.Logger = slog.NewDumbLogger()
)

func SetLogger(log slog.Logger) {
	pluginLogger = log
}
