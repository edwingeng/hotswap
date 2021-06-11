package arya

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/edwingeng/hotswap/cli/hotswap/trial/arya/needle"
	"github.com/edwingeng/hotswap/cli/hotswap/trial/arya/pg"
	"github.com/edwingeng/hotswap/internal/hctx"
	"github.com/edwingeng/hotswap/vault"
	"github.com/edwingeng/slog"
)

func OnLoad(data interface{}) error {
	return nil
}

func OnInit(sharedVault *vault.Vault) error {
	sharedVault.DataBag["arya:OnInit:called"] = true
	pg.SharedVault = sharedVault
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
	switch name {
	case "howl":
		return needle.Howl(), nil
	case "polish":
		return needle.Polish(params[0].(int)), nil
	}

	if v, ok := pg.SharedVault.LiveFuncs[name]; ok {
		ctx := hctx.NewContext(params[0].(slog.Logger))
		switch fn := v.(type) {
		case func(*hctx.Context, time.Time) string:
			return fn(ctx, params[1].(time.Time)), nil
		case func(*hctx.Context) string:
			return fn(ctx), nil
		default:
			return nil, fmt.Errorf("unexpected function signature. funcName: %s", name)
		}
	}

	if newObj, ok := pg.SharedVault.LiveTypes[name]; ok {
		obj := newObj()
		handler, ok := obj.(interface {
			Handle(ctx *hctx.Context) error
		})
		if !ok {
			return nil, fmt.Errorf("%s is not a job handler", reflect.TypeOf(obj).Name())
		}
		if err := json.Unmarshal(params[1].([]byte), obj); err != nil {
			return nil, err
		}

		ctx := hctx.NewContext(params[0].(slog.Logger))
		return nil, handler.Handle(ctx)
	}

	return nil, fmt.Errorf("unknown function name: " + name)
}

func Reloadable() bool {
	return true
}

func live_NotToday(ctx *hctx.Context, t time.Time) {
	ctx.Infof("Arya: Not today (%s).", t)
}
