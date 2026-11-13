package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
	"github.com/0xamirreza/portpilot/internal/logging"
)

var diagLevelLabels = []string{"ALL", "INFO", "WARN", "ERROR", "DEBUG"}

func (ui *UI) diag(gtx layout.Context, view app.View) layout.Dimensions {
	if ui.diagSelectedEntry != nil {
		return ui.diagDetailView(gtx, *ui.diagSelectedEntry)
	}

	if ui.diagPauseBtn.Clicked(gtx) {
		ui.diagPaused = !ui.diagPaused
		if ui.diagPaused {
			ui.diagFrozenEntries = logging.DefaultStore.All()
		} else {
			ui.diagFrozenEntries = nil
		}
	}

	if ui.diagClearBtn.Clicked(gtx) {
		_ = logging.DefaultStore.Clear()
		ui.diagFrozenEntries = nil
	}
	if ui.diagRotateBtn.Clicked(gtx) {
		if ui.app != nil {
			_ = ui.app.RotateLogs()
		}
	}
	for i := range ui.diagLevelBtns {
		if ui.diagLevelBtns[i].Clicked(gtx) {
			ui.diagLevel = i
		}
	}

	var allEntries []logging.Entry
	if ui.diagPaused && len(ui.diagFrozenEntries) > 0 {
		allEntries = ui.diagFrozenEntries
	} else {
		allEntries = logging.DefaultStore.All()
	}

	// Count statistics
	totalCount := len(allEntries)
	infoCount, warnCount, errorCount, debugCount := 0, 0, 0, 0
	for _, e := range allEntries {
		switch e.Level {
		case slog.LevelInfo:
			infoCount++
		case slog.LevelWarn:
			warnCount++
		case slog.LevelError:
			errorCount++
		case slog.LevelDebug:
			debugCount++
		}
	}

	// Filter entries
	searchText := strings.ToLower(strings.TrimSpace(ui.diagSearch.Text()))
	var filtered []logging.Entry
	for _, e := range allEntries {
		if ui.diagLevel == 1 && e.Level != slog.LevelInfo {
			continue
		} else if ui.diagLevel == 2 && e.Level != slog.LevelWarn {
			continue
		} else if ui.diagLevel == 3 && e.Level != slog.LevelError {
			continue
		} else if ui.diagLevel == 4 && e.Level != slog.LevelDebug {
			continue
		}
		if searchText != "" {
			match := strings.Contains(strings.ToLower(e.Message), searchText) ||
				strings.Contains(strings.ToLower(e.Raw), searchText)
			if !match {
				for k, v := range e.Attrs {
					if strings.Contains(strings.ToLower(k), searchText) || strings.Contains(strings.ToLower(v), searchText) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	if len(ui.diagRowBtns) < len(filtered) {
		ui.diagRowBtns = make([]widget.Clickable, len(filtered))
	}

	// Log file size
	fileSizeStr := "0 B"
	if view.LogPath != "" {
		if fi, err := os.Stat(view.LogPath); err == nil {
			fileSizeStr = formatOctets(uint64(fi.Size()))
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Search & Level Filters
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					ed := material.Editor(ui.th.Material, &ui.diagSearch, "Search logs by message or field...")
					ed.TextSize = unit.Sp(14)
					ed.Color = ui.th.Ink
					ed.HintColor = ui.th.Mute
					ed.Font.Typeface = theme.Mono
					return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, ed.Layout)
					})
				}),
				layout.Rigid(hgap(unit.Dp(12))),
				// Level Buttons
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					kids := make([]layout.FlexChild, 0, len(diagLevelLabels))
					for i, lbl := range diagLevelLabels {
						i, lbl := i, lbl
						kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.diagLevelBtns[i], lbl, ui.diagLevel == i)
						}))
						if i < len(diagLevelLabels)-1 {
							kids = append(kids, layout.Rigid(hgap(unit.Dp(6))))
						}
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
				}),
				layout.Rigid(hgap(unit.Dp(12))),
				// Actions
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					pauseText := "PAUSE"
					if ui.diagPaused {
						pauseText = "RESUME"
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.diagPauseBtn, pauseText, ui.diagPaused)
						}),
						layout.Rigid(hgap(unit.Dp(6))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.diagRotateBtn, "ROTATE", false)
						}),
						layout.Rigid(hgap(unit.Dp(6))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.diagClearBtn, "CLEAR", false)
						}),
					)
				}),
			)
		}),
		layout.Rigid(gap(unit.Dp(10))),

		// Stats Summary Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Ink, fmt.Sprintf("TOTAL: %d", totalCount))
						}),
						layout.Rigid(hgap(unit.Dp(16))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, fmt.Sprintf("INFO: %d", infoCount))
						}),
						layout.Rigid(hgap(unit.Dp(16))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Stamp, fmt.Sprintf("WARN: %d", warnCount))
						}),
						layout.Rigid(hgap(unit.Dp(16))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Signal, fmt.Sprintf("ERROR: %d", errorCount))
						}),
						layout.Rigid(hgap(unit.Dp(16))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("DEBUG: %d", debugCount))
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if ui.diagPaused {
								return layout.Inset{Left: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Stamp, "● STREAM PAUSED")
								})
							}
							return layout.Dimensions{}
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								fileInfo := "File: " + view.LogPath + " (" + fileSizeStr + ")"
								return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, fileInfo)
							})
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// Log Table Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(165))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(165))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "TIMESTAMP")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(65))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(65))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "LEVEL")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "MESSAGE & STRUCTURED ATTRIBUTES")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(115))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(115))
							return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "ACTION")
							})
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(4))),

		// Log Stream List
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(filtered) == 0 {
				return ui.notice(gtx, "No diagnostic logs matching current filter. Events logged by PortPilot will stream here in real time.")
			}
			return ui.vlist(gtx, &ui.diagList, len(filtered), func(gtx layout.Context, i int) layout.Dimensions {
				idx := i
				if ui.diagRowBtns[idx].Clicked(gtx) {
					ui.selectLogEntry(&filtered[idx])
				}
				return layout.Inset{Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.diagRow(gtx, filtered[idx], &ui.diagRowBtns[idx])
				})
			})
		}),
	)
}

