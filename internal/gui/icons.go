package gui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/0xamirreza/portpilot/internal/gui/theme"
)

// drawIconGlobe renders a crisp vector wireframe globe.
func drawIconGlobe(gtx layout.Context, sz unit.Dp, col color.NRGBA) layout.Dimensions {
	px := gtx.Dp(sz)
	if px < 10 {
		px = 10
	}
	size := image.Pt(px, px)
	cx := float32(px) / 2.0
	cy := float32(px) / 2.0
	r := cx - 1.0

	// Outer sphere boundary
	outerRect := image.Rect(int(cx-r), int(cy-r), int(cx+r), int(cy+r))
	paint.FillShape(gtx.Ops, col, clip.Stroke{
		Path:  clip.Ellipse(outerRect).Path(gtx.Ops),
		Width: 1.2,
	}.Op())

	// Equator line
	var pe clip.Path
	pe.Begin(gtx.Ops)
	pe.MoveTo(f32.Pt(cx-r, cy))
	pe.LineTo(f32.Pt(cx+r, cy))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: pe.End(), Width: 1.0}.Op())

	// Prime Meridian ellipse (vertical)
	vertR := image.Rect(int(cx-r*0.45), int(cy-r), int(cx+r*0.45), int(cy+r))
	paint.FillShape(gtx.Ops, col, clip.Stroke{
		Path:  clip.Ellipse(vertR).Path(gtx.Ops),
		Width: 1.0,
	}.Op())

	// Central vertical axis
	var pa clip.Path
	pa.Begin(gtx.Ops)
	pa.MoveTo(f32.Pt(cx, cy-r))
	pa.LineTo(f32.Pt(cx, cy+r))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: pa.End(), Width: 0.9}.Op())

	return layout.Dimensions{Size: size}
}

// drawIconMap renders a folded parchment/map icon.
func drawIconMap(gtx layout.Context, sz unit.Dp, col color.NRGBA) layout.Dimensions {
	px := gtx.Dp(sz)
	if px < 10 {
		px = 10
	}
	size := image.Pt(px, px)
	w := float32(px)
	h := float32(px)

	// 3-panel folded map outline
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w*0.08, h*0.22))
	p.LineTo(f32.Pt(w*0.38, h*0.08))
	p.LineTo(f32.Pt(w*0.68, h*0.22))
	p.LineTo(f32.Pt(w*0.92, h*0.08))
	p.LineTo(f32.Pt(w*0.92, h*0.78))
	p.LineTo(f32.Pt(w*0.68, h*0.92))
	p.LineTo(f32.Pt(w*0.38, h*0.78))
	p.LineTo(f32.Pt(w*0.08, h*0.92))
	p.Close()
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.2}.Op())

	// Inner fold crease 1
	var pf1 clip.Path
	pf1.Begin(gtx.Ops)
	pf1.MoveTo(f32.Pt(w*0.38, h*0.08))
	pf1.LineTo(f32.Pt(w*0.38, h*0.78))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: pf1.End(), Width: 1.0}.Op())

	// Inner fold crease 2
	var pf2 clip.Path
	pf2.Begin(gtx.Ops)
	pf2.MoveTo(f32.Pt(w*0.68, h*0.22))
	pf2.LineTo(f32.Pt(w*0.68, h*0.92))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: pf2.End(), Width: 1.0}.Op())

	return layout.Dimensions{Size: size}
}

