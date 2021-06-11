package hctx

import (
	"context"

	"github.com/edwingeng/slog"
)

type Context struct {
	context.Context
	slog.Logger
}

func NewContext(log slog.Logger) *Context {
	return &Context{
		Context: context.Background(),
		Logger:  log,
	}
}