func (ui *UI) selectLogEntry(e *logging.Entry) {
	ui.diagSelectedEntry = e
	if e != nil {
		ui.diagRawEditor.SetText(e.Raw)
		ui.diagJsonEditor.SetText(entryToJSON(*e))
	}
}

func entryToJSON(e logging.Entry) string {
	m := map[string]any{
		"timestamp": e.Timestamp.Format(time.RFC3339Nano),
		"level":     e.Level.String(),
		"message":   e.Message,
		"attrs":     e.Attrs,
		"raw":       e.Raw,
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (ui *UI) diagRow(gtx layout.Context, e logging.Entry, btn *widget.Clickable) layout.Dimensions {
	gtx.Constraints.Min.Y = 0

	levelCol := ui.th.Olive
	levelBg := ui.th.Raised
	levelText := "INFO"

	switch e.Level {
	case slog.LevelWarn:
		levelCol = ui.th.Stamp
		levelText = "WARN"
	case slog.LevelError:
		levelCol = ui.th.Signal
		levelText = "ERROR"
	case slog.LevelDebug:
		levelCol = ui.th.Mute
		levelText = "DEBUG"
	}

	timeStr := e.Timestamp.Format("2006-01-02 15:04:05.000")

	// Sort attribute keys alphabetically for consistent, readable layout
	var keys []string
	for k := range e.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var attrParts []string
	for _, k := range keys {
		attrParts = append(attrParts, k+"="+e.Attrs[k])
	}
	attrsStr := strings.Join(attrParts, "  ")

	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Timestamp
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(165))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(165))
					return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, timeStr)
				}),
				// Level Badge
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(65))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(65))
					return plate(gtx, levelBg, levelCol, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, levelCol, levelText)
						})
					})
				}),
				layout.Rigid(hgap(unit.Dp(12))),
				// Message & Attributes (Flexed so it fills all available middle space)
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, ui.th.Ink, e.Message)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if attrsStr == "" {
								return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
							}
							return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, attrsStr)
							})
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(12))),
				// View Full button - Exactly pinned to the far right, matching header ACTION column
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(115))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(115))
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.viewFullButton(gtx, btn)
					})
				}),
			)
		})
	})
}