// drawIconRotate renders a circular refresh/rotate arrow.
func drawIconRotate(gtx layout.Context, sz unit.Dp, col color.NRGBA) layout.Dimensions {
	px := gtx.Dp(sz)
	if px < 10 {
		px = 10
	}
	size := image.Pt(px, px)
	cx := float32(px) / 2.0
	cy := float32(px) / 2.0
	r := cx - 1.5

	// ~270 degree circular arc
	var p clip.Path
	p.Begin(gtx.Ops)
	steps := 18
	startAng := 0.2 * math.Pi
	endAng := 1.75 * math.Pi
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		ang := startAng + t*(endAng-startAng)
		x := cx + r*float32(math.Cos(ang))
		y := cy + r*float32(math.Sin(ang))
		if i == 0 {
			p.MoveTo(f32.Pt(x, y))
		} else {
			p.LineTo(f32.Pt(x, y))
		}
	}
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.2}.Op())

	// Arrow head at arc end
	tipX := cx + r*float32(math.Cos(endAng))
	tipY := cy + r*float32(math.Sin(endAng))
	var pa clip.Path
	pa.Begin(gtx.Ops)
	pa.MoveTo(f32.Pt(tipX-3.5, tipY-1.0))
	pa.LineTo(f32.Pt(tipX+1.0, tipY))
	pa.LineTo(f32.Pt(tipX+0.5, tipY+4.0))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: pa.End(), Width: 1.3}.Op())

	return layout.Dimensions{Size: size}
}

// drawIconTarget renders a crosshair focus/target icon.
func drawIconTarget(gtx layout.Context, sz unit.Dp, col color.NRGBA) layout.Dimensions {
	px := gtx.Dp(sz)
	if px < 10 {
		px = 10
	}
	size := image.Pt(px, px)
	cx := float32(px) / 2.0
	cy := float32(px) / 2.0
	r := cx - 2.5

	// Outer ring
	rRect := image.Rect(int(cx-r), int(cy-r), int(cx+r), int(cy+r))
	paint.FillShape(gtx.Ops, col, clip.Stroke{
		Path:  clip.Ellipse(rRect).Path(gtx.Ops),
		Width: 1.2,
	}.Op())

	// Center bullseye dot
	dotRect := image.Rect(int(cx-1.5), int(cy-1.5), int(cx+1.5), int(cy+1.5))
	paint.FillShape(gtx.Ops, col, clip.Ellipse(dotRect).Op(gtx.Ops))

	// 4 Crosshair ticks
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(cx, 0.5))
	p.LineTo(f32.Pt(cx, cy-r+1.0))
	p.MoveTo(f32.Pt(cx, cy+r-1.0))
	p.LineTo(f32.Pt(cx, float32(px)-0.5))
	p.MoveTo(f32.Pt(0.5, cy))
	p.LineTo(f32.Pt(cx-r+1.0, cy))
	p.MoveTo(f32.Pt(cx+r-1.0, cy))
	p.LineTo(f32.Pt(float32(px)-0.5, cy))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.1}.Op())

	return layout.Dimensions{Size: size}
}

// drawIconNetwork renders a 3-node connected network hub icon.
func drawIconNetwork(gtx layout.Context, sz unit.Dp, col color.NRGBA) layout.Dimensions {
	px := gtx.Dp(sz)
	if px < 10 {
		px = 10
	}
	size := image.Pt(px, px)
	w := float32(px)
	h := float32(px)

	// Top center node
	n1 := f32.Pt(w*0.5, h*0.22)
	// Bottom left node
	n2 := f32.Pt(w*0.22, h*0.78)
	// Bottom right node
	n3 := f32.Pt(w*0.78, h*0.78)

	// Connecting lines
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(n1)
	p.LineTo(n2)
	p.MoveTo(n1)
	p.LineTo(n3)
	p.MoveTo(n2)
	p.LineTo(n3)
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.1}.Op())

	// Draw 3 node circles
	nr := float32(2.5)
	for _, n := range []f32.Point{n1, n2, n3} {
		nrRect := image.Rect(int(n.X-nr), int(n.Y-nr), int(n.X+nr), int(n.Y+nr))
		paint.FillShape(gtx.Ops, col, clip.Ellipse(nrRect).Op(gtx.Ops))
	}

	return layout.Dimensions{Size: size}
}

