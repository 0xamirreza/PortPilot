package gui

import (
	"context"
	"image"
	"strconv"
	"time"

	gioapp "gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/0xamirreza/portpilot/assets/brand"
	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/config"
	"github.com/0xamirreza/portpilot/internal/core/ports"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
	"github.com/0xamirreza/portpilot/internal/logging"
)

// Screen order matches the rail.
var screenNames = []string{
	"Overview", "Ports", "Processes", "Firewall", "Rules",
	"Services", "Containers", "Activity", "Topology", "Diag", "Settings",
}

// UI is the immediate-mode window state.
type UI struct {
	app                *app.App
	th                 *theme.Theme
	screen             int
	query              ports.Query
	nav                [11]widget.Clickable
	navList            widget.List
	rescan             widget.Clickable
	authBtn            widget.Clickable
	theme              widget.Clickable
	search             widget.Editor
	chips              [9]widget.Clickable
	sockRow            []widget.Clickable
	procRow            []widget.Clickable
	sockList           widget.List
	procList           widget.List
	actList            widget.List
	diagList           widget.List
	diagDetailList     widget.List
	diagSearch         widget.Editor
	diagLevel          int // 0: ALL, 1: INFO, 2: WARN, 3: ERROR, 4: DEBUG
	diagLevelBtns      [5]widget.Clickable
	diagPauseBtn       widget.Clickable
	diagPaused         bool
	diagFrozenEntries  []logging.Entry
	diagClearBtn       widget.Clickable
	diagRotateBtn      widget.Clickable
	diagSelectedEntry  *logging.Entry
	diagRowBtns        []widget.Clickable
	diagBackBtn        widget.Clickable
	diagRawEditor      widget.Editor
	diagJsonEditor     widget.Editor
	logRetentionDraft  int
	logRetentionEditor widget.Editor
	logRotateNowBtn    widget.Clickable
	logPresetBtns      [5]widget.Clickable
	svcList       widget.List
	fwList        widget.List
	boxList            widget.List
	topoList           widget.List
	topoSelectedHubKey string
	sourceBtn          []widget.Clickable
	topoBackBtn        widget.Clickable
	topoTab            int // 0: Process Hubs, 1: World Geolocation
	topoTabBtns        [2]widget.Clickable
	geoList            widget.List
	geoSearch          widget.Editor
	geoSelectedCountry string
	geoScopeFilter     string // "", "FOREIGN", "INTERNAL"
	geoScopeBtns       [3]widget.Clickable
	geoCountryBtns     []widget.Clickable
	geoCountryList     widget.List
	geoZoomInBtn       widget.Clickable
	geoZoomOutBtn      widget.Clickable
	geoResetBtn        widget.Clickable
	geoFocusHostBtn    widget.Clickable
	geoGlobeDrag       gesture.Drag
	geoDragLast        f32.Point
	geoZoom            float64
	geoPanX            float64
	geoPanY            float64
	geoAnimTick        float64
	geoLastFrame       time.Time
	livePauseBtn       widget.Clickable
	liveClearBtn       widget.Clickable
	livePaused         bool
	livePackets        []LivePacketEvent
	liveList           widget.List
	livePrevStats      map[string]socketStats
	liveLastSample     time.Time
	liveTxRate         float64
	liveRxRate         float64
	ifaceList          widget.List
	ifaceRow           []widget.Clickable
	fwRow              []widget.Clickable
	selectedSock  string
	selectedProc  int
	selectedRule  string
	back          widget.Clickable
	saveBtn       widget.Clickable
	themeDraft    [2]widget.Clickable
	draftTheme    string
	settingsDirty bool
	focused       bool
	authorize     widget.Clickable
	reconcile     widget.Clickable
	rollback      widget.Clickable
	allowBtn      widget.Clickable
	denyBtn       widget.Clickable
	rejectBtn     widget.Clickable
	deleteBtn     widget.Clickable
	durBtn        [7]widget.Clickable
	protoBtn      [2]widget.Clickable
	famBtn        [3]widget.Clickable
	portEdit      widget.Editor
	sourceEdit    widget.Editor
	ifaceEdit     widget.Editor
	customEdit    widget.Editor
	actFilter     widget.Editor
	family        string
	logo          paint.ImageOp
	disableBtn    widget.Clickable
	procSvc       [2]widget.Clickable
	ownedRow      []widget.Clickable
	termBtn       widget.Clickable
	killBtn       widget.Clickable
	svcBtn        [5]widget.Clickable
	svcRow        []widget.Clickable
	selectedSvc   string
	svcSearch     widget.Editor
	svcFilter     int // 0: ALL, 1: RUNNING, 2: INACTIVE, 3: FAILED
	svcFilterBtns [4]widget.Clickable
	procSearch    widget.Editor
	procFilter    int // 0: ALL, 1: USER, 2: ROOT, 3: SERVICES
	procFilterBtns [4]widget.Clickable
	procDetailList widget.List
	protocol      string
	duration      int
	confirm       *confirm
	yes           widget.Clickable
	no            widget.Clickable
	iconTries     int
	iconSet       bool
}

