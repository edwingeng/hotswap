package arya

import (
	"github.com/edwingeng/hotswap/cli/hotswap/trial/export/arya"
)

var (
	_ arya.Export = exportX{}
)

type exportX struct{}

func (_ exportX) Greet() string {
	return "Valar morghulis."
}

func (_ exportX) Sneak() string {
	return "Arya slipped into the shadows."
}