func (ui *UI) diagDetailView(gtx layout.Context, e logging.Entry) layout.Dimensions {
	if ui.diagBackBtn.Clicked(gtx) {
		ui.diagSelectedEntry = nil
		return layout.Dimensions{}
	}

	levelCol := ui.th.Olive
	levelBg := ui.th.Raised
	levelText := "INFO"
	switch e.Level {
	case slog.LevelWarn:
		levelCol = ui.th.Stamp
		levelText = "WARN"
	case slog.LevelError:
		levelCol = ui.th.Signal
		levelText = "ERROR"
	case slog.LevelDebug:
		levelCol = ui.th.Mute
		levelText = "DEBUG"
	}

	var keys []string
	for k := range e.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	attrCount := len(keys)
	attrRows := attrCount
	if attrRows == 0 {
		attrRows = 1
	}

	totalItems := 1 + 1 + attrRows + 1 + 1 + 1 + 1 // top, attr-header, attrs, raw-header, raw-editor, json-header, json-editor

	return ui.vlist(gtx, &ui.diagDetailList, totalItems, func(gtx layout.Context, i int) layout.Dimensions {
		switch {
		case i == 0:
			// Top Navigation & Header Bar
			return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &ui.diagBackBtn, "< BACK TO LOG LIST", false)
					}),
					layout.Rigid(hgap(unit.Dp(16))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return plate(gtx, levelBg, levelCol, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, levelCol, levelText)
							})
						})
					}),
					layout.Rigid(hgap(unit.Dp(12))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(16), font.Bold, theme.Sans, ui.th.Ink, "Event: "+e.Message)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ui.th.Mute, e.Timestamp.Format("2006-01-02 15:04:05.000 MST (-07:00)"))
						})
					}),
				)
			})

		case i == 1:
			// Section: STRUCTURED ATTRIBUTES
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, ui.th.Mute, "STRUCTURED ATTRIBUTES")
			})

		case i >= 2 && i < 2+attrRows:
			attrIdx := i - 2
			if attrCount == 0 {
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.notice(gtx, "No structured attributes for this log event.")
				})
			}
			k := keys[attrIdx]
			v := e.Attrs[k]
			return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
								gtx.Constraints.Max.X = gtx.Dp(unit.Dp(160))
								return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Olive, k)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return ui.wrap(gtx, unit.Sp(12), ui.th.Ink, v)
							}),
						)
					})
				})
			})

		case i == 2+attrRows:
			// Section: RAW LOG LINE
			return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, ui.th.Mute, "RAW LOG LINE (SELECTABLE)")
			})

		case i == 3+attrRows:
			// Editor for RAW
			return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(ui.th.Material, &ui.diagRawEditor, "")
				ed.TextSize = unit.Sp(12)
				ed.Color = ui.th.Ink
				ed.Font.Typeface = theme.Mono
				return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, ed.Layout)
				})
			})

		case i == 4+attrRows:
			// Section: FORMATTED JSON
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, ui.th.Mute, "FORMATTED JSON (SELECTABLE)")
			})

		case i == 5+attrRows:
			// Editor for JSON
			return layout.Inset{Bottom: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(ui.th.Material, &ui.diagJsonEditor, "")
				ed.TextSize = unit.Sp(12)
				ed.Color = ui.th.Ink
				ed.Font.Typeface = theme.Mono
				return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, ed.Layout)
				})
			})
		}
		return layout.Dimensions{}
	})
}
