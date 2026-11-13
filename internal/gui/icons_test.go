package gui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
)

func TestVectorIconsRender(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Constraints: layout.Exact(image.Pt(300, 300)),
	}

	col := color.NRGBA{255, 255, 255, 255}

	// Test each standalone vector icon
	d1 := drawIconGlobe(gtx, unit.Dp(24), col)
	if d1.Size.X == 0 || d1.Size.Y == 0 {
		t.Errorf("drawIconGlobe returned zero dimensions: %+v", d1)
	}

	d2 := drawIconMap(gtx, unit.Dp(24), col)
	if d2.Size.X == 0 || d2.Size.Y == 0 {
		t.Errorf("drawIconMap returned zero dimensions: %+v", d2)
	}

	d3 := drawIconRotate(gtx, unit.Dp(24), col)
	if d3.Size.X == 0 || d3.Size.Y == 0 {
		t.Errorf("drawIconRotate returned zero dimensions: %+v", d3)
	}

	d4 := drawIconTarget(gtx, unit.Dp(24), col)
	if d4.Size.X == 0 || d4.Size.Y == 0 {
		t.Errorf("drawIconTarget returned zero dimensions: %+v", d4)
	}

	d5 := drawIconNetwork(gtx, unit.Dp(24), col)
	if d5.Size.X == 0 || d5.Size.Y == 0 {
		t.Errorf("drawIconNetwork returned zero dimensions: %+v", d5)
	}

	d6 := drawIconArrowRight(gtx, unit.Dp(16), col)
	if d6.Size.X == 0 || d6.Size.Y == 0 {
		t.Errorf("drawIconArrowRight returned zero dimensions: %+v", d6)
	}
}

func TestFlagBadgesRender(t *testing.T) {
	th, err := theme.Load()
	if err != nil {
		t.Fatalf("failed to load theme: %v", err)
	}
	ui := &UI{th: th}

	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Constraints: layout.Exact(image.Pt(200, 200)),
	}

	// Test all supported country codes and generic fallbacks
	codes := []string{
		"IR", "US", "DE", "NL", "FI", "FR", "IT", "EU", "GB", "UK",
		"RU", "SE", "JP", "CN", "CA", "HOST", "LOCAL", "LAN", "LINK", "GLOBAL",
		"SG", "BR", "AU", "ZA", "XYZ", "",
	}

	for _, code := range codes {
		dims := ui.flagBadge(gtx, code, unit.Dp(18), unit.Dp(12))
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("flagBadge(%s) returned empty size: %+v", code, dims)
		}
	}
}

func TestIconButtonAndCountryPill(t *testing.T) {
	th, err := theme.Load()
	if err != nil {
		t.Fatalf("failed to load theme: %v", err)
	}
	ui := &UI{th: th}

	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Constraints: layout.Exact(image.Pt(400, 100)),
	}

	btn := widget.Clickable{}
	d1 := ui.iconButton(gtx, &btn, drawIconGlobe, "TEST GLOBE", true)
	if d1.Size.X == 0 || d1.Size.Y == 0 {
		t.Errorf("iconButton returned zero dimensions: %+v", d1)
	}

	pillBtn := widget.Clickable{}
	d2 := ui.countryPillButton(gtx, &pillBtn, "IR", "IRAN (5)", false)
	if d2.Size.X == 0 || d2.Size.Y == 0 {
		t.Errorf("countryPillButton returned zero dimensions: %+v", d2)
	}
}
