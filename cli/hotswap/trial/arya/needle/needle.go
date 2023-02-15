package needle

import (
	"fmt"

	"github.com/edwingeng/hotswap/cli/hotswap/trial/syrio"
	"github.com/edwingeng/hotswap/internal/hctx"
	"github.com/edwingeng/hotswap/internal/hutils"
)

var (
	_ = syrio.WaterDance
)

func Howl() string {
	return "Howl..."
}

func Polish(repeat int) string {
	return fmt.Sprintf("Polished %d times.", repeat)
}

func Live_Anyone(ctx *hctx.Context) {
	ctx.Infof("Arya: Anyone can be killed.")
}

type Live_AryaKill struct {
	Names []string
}

func (ak *Live_AryaKill) Handle(ctx *hctx.Context) error {
	if len(ak.Names) > 0 {
		ctx.Infof("Arya killed %s.", hutils.Join(ak.Names...))
	}
	return nil
}
