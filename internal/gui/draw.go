package gui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/0xamirreza/portpilot/internal/gui/theme"
)

func fill(ops *op.Ops, r image.Rectangle, c color.NRGBA) {
	defer clip.Rect(r).Push(ops).Pop()
	paint.ColorOp{Color: c}.Add(ops)
	paint.PaintOp{}.Add(ops)
}

func plate(gtx layout.Context, fillC, line color.NRGBA, w layout.Widget) layout.Dimensions {
	rec := op.Record(gtx.Ops)
	dims := w(gtx)
	call := rec.Stop()
	sz := dims.Size
	fill(gtx.Ops, image.Rectangle{Max: sz}, fillC)
	fill(gtx.Ops, image.Rect(0, 0, sz.X, 1), line)
	fill(gtx.Ops, image.Rect(0, max0(sz.Y-1), sz.X, sz.Y), line)
	fill(gtx.Ops, image.Rect(0, 0, 1, sz.Y), line)
	fill(gtx.Ops, image.Rect(max0(sz.X-1), 0, sz.X, sz.Y), line)
	call.Add(gtx.Ops)
	return dims
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// vlist lays out a vertical list and reserves a visible scrollbar track.
func (ui *UI) vlist(gtx layout.Context, list *widget.List, n int, el layout.ListElement) layout.Dimensions {
	list.Axis = layout.Vertical
	style := material.List(ui.th.Material, list)
	style.AnchorStrategy = material.Occupy
	style.Indicator.MinorWidth = unit.Dp(10)
	style.Indicator.CornerRadius = 0
	style.Indicator.Color = ui.th.Stamp
	style.Indicator.HoverColor = ui.th.Signal
	style.Track.Color = ui.th.Rule
	style.Track.MinorPadding = unit.Dp(2)
	style.Track.MajorPadding = unit.Dp(2)
	return style.Layout(gtx, n, el)
}

func (ui *UI) label(gtx layout.Context, size unit.Sp, weight font.Weight, face font.Typeface, col color.NRGBA, txt string) layout.Dimensions {
	l := material.Label(ui.th.Material, size, txt)
	l.Color = col
	l.Font = font.Font{Typeface: face, Weight: weight}
	l.MaxLines = 1
	return l.Layout(gtx)
}

func (ui *UI) wrap(gtx layout.Context, size unit.Sp, col color.NRGBA, txt string) layout.Dimensions {
	l := material.Label(ui.th.Material, size, txt)
	l.Color = col
	l.Font = font.Font{Typeface: theme.Sans}
	return l.Layout(gtx)
}

func (ui *UI) button(gtx layout.Context, click *widget.Clickable, txt string, on bool) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Ink, ui.th.Rule
		if on || click.Pressed() {
			bg, fg, line = ui.th.Ink, ui.th.Void, ui.th.Ink
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, fg, txt)
			})
		})
	})
}

func (ui *UI) dangerButton(gtx layout.Context, click *widget.Clickable, txt string) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Signal, ui.th.Signal
		if click.Pressed() {
			bg, fg, line = ui.th.Signal, ui.th.Void, ui.th.Signal
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, fg, txt)
			})
		})
	})
}

func (ui *UI) successButton(gtx layout.Context, click *widget.Clickable, txt string) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Olive, ui.th.Olive
		if click.Pressed() {
			bg, fg, line = ui.th.Olive, ui.th.Void, ui.th.Olive
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, fg, txt)
			})
		})
	})
}

func (ui *UI) warningButton(gtx layout.Context, click *widget.Clickable, txt string) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Stamp, ui.th.Stamp
		if click.Pressed() {
			bg, fg, line = ui.th.Stamp, ui.th.Void, ui.th.Stamp
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, fg, txt)
			})
		})
	})
}

