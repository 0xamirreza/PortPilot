package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/0xamirreza/portpilot/internal/core/network/geoip"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
)

// Destination node aggregated for visualization.
type geoDestNode struct {
	code      string
	name      string
	flag      string
	lat       float64
	lon       float64
	flowCount int
	bytesSent uint64
	bytesRecv uint64
}

// projectEquirectangular maps spherical coordinates (lat, lon) directly onto normalized [0, 1] coordinates.
func projectEquirectangular(lat, lon float64, w, h, offX, offY float64) f32.Point {
	normLon := (lon + 180.0) / 360.0
	normLat := (90.0 - lat) / 180.0
	if normLat < 0 {
		normLat = 0
	} else if normLat > 1 {
		normLat = 1
	}
	return f32.Pt(float32(offX+normLon*w), float32(offY+normLat*h))
}

// nodeNormCoords returns normalized (nx, ny in [0, 1]) map coordinates for any destination node,
// ensuring GLOBAL and all world countries map consistently into mapPoint space.
func (ui *UI) nodeNormCoords(node *geoDestNode) (float32, float32) {
	if c := GetMapCountry(node.code); c != nil {
		return c.Center[0], c.Center[1]
	}
	// Fallback conversion from (lat, lon) to normalized map coordinates
	rad := node.lat * math.Pi / 180.0
	yMil := 1.25 * math.Log(math.Tan(math.Pi/4.0+0.4*rad))
	normY := float32(0.655 - 0.35*yMil)
	normX := float32((node.lon + 172.0) / 355.0)
	if normX < 0 {
		normX = 0
	} else if normX > 1 {
		normX = 1
	}
	if normY < 0 {
		normY = 0
	} else if normY > 1 {
		normY = 1
	}
	return normX, normY
}

// mapPoint transforms a normalized point (nx, ny in [0, 1]) from the SVG dataset into screen space,
// accounting for map aspect ratio, zoom level, and pan translation.
func (ui *UI) mapPoint(nx, ny float32, mapW, mapH, offX, offY float64) f32.Point {
	cx := offX + mapW/2.0
	cy := offY + mapH/2.0

	bx := offX + float64(nx)*mapW
	by := offY + float64(ny)*mapH

	z := ui.geoZoom
	if z < 1.0 {
		z = 1.0
	}

	sx := cx + (bx-cx)*z + ui.geoPanX
	sy := cy + (by-cy)*z + ui.geoPanY
	return f32.Pt(float32(sx), float32(sy))
}

// focusMapCountry zooms in (2.4x) and centers the map view smoothly on the requested country.
func (ui *UI) focusMapCountry(code string, mapW, mapH, offX, offY float64) {
	c := GetMapCountry(code)
	if c == nil {
		return
	}
	ui.geoZoom = 2.4
	cx := offX + mapW/2.0
	cy := offY + mapH/2.0
	bx := offX + float64(c.Center[0])*mapW
	by := offY + float64(c.Center[1])*mapH

	ui.geoPanX = -(bx - cx) * ui.geoZoom
	ui.geoPanY = -(by - cy) * ui.geoZoom

	maxPanX := (mapW * (ui.geoZoom - 1.0)) / 2.0
	maxPanY := (mapH * (ui.geoZoom - 1.0)) / 2.0
	if ui.geoPanX > maxPanX {
		ui.geoPanX = maxPanX
	} else if ui.geoPanX < -maxPanX {
		ui.geoPanX = -maxPanX
	}
	if ui.geoPanY > maxPanY {
		ui.geoPanY = maxPanY
	} else if ui.geoPanY < -maxPanY {
		ui.geoPanY = -maxPanY
	}
}

// drawStarfield paints subtle cyber background stars around canvas.
func drawStarfield(ops *op.Ops, size image.Point) {
	dots := [][2]int{
		{20, 25}, {140, 40}, {310, 15}, {450, 30}, {620, 20}, {780, 35}, {920, 18},
		{60, 280}, {210, 290}, {410, 275}, {590, 285}, {750, 270}, {890, 280},
	}
	for _, d := range dots {
		if d[0] < size.X && d[1] < size.Y {
			r := image.Rect(d[0], d[1], d[0]+2, d[1]+2)
			paint.FillShape(ops, color.NRGBA{100, 145, 185, 45}, clip.Rect(r).Op())
		}
	}
}

