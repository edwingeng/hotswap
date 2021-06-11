package importall

import (
	"errors"

	"github.com/edwingeng/hotswap/cli/hotswap/trial/export/importall"
)

var (
	_ importall.Export = exportX{}
)

type exportX struct{}

func (_ exportX) TestDeps() error {
	if fx.Arya.Greet() != "Valar morghulis." {
		return errors.New("something is wrong with fx.Arya.Greet()")
	}
	if fx.Arya.Sneak() != "Arya slipped into the shadows." {
		return errors.New("something is wrong with fx.Arya.Sneak()")
	}
	if fx.Snow.Greet() != "I'm Jon Nothing Snow." {
		return errors.New("something is wrong with fx.Snow.Greet()")
	}
	if fx.Snow.Sword() != "Longclaw, forged in Valyrian Steel." {
		return errors.New("something is wrong with fx.Snow.Sword()")
	}
	return nil
}
