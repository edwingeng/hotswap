package importall

import (
	"github.com/edwingeng/hotswap/cli/hotswap/trial/export/arya"
	"github.com/edwingeng/hotswap/cli/hotswap/trial/export/snow"
)

var fx struct {
	Arya arya.Export
	Snow snow.Export
	bran string
}

var fxMini struct {
	Arya arya.Export
}

var fxPanic struct {
	Panics interface{}
}

var fxStark struct {
	Arya arya.Export
	Snow snow.Export
	Xdep interface{}
}