// Run opens the native window and blocks until it closes.
func Run(application *app.App) error {
	th, err := theme.Load()
	if err != nil {
		return err
	}
	ui := &UI{
		app:          application,
		th:           th,
		query:        ports.DefaultQuery(),
		selectedProc: -1,
		protocol:     "tcp",
		family:       "both",
		geoZoom:      1.0,
	}
	ui.search.SingleLine = true
	ui.portEdit.SingleLine = true
	ui.sourceEdit.SingleLine = true
	ui.ifaceEdit.SingleLine = true
	ui.customEdit.SingleLine = true
	ui.actFilter.SingleLine = true
	ui.svcSearch.SingleLine = true
	ui.procSearch.SingleLine = true
	if img, err := brand.ImageAt(256); err == nil {
		ui.logo = paint.NewImageOp(img)
	} else if img, err := brand.Image(); err == nil {
		ui.logo = paint.NewImageOp(img)
	}
	ui.sockList.Axis = layout.Vertical
	ui.procList.Axis = layout.Vertical
	ui.procDetailList.Axis = layout.Vertical
	ui.actList.Axis = layout.Vertical
	ui.svcList.Axis = layout.Vertical
	ui.fwList.Axis = layout.Vertical
	ui.boxList.Axis = layout.Vertical
	ui.navList.Axis = layout.Vertical
	ui.topoList.Axis = layout.Vertical
	ui.liveList.Axis = layout.Vertical
	ui.ifaceList.Axis = layout.Vertical
	ui.diagList.Axis = layout.Vertical
	ui.diagDetailList.Axis = layout.Vertical
	ui.diagRawEditor.ReadOnly = true
	ui.diagJsonEditor.ReadOnly = true

	w := new(gioapp.Window)
	application.SetInvalidate(w.Invalidate)
	w.Option(
		gioapp.Title("PortPilot"),
		gioapp.Size(unit.Dp(1280), unit.Dp(800)),
		gioapp.MinSize(unit.Dp(960), unit.Dp(640)),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go application.Loop(ctx, w.Invalidate)
	go func() {
		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				w.Invalidate()
			}
		}
	}()
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case gioapp.DestroyEvent:
			cancel()
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			ui.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (ui *UI) layout(gtx layout.Context) layout.Dimensions {
	ui.publishIcon()
	view := ui.app.Store().Copy()
	ui.th.Use(view.Theme != config.ThemeLight)
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, ui)
	ui.keys(gtx)
	paint.Fill(gtx.Ops, ui.th.Void)
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(176))
			gtx.Constraints.Max.X = gtx.Dp(unit.Dp(176))
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return ui.rail(gtx, view)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if ui.confirm != nil {
				return ui.confirmPlate(gtx)
			}
			return ui.body(gtx, view)
		}),
	)
}

func (ui *UI) keys(gtx layout.Context) {
	filters := []event.Filter{key.FocusFilter{Target: ui}}
	names := []key.Name{"1", "2", "3", "4", "5", "6", "7", "8", "9", "R", "T", "J", "K", "F"}
	for _, name := range names {
		filters = append(filters, key.Filter{Name: name, Required: key.ModShortcut})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		switch e.Name {
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			n, _ := strconv.Atoi(string(e.Name))
			ui.screen = n - 1
		case "R":
			ui.app.Store().RequestRescan()
		case "T":
			ui.toggleTheme(ui.app.Store().Copy())
		case "J":
			ui.move(1)
		case "K":
			ui.move(-1)
		case "F":
			gtx.Execute(key.FocusCmd{Tag: &ui.search})
		}
	}
	if !ui.focused {
		gtx.Execute(key.FocusCmd{Tag: ui})
		ui.focused = true
	}
}

func (ui *UI) toggleTheme(view app.View) {
	next := config.ThemeDark
	if view.Theme != config.ThemeLight {
		next = config.ThemeLight
	}
	_ = ui.app.SetTheme(next)
}