// drawIconArrowRight renders a clean directional arrow (→).
func drawIconArrowRight(gtx layout.Context, sz unit.Dp, col color.NRGBA) layout.Dimensions {
	wPx := gtx.Dp(sz)
	if wPx < 10 {
		wPx = 10
	}
	hPx := int(float32(wPx) * 0.7)
	size := image.Pt(wPx, hPx)
	w := float32(wPx)
	h := float32(hPx)
	midY := h / 2.0

	var p clip.Path
	p.Begin(gtx.Ops)
	// Shaft
	p.MoveTo(f32.Pt(0.5, midY))
	p.LineTo(f32.Pt(w-2.0, midY))
	// Arrowhead top
	p.MoveTo(f32.Pt(w-5.0, midY-3.5))
	p.LineTo(f32.Pt(w-1.0, midY))
	// Arrowhead bottom
	p.LineTo(f32.Pt(w-5.0, midY+3.5))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.3}.Op())

	return layout.Dimensions{Size: size}
}

// flagBadge renders a miniature, high-definition vector national flag or country badge.
// It completely replaces missing Unicode emoji glyphs with sharp, vector-drawn flags.
func (ui *UI) flagBadge(gtx layout.Context, countryCode string, w, h unit.Dp) layout.Dimensions {
	wPx := gtx.Dp(w)
	hPx := gtx.Dp(h)
	if wPx < 14 {
		wPx = 14
	}
	if hPx < 10 {
		hPx = 10
	}
	size := image.Pt(wPx, hPx)
	rect := image.Rectangle{Max: size}

	// Rounded clip boundary for smooth flag edges
	rr := 2
	rClip := clip.RRect{Rect: rect, SE: rr, SW: rr, NW: rr, NE: rr}.Push(gtx.Ops)

	code := strings.ToUpper(strings.TrimSpace(countryCode))
	switch code {
	case "IR": // 🇮🇷 Iran: Green, White, Red horizontal tricolor with center red emblem
		h1 := hPx / 3
		h2 := 2 * hPx / 3
		// Top: Green
		fill(gtx.Ops, image.Rect(0, 0, wPx, h1), color.NRGBA{35, 159, 64, 255})
		// Middle: White
		fill(gtx.Ops, image.Rect(0, h1, wPx, h2), color.NRGBA{255, 255, 255, 255})
		// Bottom: Red
		fill(gtx.Ops, image.Rect(0, h2, wPx, hPx), color.NRGBA{218, 0, 0, 255})
		// Center emblem
		cx := wPx / 2
		cy := (h1 + h2) / 2
		fill(gtx.Ops, image.Rect(cx-1, cy-2, cx+2, cy+2), color.NRGBA{218, 0, 0, 255})

	case "US": // 🇺🇸 United States: Red/white stripes with navy canton
		fill(gtx.Ops, rect, color.NRGBA{255, 255, 255, 255})
		// 5 stripes
		stripes := 5
		for s := 0; s < stripes; s += 2 {
			y0 := s * hPx / stripes
			y1 := (s + 1) * hPx / stripes
			fill(gtx.Ops, image.Rect(0, y0, wPx, y1), color.NRGBA{179, 25, 44, 255})
		}
		// Navy Canton (upper left)
		cantonW := wPx * 42 / 100
		cantonH := hPx * 55 / 100
		fill(gtx.Ops, image.Rect(0, 0, cantonW, cantonH), color.NRGBA{10, 40, 110, 255})
		// White star dots
		fill(gtx.Ops, image.Rect(cantonW/3, cantonH/3, cantonW/3+2, cantonH/3+2), color.NRGBA{255, 255, 255, 255})
		fill(gtx.Ops, image.Rect(cantonW*2/3, cantonH*2/3, cantonW*2/3+2, cantonH*2/3+2), color.NRGBA{255, 255, 255, 255})

	case "DE": // 🇩🇪 Germany: Black, Red, Gold
		h1 := hPx / 3
		h2 := 2 * hPx / 3
		fill(gtx.Ops, image.Rect(0, 0, wPx, h1), color.NRGBA{20, 20, 20, 255})
		fill(gtx.Ops, image.Rect(0, h1, wPx, h2), color.NRGBA{221, 0, 0, 255})
		fill(gtx.Ops, image.Rect(0, h2, wPx, hPx), color.NRGBA{255, 206, 0, 255})

	case "NL": // 🇳🇱 Netherlands: Red, White, Blue
		h1 := hPx / 3
		h2 := 2 * hPx / 3
		fill(gtx.Ops, image.Rect(0, 0, wPx, h1), color.NRGBA{174, 28, 40, 255})
		fill(gtx.Ops, image.Rect(0, h1, wPx, h2), color.NRGBA{255, 255, 255, 255})
		fill(gtx.Ops, image.Rect(0, h2, wPx, hPx), color.NRGBA{33, 70, 139, 255})

	case "FI": // 🇫🇮 Finland: White with Blue Nordic Cross
		fill(gtx.Ops, rect, color.NRGBA{250, 250, 252, 255})
		blue := color.NRGBA{0, 53, 128, 255}
		// Cross vertical bar
		barX := wPx * 34 / 100
		barW := int(float32(wPx) * 0.22)
		fill(gtx.Ops, image.Rect(barX, 0, barX+barW, hPx), blue)
		// Cross horizontal bar
		barY := hPx * 38 / 100
		barH := int(float32(hPx) * 0.26)
		fill(gtx.Ops, image.Rect(0, barY, wPx, barY+barH), blue)

	case "FR": // 🇫🇷 France: Blue, White, Red vertical
		w1 := wPx / 3
		w2 := 2 * wPx / 3
		fill(gtx.Ops, image.Rect(0, 0, w1, hPx), color.NRGBA{0, 38, 84, 255})
		fill(gtx.Ops, image.Rect(w1, 0, w2, hPx), color.NRGBA{255, 255, 255, 255})
		fill(gtx.Ops, image.Rect(w2, 0, wPx, hPx), color.NRGBA{237, 41, 57, 255})

	case "IT": // 🇮🇹 Italy: Green, White, Red vertical
		w1 := wPx / 3
		w2 := 2 * wPx / 3
		fill(gtx.Ops, image.Rect(0, 0, w1, hPx), color.NRGBA{0, 140, 69, 255})
		fill(gtx.Ops, image.Rect(w1, 0, w2, hPx), color.NRGBA{255, 255, 255, 255})
		fill(gtx.Ops, image.Rect(w2, 0, wPx, hPx), color.NRGBA{205, 33, 42, 255})

	case "EU": // 🇪🇺 European Union: Navy with circle of gold stars
		fill(gtx.Ops, rect, color.NRGBA{0, 51, 153, 255})
		gold := color.NRGBA{255, 204, 0, 255}
		cx := float64(wPx) / 2.0
		cy := float64(hPx) / 2.0
		rad := float64(hPx) * 0.32
		for a := 0.0; a < 2*math.Pi; a += math.Pi / 4 {
			sx := int(cx + rad*math.Cos(a))
			sy := int(cy + rad*math.Sin(a))
			fill(gtx.Ops, image.Rect(sx-1, sy-1, sx+1, sy+1), gold)
		}

	case "GB", "UK": // 🇬🇧 United Kingdom
		fill(gtx.Ops, rect, color.NRGBA{1, 33, 105, 255})
		white := color.NRGBA{255, 255, 255, 255}
		red := color.NRGBA{200, 16, 46, 255}
		// Diagonals
		fill(gtx.Ops, image.Rect(0, 0, wPx, hPx), color.NRGBA{1, 33, 105, 255})
		// White cross
		cx := wPx / 2
		cy := hPx / 2
		fill(gtx.Ops, image.Rect(cx-2, 0, cx+2, hPx), white)
		fill(gtx.Ops, image.Rect(0, cy-2, wPx, cy+2), white)
		// Red inner cross
		fill(gtx.Ops, image.Rect(cx-1, 0, cx+1, hPx), red)
		fill(gtx.Ops, image.Rect(0, cy-1, wPx, cy+1), red)

	case "SE": // 🇸🇪 Sweden: Blue with Yellow Nordic Cross
		fill(gtx.Ops, rect, color.NRGBA{0, 106, 167, 255})
		yellow := color.NRGBA{254, 204, 0, 255}
		barX := wPx * 34 / 100
		barW := int(float32(wPx) * 0.22)
		fill(gtx.Ops, image.Rect(barX, 0, barX+barW, hPx), yellow)
		barY := hPx * 38 / 100
		barH := int(float32(hPx) * 0.26)
		fill(gtx.Ops, image.Rect(0, barY, wPx, barY+barH), yellow)

	case "RU": // 🇷🇺 Russia: White, Blue, Red
		h1 := hPx / 3
		h2 := 2 * hPx / 3
		fill(gtx.Ops, image.Rect(0, 0, wPx, h1), color.NRGBA{255, 255, 255, 255})
		fill(gtx.Ops, image.Rect(0, h1, wPx, h2), color.NRGBA{0, 57, 166, 255})
		fill(gtx.Ops, image.Rect(0, h2, wPx, hPx), color.NRGBA{213, 43, 30, 255})

	case "HOST", "LOCAL", "LAN", "LINK": // Local host emblem
		fill(gtx.Ops, rect, color.NRGBA{35, 45, 52, 255})
		fill(gtx.Ops, image.Rect(1, 1, wPx-1, hPx-1), color.NRGBA{22, 30, 36, 255})
		// Small golden node in center
		cx := wPx / 2
		cy := hPx / 2
		fill(gtx.Ops, image.Rect(cx-2, cy-2, cx+2, cy+2), color.NRGBA{255, 215, 0, 255})

	case "GLOBAL": // International Internet
		fill(gtx.Ops, rect, color.NRGBA{14, 38, 64, 255})
		cx := wPx / 2
		cy := hPx / 2
		r := hPx/2 - 2
		gRect := image.Rect(cx-r, cy-r, cx+r, cy+r)
		paint.FillShape(gtx.Ops, color.NRGBA{0, 210, 255, 200}, clip.Stroke{
			Path:  clip.Ellipse(gRect).Path(gtx.Ops),
			Width: 1.0,
		}.Op())

	default: // Generic ISO country badge with elegant plate and 2-letter code
		fill(gtx.Ops, rect, color.NRGBA{28, 38, 48, 255})
		fill(gtx.Ops, image.Rect(1, 1, wPx-1, hPx-1), color.NRGBA{18, 26, 34, 255})
		labelStr := code
		if len(labelStr) > 2 {
			labelStr = labelStr[:2]
		}
		ui.label(gtx, unit.Sp(7.5), font.Bold, theme.Mono, color.NRGBA{180, 215, 240, 255}, labelStr)
	}

	rClip.Pop()

	// Subtle hairline border around badge
	borderRect := image.Rect(0, 0, wPx, hPx)
	paint.FillShape(gtx.Ops, color.NRGBA{255, 255, 255, 60}, clip.Stroke{
		Path:  clip.RRect{Rect: borderRect, SE: rr, SW: rr, NW: rr, NE: rr}.Path(gtx.Ops),
		Width: 1.0,
	}.Op())

	return layout.Dimensions{Size: size}
}