// globeView renders the interactive Flat World Map (2D) with live outbound flow arcs, zoom, and pan controls.
func (ui *UI) globeView(gtx layout.Context, flows []geoFlowItem, localIP string) layout.Dimensions {
	now := time.Now()
	dt := 0.033
	if !ui.geoLastFrame.IsZero() {
		dt = now.Sub(ui.geoLastFrame).Seconds()
		if dt > 0.1 {
			dt = 0.033
		}
	}
	ui.geoLastFrame = now
	ui.geoAnimTick += dt

	if ui.geoZoom < 1.0 {
		ui.geoZoom = 1.0
	}

	// Aggregate destination nodes
	destMap := make(map[string]*geoDestNode)
	for _, f := range flows {
		code := f.destLoc.CountryCode
		if code == "" || f.destLoc.IsLocal {
			continue
		}
		node, exists := destMap[code]
		if !exists {
			lat, lon := f.destLoc.Latitude, f.destLoc.Longitude
			if lat == 0 && lon == 0 {
				lat, lon = geoip.Coordinates(code)
			}
			node = &geoDestNode{
				code: code,
				name: f.destLoc.CountryName,
				flag: f.destLoc.Flag,
				lat:  lat,
				lon:  lon,
			}
			destMap[code] = node
		}
		node.flowCount++
		if f.socket.BytesKnown {
			node.bytesSent += f.socket.BytesSent
			node.bytesRecv += f.socket.BytesReceived
		}
	}

	nodes := make([]*geoDestNode, 0, len(destMap))
	for _, n := range destMap {
		nodes = append(nodes, n)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].flowCount > nodes[j].flowCount
	})

	// Trigger next frame for smooth animation
	gtx.Execute(op.InvalidateCmd{At: now.Add(33 * time.Millisecond)})

	// Canvas dimensions calculation for zoom/pan actions
	height := gtx.Dp(unit.Dp(330))
	width := gtx.Constraints.Max.X
	size := image.Pt(width, height)

	margin := 8.0
	availW := float64(size.X) - 2.0*margin
	availH := float64(size.Y) - 2.0*margin

	const mapAspect = 784.077 / 458.627
	var mapW, mapH, offX, offY float64
	if availW/availH > mapAspect {
		mapH = availH
		mapW = availH * mapAspect
		offX = margin + (availW-mapW)/2.0
		offY = margin
	} else {
		mapW = availW
		mapH = availW / mapAspect
		offX = margin
		offY = margin + (availH-mapH)/2.0
	}

	// Toolbar button interactions
	if ui.geoZoomInBtn.Clicked(gtx) {
		ui.geoZoom = math.Min(8.0, ui.geoZoom*1.4)
	}
	if ui.geoZoomOutBtn.Clicked(gtx) {
		ui.geoZoom = math.Max(1.0, ui.geoZoom/1.4)
		if ui.geoZoom <= 1.0 {
			ui.geoPanX = 0
			ui.geoPanY = 0
		}
	}
	if ui.geoResetBtn.Clicked(gtx) {
		ui.geoZoom = 1.0
		ui.geoPanX = 0
		ui.geoPanY = 0
		ui.geoSelectedCountry = ""
	}
	if ui.geoFocusHostBtn.Clicked(gtx) {
		ui.focusMapCountry("IR", mapW, mapH, offX, offY)
		ui.geoSelectedCountry = "IR"
	}

	// Pan clamping
	if ui.geoZoom <= 1.0 {
		ui.geoPanX = 0
		ui.geoPanY = 0
	} else {
		maxPanX := (mapW * (ui.geoZoom - 1.0)) / 2.0
		maxPanY := (mapH * (ui.geoZoom - 1.0)) / 2.0
		if ui.geoPanX > maxPanX {
			ui.geoPanX = maxPanX
		} else if ui.geoPanX < -maxPanX {
			ui.geoPanX = -maxPanX
		}
		if ui.geoPanY > maxPanY {
			ui.geoPanY = maxPanY
		} else if ui.geoPanY < -maxPanY {
			ui.geoPanY = -maxPanY
		}
	}

	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// Header Toolbar
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return drawIconMap(gtx, unit.Dp(16), ui.th.Ink)
								}),
								layout.Rigid(hgap(unit.Dp(7))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(13), font.Bold, theme.Sans, ui.th.Ink, "FLAT WORLD TOPOLOGY MAP")
								}),
							)
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							zoomLabel := fmt.Sprintf("ZOOM: %.1fx · DRAG TO PAN", ui.geoZoom)
							return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, ui.th.Olive, zoomLabel)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
						// Zoom and View Controls
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.iconButton(gtx, &ui.geoZoomInBtn, drawIconTarget, "ZOOM IN (+)", false)
						}),
						layout.Rigid(hgap(unit.Dp(6))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.iconButton(gtx, &ui.geoZoomOutBtn, drawIconTarget, "ZOOM OUT (-)", false)
						}),
						layout.Rigid(hgap(unit.Dp(6))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.iconButton(gtx, &ui.geoResetBtn, drawIconRotate, "RESET (1:1)", ui.geoZoom == 1.0)
						}),
						layout.Rigid(hgap(unit.Dp(6))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.iconButton(gtx, &ui.geoFocusHostBtn, drawIconNetwork, "FOCUS HOST (IRAN)", ui.geoSelectedCountry == "IR")
						}),
					)
				}),
				layout.Rigid(gap(unit.Dp(8))),

				// Canvas Area
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// 1. Strictly clip all pointer events, gestures, and drawing to the map canvas area
					// so map drag gestures cannot intercept clicks outside (e.g. sidebar or tabs).
					canvasClip := clip.Rect(image.Rectangle{Max: size}).Push(gtx.Ops)

					// 2. Drag gesture handling for panning across map
					ui.geoGlobeDrag.Add(gtx.Ops)
					for {
						ev, ok := ui.geoGlobeDrag.Update(gtx.Metric, gtx.Source, gesture.Both)
						if !ok {
							break
						}
						switch ev.Kind {
						case pointer.Press:
							ui.geoDragLast = ev.Position
						case pointer.Drag:
							diff := ev.Position.Sub(ui.geoDragLast)
							ui.geoDragLast = ev.Position
							if ui.geoZoom > 1.0 {
								ui.geoPanX += float64(diff.X)
								ui.geoPanY += float64(diff.Y)
							}
						case pointer.Release, pointer.Cancel:
							// Gesture finished cleanly
						}
					}

					// 3. Mouse scroll wheel for zoom with dedicated tag inside canvas
					event.Op(gtx.Ops, &ui.geoZoom)
					for {
						ev, ok := gtx.Event(pointer.Filter{
							Target:  &ui.geoZoom,
							Kinds:   pointer.Scroll,
							ScrollY: pointer.ScrollRange{Min: -10000, Max: 10000},
						})
						if !ok {
							break
						}
						if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Scroll {
							if pe.Scroll.Y < 0 {
								ui.geoZoom = math.Min(8.0, ui.geoZoom*1.15)
							} else if pe.Scroll.Y > 0 {
								ui.geoZoom = math.Max(1.0, ui.geoZoom/1.15)
								if ui.geoZoom <= 1.0 {
									ui.geoPanX = 0
									ui.geoPanY = 0
								}
							}
						}
					}

					// Canvas background
					canvasBg := color.NRGBA{8, 14, 22, 255}
					fill(gtx.Ops, image.Rectangle{Max: size}, canvasBg)
					drawStarfield(gtx.Ops, size)

					ui.draw2DMap(gtx, size, mapW, mapH, offX, offY, nodes)

					canvasClip.Pop()
					return layout.Dimensions{Size: size}
				}),
			)
		})
	})
}