func (ui *UI) move(delta int) {
	switch ui.screen {
	case 1:
		rows := ui.visibleSockets(ui.app.Store().Copy())
		if len(rows) == 0 {
			return
		}
		idx := 0
		for i, s := range rows {
			if s.Key() == ui.selectedSock {
				idx = i
				break
			}
		}
		idx += delta
		if idx < 0 {
			idx = 0
		}
		if idx >= len(rows) {
			idx = len(rows) - 1
		}
		ui.selectedSock = rows[idx].Key()
	case 2:
		procs := ui.app.Store().Copy().Processes
		if len(procs) == 0 {
			return
		}
		curr := -1
		for i, p := range procs {
			if p.PID == ui.selectedProc {
				curr = i
				break
			}
		}
		if curr < 0 {
			curr = 0
		} else {
			curr += delta
			if curr < 0 {
				curr = 0
			}
			if curr >= len(procs) {
				curr = len(procs) - 1
			}
		}
		ui.selectedProc = procs[curr].PID
	}
}

func (ui *UI) mark(gtx layout.Context) layout.Dimensions {
	side := gtx.Constraints.Max.X
	if side <= 0 {
		side = gtx.Dp(unit.Dp(152))
	}
	radius := side / 8
	if ui.logo.Size().X == 0 {
		return layout.Dimensions{Size: image.Pt(side, side)}
	}
	defer clip.RRect{Rect: image.Rect(0, 0, side, side), NE: radius, NW: radius, SE: radius, SW: radius}.Push(gtx.Ops).Pop()
	scale := float32(side) / float32(ui.logo.Size().X)
	defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(scale, scale))).Push(gtx.Ops).Pop()
	ui.logo.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(side, side)}
}

func (ui *UI) rail(gtx layout.Context, view app.View) layout.Dimensions {
	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.mark(gtx)
						}),
						layout.Rigid(gap(unit.Dp(10))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(16), font.Bold, theme.Sans, ui.th.Ink, "PORTPILOT")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, "LINUX PORTS")
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return ui.vlist(gtx, &ui.navList, len(screenNames), func(gtx layout.Context, i int) layout.Dimensions {
					if ui.nav[i].Clicked(gtx) {
						ui.screen = i
					}
					return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.button(gtx, &ui.nav[i], screenNames[i], ui.screen == i)
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, view.Host.Hostname)
						}),
						layout.Rigid(gap(unit.Dp(6))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if ui.theme.Clicked(gtx) {
								ui.toggleTheme(view)
							}
							label := "THEME DARK"
							if view.Theme == config.ThemeLight {
								label = "THEME LIGHT"
							}
							return ui.button(gtx, &ui.theme, label, false)
						}),
					)
				})
			}),
		)
	})
}

func (ui *UI) body(gtx layout.Context, view app.View) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.header(gtx, view)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if view.Banner == "" {
				return layout.Dimensions{}
			}
			return plate(gtx, ui.th.Signal, ui.th.Signal, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.wrap(gtx, unit.Sp(13), ui.th.Void, view.Banner)
				})
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				switch ui.screen {
				case 0:
					return ui.overview(gtx, view)
				case 1:
					return ui.ports(gtx, view)
				case 2:
					return ui.processes(gtx, view)
				case 3:
					return ui.firewallPage(gtx, view, false)
				case 4:
					return ui.firewallPage(gtx, view, true)
				case 5:
					return ui.services(gtx, view)
				case 6:
					return ui.containers(gtx, view)
				case 7:
					return ui.activity(gtx, view)
				case 8:
					return ui.topology(gtx, view)
				case 9:
					return ui.diag(gtx, view)
				default:
					return ui.settings(gtx, view)
				}
			})
		}),
	)
}

func (ui *UI) header(gtx layout.Context, view app.View) layout.Dimensions {
	if ui.authBtn.Clicked(gtx) && ui.app != nil {
		ui.app.ToggleAuthorization()
	}
	statusCol := ui.th.Olive
	statusText := scanClock(time.Now(), view.ScannedAt, view.NextScan, view.Scanning)
	if view.ScannedAt.IsZero() {
		statusCol = ui.th.Mute
	}
	if view.Source != "" && !view.ScannedAt.IsZero() {
		statusText += "  (" + view.Source + ")"
	}
	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(20), font.Bold, theme.Sans, ui.th.Ink, screenNames[ui.screen])
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, statusCol, statusText)
				}),
				layout.Rigid(hgap(unit.Dp(12))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.authButton(gtx, &ui.authBtn, view.Authorized)
				}),
			)
		})
	})
}

func (ui *UI) notice(gtx layout.Context, text string) layout.Dimensions {
	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.wrap(gtx, unit.Sp(15), ui.th.Ink, text)
		})
	})
}