func drawLockIcon(gtx layout.Context, col color.NRGBA, locked bool) layout.Dimensions {
	w := gtx.Dp(unit.Dp(13))
	h := gtx.Dp(unit.Dp(15))

	bodyW := gtx.Dp(unit.Dp(11))
	bodyH := gtx.Dp(unit.Dp(8))
	bodyTop := h - bodyH
	bodyLeft := (w - bodyW) / 2

	// Body of the lock
	fill(gtx.Ops, image.Rect(bodyLeft, bodyTop, bodyLeft+bodyW, h), col)

	// Shackle (arch)
	shackleW := gtx.Dp(unit.Dp(7))
	shackleH := gtx.Dp(unit.Dp(6))
	stroke := gtx.Dp(unit.Dp(1.5))
	if stroke < 1 {
		stroke = 1
	}
	shackleLeft := (w - shackleW) / 2
	shackleTop := bodyTop - shackleH + stroke
	if !locked {
		shackleTop -= gtx.Dp(unit.Dp(2))
		shackleLeft += gtx.Dp(unit.Dp(2))
	}

	// Shackle top bar
	fill(gtx.Ops, image.Rect(shackleLeft, shackleTop, shackleLeft+shackleW, shackleTop+stroke), col)
	// Shackle left post
	fill(gtx.Ops, image.Rect(shackleLeft, shackleTop, shackleLeft+stroke, bodyTop), col)
	// Shackle right post
	rightPostBottom := bodyTop
	if !locked {
		rightPostBottom = shackleTop + gtx.Dp(unit.Dp(3))
	}
	fill(gtx.Ops, image.Rect(shackleLeft+shackleW-stroke, shackleTop, shackleLeft+shackleW, rightPostBottom), col)

	return layout.Dimensions{Size: image.Pt(w, h)}
}

func (ui *UI) authButton(gtx layout.Context, click *widget.Clickable, authorized bool) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Ink, ui.th.Rule
		if authorized || click.Pressed() {
			bg, fg, line = ui.th.Ink, ui.th.Void, ui.th.Ink
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		label := "AUTHORIZE"
		if authorized {
			label = "ROOT (ACTIVE)"
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return drawLockIcon(gtx, fg, !authorized)
					}),
					layout.Rigid(hgap(unit.Dp(8))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, fg, label)
					}),
				)
			})
		})
	})
}

func drawDocIcon(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	w := gtx.Dp(unit.Dp(10))
	h := gtx.Dp(unit.Dp(12))

	fill(gtx.Ops, image.Rect(0, 0, w, h), col)
	fill(gtx.Ops, image.Rect(1, 1, w-1, h-1), color.NRGBA{0, 0, 0, 0})

	lineH := 1
	if gtx.Dp(unit.Dp(1)) > 1 {
		lineH = gtx.Dp(unit.Dp(1))
	}
	fill(gtx.Ops, image.Rect(2, 3, w-2, 3+lineH), col)
	fill(gtx.Ops, image.Rect(2, 5, w-2, 5+lineH), col)
	fill(gtx.Ops, image.Rect(2, 7, w-4, 7+lineH), col)

	return layout.Dimensions{Size: image.Pt(w, h)}
}

func (ui *UI) viewFullButton(gtx layout.Context, click *widget.Clickable) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, fg, line := ui.th.Plate, ui.th.Ink, ui.th.Rule
		if click.Pressed() {
			bg, fg, line = ui.th.Ink, ui.th.Void, ui.th.Ink
		} else if click.Hovered() {
			bg = ui.th.Raised
		}
		return plate(gtx, bg, line, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return drawDocIcon(gtx, fg)
					}),
					layout.Rigid(hgap(unit.Dp(6))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Sans, fg, "VIEW FULL")
					}),
				)
			})
		})
	})
}

func (ui *UI) inset(top, right, bottom, left unit.Dp, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: top, Right: right, Bottom: bottom, Left: left}.Layout(gtx, w)
	}
}

func gap(h unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(h))}
	}
}

func hgap(w unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Dp(w), gtx.Constraints.Min.Y)}
	}
}
