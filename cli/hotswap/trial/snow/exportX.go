package snow

import "github.com/edwingeng/hotswap/cli/hotswap/trial/export/snow"

var (
	_ snow.Export = exportX{}
)

type exportX struct{}

func (_ exportX) Greet() string {
	return "I'm Jon Nothing Snow."
}

func (_ exportX) Sword() string {
	return "Longclaw, forged in Valyrian Steel."
}
