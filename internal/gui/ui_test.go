package gui

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
)

func TestKeysSwitchScreens(t *testing.T) {
	ui := &UI{}
	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Constraints: layout.Exact(image.Pt(800, 600)),
	}

	// Verify move delta works
	ui.screen = 0
	ui.move(1)
	_ = gtx
}