// draw2DMap renders the flat SVG world map with separated, demarcated countries from flekschas/simple-world-map.
func (ui *UI) draw2DMap(gtx layout.Context, size image.Point, mapW, mapH, offX, offY float64, nodes []*geoDestNode) {
	// 1. Map Canvas Border & Deep Oceanic Base
	mapRect := image.Rect(int(offX), int(offY), int(offX+mapW), int(offY+mapH))
	oceanColor := color.NRGBA{10, 18, 28, 255}
	paint.FillShape(gtx.Ops, oceanColor, clip.Rect(mapRect).Op())
	paint.FillShape(gtx.Ops, color.NRGBA{35, 75, 110, 180}, clip.Stroke{
		Path:  clip.Rect(mapRect).Path(),
		Width: 1.5,
	}.Op())

	defer clip.Rect(mapRect).Push(gtx.Ops).Pop()

	// 2. Graticule Lines (Parallels & Meridians with Zoom Scaling)
	gridCol := color.NRGBA{25, 45, 66, 75}
	equatorCol := color.NRGBA{40, 95, 130, 120}
	for _, f := range []float64{0.2, 0.35, 0.5, 0.65, 0.8} {
		p1 := ui.mapPoint(0.0, float32(f), mapW, mapH, offX, offY)
		p2 := ui.mapPoint(1.0, float32(f), mapW, mapH, offX, offY)
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(p1)
		p.LineTo(p2)
		c := gridCol
		if f == 0.5 {
			c = equatorCol
		}
		paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: 1.0}.Op())
	}
	for _, f := range []float64{0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875} {
		p1 := ui.mapPoint(float32(f), 0.0, mapW, mapH, offX, offY)
		p2 := ui.mapPoint(float32(f), 1.0, mapW, mapH, offX, offY)
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(p1)
		p.LineTo(p2)
		paint.FillShape(gtx.Ops, gridCol, clip.Stroke{Path: p.End(), Width: 1.0}.Op())
	}

	// 3. Render All 180 Demarcated Countries from flekschas/simple-world-map
	activeCountryMap := make(map[string]bool)
	for _, n := range nodes {
		activeCountryMap[n.code] = true
	}

	for _, country := range SimpleWorldMapCountries {
		isHost := country.Code == "IR"
		isSelected := ui.geoSelectedCountry == country.Code
		hasFlows := activeCountryMap[country.Code]

		fillCol := color.NRGBA{25, 42, 58, 255}
		borderCol := color.NRGBA{48, 85, 118, 190}
		borderWidth := float32(0.9)

		if isHost {
			fillCol = color.NRGBA{78, 62, 20, 245}
			borderCol = color.NRGBA{255, 215, 0, 255}
			borderWidth = 2.0
		} else if isSelected {
			fillCol = color.NRGBA{45, 90, 130, 245}
			borderCol = color.NRGBA{255, 200, 40, 255}
			borderWidth = 2.0
		} else if hasFlows {
			fillCol = color.NRGBA{18, 68, 92, 235}
			borderCol = color.NRGBA{0, 225, 255, 255}
			borderWidth = 1.5
		}

		for _, poly := range country.Polys {
			if len(poly) < 2 {
				continue
			}
			var p clip.Path
			p.Begin(gtx.Ops)
			for i, pt := range poly {
				sPt := ui.mapPoint(pt[0], pt[1], mapW, mapH, offX, offY)
				if i == 0 {
					p.MoveTo(sPt)
				} else {
					p.LineTo(sPt)
				}
			}
			spec := p.End()
			paint.FillShape(gtx.Ops, fillCol, clip.Outline{Path: spec}.Op())
			paint.FillShape(gtx.Ops, borderCol, clip.Stroke{Path: spec, Width: borderWidth}.Op())
		}
	}

	// 4. Host Marker Coordinates (Iran from simple-world-map)
	var hostPt f32.Point
	if irCountry := GetMapCountry("IR"); irCountry != nil {
		hostPt = ui.mapPoint(irCountry.Center[0], irCountry.Center[1], mapW, mapH, offX, offY)
	} else {
		hostPt = ui.mapPoint(0.634, 0.439, mapW, mapH, offX, offY)
	}

	// 5. Parabolic Flight Arcs (Host -> Destinations)
	pulseT := math.Mod(ui.geoAnimTick*0.75, 1.0)
	for _, node := range nodes {
		nx, ny := ui.nodeNormCoords(node)
		destPt := ui.mapPoint(nx, ny, mapW, mapH, offX, offY)

		isSelected := ui.geoSelectedCountry == node.code
		arcCol := color.NRGBA{0, 210, 255, 150}
		arcWidth := float32(1.5)
		if isSelected {
			arcCol = color.NRGBA{255, 200, 40, 240}
			arcWidth = 2.2
		}

		midX := (hostPt.X + destPt.X) / 2.0
		minY := float32(math.Min(float64(hostPt.Y), float64(destPt.Y)))
		archHeight := float32(math.Abs(float64(destPt.X-hostPt.X))*0.18 + 25.0)
		ctrlPt := f32.Pt(midX, minY-archHeight)

		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(hostPt)
		p.QuadTo(ctrlPt, destPt)
		paint.FillShape(gtx.Ops, arcCol, clip.Stroke{Path: p.End(), Width: arcWidth}.Op())

		// Animated traveling pulse packet along arc
		t := float32(pulseT)
		pulseX := (1-t)*(1-t)*hostPt.X + 2*(1-t)*t*ctrlPt.X + t*t*destPt.X
		pulseY := (1-t)*(1-t)*hostPt.Y + 2*(1-t)*t*ctrlPt.Y + t*t*destPt.Y
		pRect := image.Rect(int(pulseX-3), int(pulseY-3), int(pulseX+3), int(pulseY+3))
		paint.FillShape(gtx.Ops, color.NRGBA{255, 255, 255, 255}, clip.Ellipse(pRect).Op(gtx.Ops))
	}

	// 6. Destination Markers & Badges
	for _, node := range nodes {
		nx, ny := ui.nodeNormCoords(node)
		destPt := ui.mapPoint(nx, ny, mapW, mapH, offX, offY)

		isSelected := ui.geoSelectedCountry == node.code
		dotCol := color.NRGBA{0, 230, 200, 255}
		dotRadius := 4
		if isSelected {
			dotCol = color.NRGBA{255, 200, 40, 255}
			dotRadius = 6
		}

		haloRect := image.Rect(int(destPt.X)-dotRadius-3, int(destPt.Y)-dotRadius-3, int(destPt.X)+dotRadius+3, int(destPt.Y)+dotRadius+3)
		paint.FillShape(gtx.Ops, color.NRGBA{dotCol.R, dotCol.G, dotCol.B, 70}, clip.Ellipse(haloRect).Op(gtx.Ops))
		coreRect := image.Rect(int(destPt.X)-dotRadius, int(destPt.Y)-dotRadius, int(destPt.X)+dotRadius, int(destPt.Y)+dotRadius)
		paint.FillShape(gtx.Ops, dotCol, clip.Ellipse(coreRect).Op(gtx.Ops))

		offset := op.Offset(image.Pt(int(destPt.X)+7, int(destPt.Y)-9)).Push(gtx.Ops)
		layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.flagBadge(gtx, node.code, unit.Dp(15), unit.Dp(10.5))
			}),
			layout.Rigid(hgap(unit.Dp(5))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				bText := fmt.Sprintf("%s (%d)", node.code, node.flowCount)
				return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Ink, bText)
			}),
		)
		offset.Pop()
	}

	// 7. Host Marker (Tehran, Iran)
	radarT := math.Mod(ui.geoAnimTick*0.8, 1.0)
	for r := 1; r <= 3; r++ {
		curR := float64(r*7) + radarT*8.0
		alpha := uint8(math.Max(0, 180.0-curR*6.5))
		ringRect := image.Rect(int(float64(hostPt.X)-curR), int(float64(hostPt.Y)-curR), int(float64(hostPt.X)+curR), int(float64(hostPt.Y)+curR))
		paint.FillShape(gtx.Ops, color.NRGBA{255, 215, 0, alpha}, clip.Stroke{
			Path:  clip.Ellipse(ringRect).Path(gtx.Ops),
			Width: 1.2,
		}.Op())
	}

	hostCore := image.Rect(int(hostPt.X)-5, int(hostPt.Y)-5, int(hostPt.X)+5, int(hostPt.Y)+5)
	paint.FillShape(gtx.Ops, color.NRGBA{255, 215, 0, 255}, clip.Ellipse(hostCore).Op(gtx.Ops))
	hostCenter := image.Rect(int(hostPt.X)-2, int(hostPt.Y)-2, int(hostPt.X)+2, int(hostPt.Y)+2)
	paint.FillShape(gtx.Ops, color.NRGBA{255, 255, 255, 255}, clip.Ellipse(hostCenter).Op(gtx.Ops))

	hostOffset := op.Offset(image.Pt(int(hostPt.X)+9, int(hostPt.Y)-9)).Push(gtx.Ops)
	layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.flagBadge(gtx, "IR", unit.Dp(17), unit.Dp(11.5))
		}),
		layout.Rigid(hgap(unit.Dp(5))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, color.NRGBA{255, 215, 0, 255}, "HOST · IRAN")
		}),
	)
	hostOffset.Pop()
}