// iconButton renders a clickable button with a vector icon and label text.
func (ui *UI) iconButton(gtx layout.Context, click *widget.Clickable, iconFn func(gtx layout.Context, sz unit.Dp, col color.NRGBA) layout.Dimensions, txt string, on bool) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Ink, ui.th.Rule
		if on || click.Pressed() {
			bg, fg, line = ui.th.Ink, ui.th.Void, ui.th.Ink
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return iconFn(gtx, unit.Dp(14), fg)
					}),
					layout.Rigid(hgap(unit.Dp(7))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, fg, txt)
					}),
				)
			})
		})
	})
}

// countryPillButton renders a country filter pill with its vector flag badge and text.
func (ui *UI) countryPillButton(gtx layout.Context, click *widget.Clickable, countryCode, txt string, on bool) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Ink, ui.th.Rule
		if on || click.Pressed() {
			bg, fg, line = ui.th.Ink, ui.th.Void, ui.th.Ink
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				var kids []layout.FlexChild
				if countryCode != "" {
					kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.flagBadge(gtx, countryCode, unit.Dp(16), unit.Dp(11))
					}))
					kids = append(kids, layout.Rigid(hgap(unit.Dp(6))))
				}
				kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(11.5), font.Bold, theme.Sans, fg, txt)
				}))
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, kids...)
			})
		})
	})
}
