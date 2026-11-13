package gui

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/core/network/geoip"
	"github.com/0xamirreza/portpilot/internal/core/ports"
	"github.com/0xamirreza/portpilot/internal/discovery/netlink"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
)

// LivePacketEvent represents one observed real-time traffic or packet event.
type LivePacketEvent struct {
	Timestamp time.Time
	Direction string // "TX", "RX", "STATE", "INIT"
	Proto     string
	Local     string
	Remote    string
	Bytes     uint64
	Packets   uint32
	TotalSent uint64
	TotalRecv uint64
	RTT       uint32 // in microseconds
	State     string
	Summary   string
}

type socketStats struct {
	BytesSent uint64
	BytesRecv uint64
	SegsOut   uint32
	SegsIn    uint32
	State     string
	SeenAt    time.Time
}

func (ui *UI) topology(gtx layout.Context, view app.View) layout.Dimensions {
	// If a source hub is selected, drill down into its live traffic & packet log
	if ui.topoSelectedHubKey != "" {
		allHubs := ports.Topology(view.Sockets)
		for _, h := range allHubs {
			if h.Key == ui.topoSelectedHubKey {
				return ui.sourceLiveView(gtx, h, view)
			}
		}
		// Connection may have terminated: still allow inspection of logs
		return ui.sourceLiveView(gtx, ports.Hub{
			Key:   ui.topoSelectedHubKey,
			Title: "Connection closed or idle",
		}, view)
	}

	for i := range ui.topoTabBtns {
		if ui.topoTabBtns[i].Clicked(gtx) {
			ui.topoTab = i
		}
	}

	tabBar := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.iconButton(gtx, &ui.topoTabBtns[0], drawIconNetwork, "PROCESS HUBS", ui.topoTab == 0)
			}),
			layout.Rigid(hgap(unit.Dp(8))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.iconButton(gtx, &ui.topoTabBtns[1], drawIconMap, "WORLD GEOLOCATION", ui.topoTab == 1)
			}),
		)
	}

	if ui.topoTab == 1 {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(tabBar),
			layout.Rigid(gap(unit.Dp(10))),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return ui.worldGeolocationView(gtx, view)
			}),
		)
	}

	q := ports.Query{
		Text: ui.search.Text(),
		TCP:  true, UDP: true,
		IPv4: true, IPv6: true,
		Localhost: true, Wildcard: true, Specific: true,
		Listening: true, Active: true,
	}
	hubs := ports.Topology(ports.Apply(view.Sockets, q))

	var sent, recv uint64
	known, peers := 0, 0
	for _, h := range hubs {
		peers += len(h.Peers)
		if h.Known {
			known++
			sent += h.Sent
			recv += h.Recv
		}
	}

	if len(ui.sourceBtn) < len(hubs) {
		ui.sourceBtn = make([]widget.Clickable, len(hubs))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(tabBar),
		layout.Rigid(gap(unit.Dp(8))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ed := material.Editor(ui.th.Material, &ui.search, "Filter by process, local port, or destination")
			ed.TextSize = unit.Sp(14)
			ed.Color = ui.th.Ink
			ed.HintColor = ui.th.Mute
			ed.Font.Typeface = theme.Mono
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, ed.Layout)
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			line := strconv.Itoa(len(hubs)) + " sources    " + strconv.Itoa(peers) + " destinations    sent " + formatOctets(sent) + "    received " + formatOctets(recv)
			if known != len(hubs) {
				line += "    some sockets have no byte counter"
			}
			return ui.wrap(gtx, unit.Sp(13), ui.th.Mute, line)
		}),
		layout.Rigid(gap(unit.Dp(8))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.graphLegend(gtx)
		}),
		layout.Rigid(gap(unit.Dp(8))),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(hubs) == 0 {
				return ui.notice(gtx, "No socket in this scan has a destination. A listening port appears here after something connects.")
			}
			return ui.vlist(gtx, &ui.topoList, len(hubs), func(gtx layout.Context, i int) layout.Dimensions {
				if ui.sourceBtn[i].Clicked(gtx) {
					ui.openLiveSource(hubs[i])
				}
				return layout.Inset{Bottom: unit.Dp(10), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.flowGraph(gtx, hubs[i], &ui.sourceBtn[i])
				})
			})
		}),
	)
}

func (ui *UI) openLiveSource(h ports.Hub) {
	ui.topoSelectedHubKey = h.Key
	ui.livePackets = nil
	ui.livePrevStats = make(map[string]socketStats)
	ui.liveLastSample = time.Time{}
	ui.liveTxRate = 0
	ui.liveRxRate = 0
	ui.livePaused = false
}

func (ui *UI) graphLegend(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, ui.th.Ink, "PROCESSES & SERVICES (CLICK 'LIVE PACKETS' ON ANY CARD TO INSPECT)")
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(12), font.Bold, theme.Sans, ui.th.Signal, "DATA FLOW: LOCAL PORT -> REMOTE PEER")
			})
		}),
	)
}

func (ui *UI) flowGraph(gtx layout.Context, h ports.Hub, btn *widget.Clickable) layout.Dimensions {
	borderColor := ui.th.Rule
	if btn.Hovered() {
		borderColor = ui.th.Signal
	}
	sentText := "byte count unavailable"
	if h.Known {
		sentText = "▲ " + formatOctets(h.Sent) + "  ▼ " + formatOctets(h.Recv)
	}

	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return plate(gtx, ui.th.Plate, borderColor, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// Header of the Source card
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							// Process Name
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, unit.Sp(17), font.Bold, theme.Sans, ui.th.Ink, h.Title)
							}),
							layout.Rigid(hgap(unit.Dp(10))),
							// Detail / PID badge
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, h.Detail)
									})
								})
							}),
							layout.Rigid(hgap(unit.Dp(8))),
							// Flows count badge
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										flows := strconv.Itoa(len(h.Peers)) + " destinations"
										if len(h.Peers) == 1 {
											flows = "1 destination"
										}
										return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Ink, flows)
									})
								})
							}),
							layout.Rigid(hgap(unit.Dp(12))),
							// Bandwidth summary
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Stamp, sentText)
							}),
							// Right side: Live packets badge
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									badgeBg := ui.th.Raised
									badgeBorder := ui.th.Rule
									badgeTextCol := ui.th.Ink
									if btn.Hovered() {
										badgeBg = ui.th.Signal
										badgeBorder = ui.th.Signal
										badgeTextCol = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
									}
									return plate(gtx, badgeBg, badgeBorder, func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, badgeTextCol, "LIVE PACKETS >")
										})
									})
								})
							}),
						)
					}),
					layout.Rigid(gap(unit.Dp(8))),
					// Separator line
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						h := gtx.Dp(unit.Dp(1))
						fill(gtx.Ops, image.Rect(0, 0, gtx.Constraints.Max.X, h), ui.th.Rule)
						return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
					}),
					layout.Rigid(gap(unit.Dp(6))),
					// Flows
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						maxPeers := 8
						shown := h.Peers
						if len(shown) > maxPeers {
							shown = shown[:maxPeers]
						}
						kids := make([]layout.FlexChild, 0, len(shown)+1)
						for _, peer := range shown {
							peer := peer
							kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.flowRow(gtx, peer)
								})
							}))
						}
						if len(h.Peers) > maxPeers {
							kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								more := fmt.Sprintf("+ %d more active destinations (click card to view all in live monitor)", len(h.Peers)-maxPeers)
								return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(11), font.Normal, theme.Sans, ui.th.Mute, more)
								})
							}))
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
					}),
				)
			})
		})
	})
}

func (ui *UI) flowRow(gtx layout.Context, peer ports.Peer) layout.Dimensions {
	localText := ":" + strconv.Itoa(int(peer.LocalPort))
	if peer.LocalPort == 0 {
		localText = "local"
	}

	whereText := "EXTERNAL"
	whereCol := ui.th.Signal
	if peer.OnHost {
		whereText = "ON HOST"
		whereCol = ui.th.Olive
	}

	return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Local Port
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
					return ui.label(gtx, unit.Sp(13), font.Bold, theme.Mono, ui.th.Ink, localText)
				}),
				// Arrow & Counters
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.outArrow(gtx, peer)
				}),
				layout.Rigid(hgap(unit.Dp(14))),
				// Destination Endpoint
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(220))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(220))
					return ui.label(gtx, unit.Sp(13), font.Bold, theme.Mono, ui.th.Ink, peer.Remote)
				}),
				layout.Rigid(hgap(unit.Dp(8))),
				// Host indicator
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return plate(gtx, ui.th.Plate, whereCol, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(10), font.Bold, theme.Sans, whereCol, whereText)
						})
					})
				}),
				layout.Rigid(hgap(unit.Dp(12))),
				// State
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ui.th.Mute, peer.State)
				}),
			)
		})
	})
}

func (ui *UI) outArrow(gtx layout.Context, peer ports.Peer) layout.Dimensions {
	w := gtx.Dp(unit.Dp(160))
	sent := "unknown"
	if peer.BytesKnown {
		sent = formatOctets(peer.BytesSent)
	}
	back := ""
	if peer.BytesKnown && peer.BytesReceived > 0 {
		back = " / " + formatOctets(peer.BytesReceived)
	}

	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Signal, "▲ "+sent+back)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			arrowH := gtx.Dp(unit.Dp(10))
			ui.arrowShape(gtx, w, arrowH, ui.th.Signal)
			return layout.Dimensions{Size: image.Pt(w, arrowH)}
		}),
	)
}

func (ui *UI) arrowShape(gtx layout.Context, w, h int, c color.NRGBA) {
	mid := float32(h) / 2
	shaft := float32(w - 18)
	fill(gtx.Ops, image.Rect(0, int(mid)-1, int(shaft), int(mid)+2), c)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(shaft-1, mid-6))
	p.LineTo(f32.Pt(float32(w-2), mid))
	p.LineTo(f32.Pt(shaft-1, mid+6))
	p.Close()
	paint.FillShape(gtx.Ops, c, clip.Outline{Path: p.End()}.Op())
}

// sourceLiveView renders the real-time live packet and traffic stream for the selected source hub.
func (ui *UI) sourceLiveView(gtx layout.Context, h ports.Hub, view app.View) layout.Dimensions {
	if ui.topoBackBtn.Clicked(gtx) {
		ui.topoSelectedHubKey = ""
		return layout.Dimensions{}
	}
	if ui.livePauseBtn.Clicked(gtx) {
		ui.livePaused = !ui.livePaused
	}
	if ui.liveClearBtn.Clicked(gtx) {
		ui.livePackets = nil
	}

	ui.updateLiveStream(h, view.Sockets)

	pauseText := "PAUSE STREAM"
	if ui.livePaused {
		pauseText = "RESUME STREAM"
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Navigation & Control Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.button(gtx, &ui.topoBackBtn, "< BACK TO TOPOLOGY", false)
				}),
				layout.Rigid(hgap(unit.Dp(16))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					badgeColor := ui.th.Olive
					badgeText := "● LIVE REALTIME MONITOR"
					if ui.livePaused {
						badgeColor = ui.th.Stamp
						badgeText = "PAUSED (STREAM FROZEN)"
					}
					return ui.label(gtx, unit.Sp(13), font.Bold, theme.Sans, badgeColor, badgeText)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.button(gtx, &ui.livePauseBtn, pauseText, ui.livePaused)
							}),
							layout.Rigid(hgap(unit.Dp(8))),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.button(gtx, &ui.liveClearBtn, "CLEAR LOG", false)
							}),
						)
					})
				}),
			)
		}),
		layout.Rigid(gap(unit.Dp(10))),

		// Source Info Card & Metrics
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(18), font.Bold, theme.Sans, ui.th.Ink, h.Title)
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(13), font.Normal, theme.Mono, ui.th.Mute, h.Detail)
								}),
							)
						}),
						layout.Rigid(gap(unit.Dp(10))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return ui.statCard(gtx, "TX RATE (SENT)", formatOctets(uint64(ui.liveTxRate))+"/s", ui.th.Signal)
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return ui.statCard(gtx, "RX RATE (RECEIVED)", formatOctets(uint64(ui.liveRxRate))+"/s", ui.th.Olive)
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return ui.statCard(gtx, "TOTAL DATA", "▲ "+formatOctets(h.Sent)+"  ▼ "+formatOctets(h.Recv), ui.th.Ink)
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return ui.statCard(gtx, "ACTIVE DESTINATIONS", strconv.Itoa(len(h.Peers))+" peers", ui.th.Stamp)
								}),
							)
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(12))),

		// Live Packet Stream Table Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(90))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "TIME")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "DIR")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(240))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(240))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "DESTINATION (PEER)")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(95))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(95))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "PAYLOAD")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "PKTS")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "RTT")
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "STATE / EVENT")
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(4))),

		// Live Packet Stream Rows
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(ui.livePackets) == 0 {
				return ui.notice(gtx, "Monitoring socket traffic in real time. Packets and byte movements will appear here live as they occur...")
			}
			return ui.vlist(gtx, &ui.liveList, len(ui.livePackets), func(gtx layout.Context, i int) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.packetRow(gtx, ui.livePackets[i])
				})
			})
		}),
	)
}

func (ui *UI) statCard(gtx layout.Context, label, val string, col color.NRGBA) layout.Dimensions {
	return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(10), font.Bold, theme.Sans, ui.th.Mute, label)
				}),
				layout.Rigid(gap(unit.Dp(2))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(14), font.Bold, theme.Mono, col, val)
				}),
			)
		})
	})
}

func (ui *UI) packetRow(gtx layout.Context, p LivePacketEvent) layout.Dimensions {
	dirColor := ui.th.Signal
	dirText := "TX ▲"
	if p.Direction == "RX" {
		dirColor = ui.th.Olive
		dirText = "RX ▼"
	} else if p.Direction == "INIT" || p.Direction == "ACTIVE" {
		dirColor = ui.th.Olive
		dirText = "ACTIVE ●"
	} else if p.Direction == "STATE" {
		dirColor = ui.th.Stamp
		dirText = "STATE ●"
	}

	sizeText := "-"
	if p.Bytes > 0 {
		if p.Direction == "INIT" || p.Direction == "ACTIVE" {
			sizeText = formatOctets(p.Bytes)
		} else {
			sizeText = "+" + formatOctets(p.Bytes)
		}
	}

	pktText := "-"
	if p.Packets > 0 {
		pktText = strconv.Itoa(int(p.Packets)) + " pkt"
	}

	rttText := formatRTT(p.RTT)

	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Timestamp
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(90))
					return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, p.Timestamp.Format("15:04:05.000"))
				}),
				// Direction
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
					return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, dirColor, dirText)
				}),
				// Remote Destination
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(240))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(240))
					destStr := p.Remote
					if p.Local != "" && !strings.Contains(destStr, p.Local) {
						destStr += " (" + p.Local + ")"
					}
					return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, destStr)
				}),
				// Payload / Size
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(95))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(95))
					return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, dirColor, sizeText)
				}),
				// Packets
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
					return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, pktText)
				}),
				// RTT
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
					return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, rttText)
				}),
				// State / Summary
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					text := p.State
					if p.Summary != "" {
						text += "  (" + p.Summary + ")"
					}
					return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Stamp, text)
				}),
			)
		})
	})
}

var dumpSockets = netlink.DumpSockets

func (ui *UI) updateLiveStream(h ports.Hub, fallbackSocks []ports.Socket) {
	now := time.Now()
	if !ui.liveLastSample.IsZero() && now.Sub(ui.liveLastSample) < 180*time.Millisecond {
		return
	}
	dt := now.Sub(ui.liveLastSample).Seconds()
	if ui.liveLastSample.IsZero() {
		dt = 1.0
	}
	ui.liveLastSample = now

	// 1. Collect target ports and inodes for this hub
	targetPorts := make(map[uint16]bool)
	targetInodes := make(map[uint64]bool)
	for _, s := range h.Sockets {
		if s.LocalPort > 0 {
			targetPorts[s.LocalPort] = true
		}
		if s.Inode > 0 {
			targetInodes[s.Inode] = true
		}
	}
	for _, p := range h.Peers {
		if p.LocalPort > 0 {
			targetPorts[p.LocalPort] = true
		}
	}
	for _, s := range fallbackSocks {
		if (h.PID > 0 && s.PID == h.PID) ||
			(h.Process != "" && s.Process == h.Process) ||
			ports.HubKey(s) == h.Key {
			if s.LocalPort > 0 {
				targetPorts[s.LocalPort] = true
			}
			if s.Inode > 0 {
				targetInodes[s.Inode] = true
			}
		}
	}

	// 2. Identify all matched sockets belonging to this hub
	var matched []ports.Socket
	for _, s := range fallbackSocks {
		if !ports.HasPeer(s) {
			continue
		}
		if (h.PID > 0 && s.PID == h.PID) ||
			(h.Process != "" && s.Process == h.Process) ||
			targetPorts[s.LocalPort] ||
			(s.Inode > 0 && targetInodes[s.Inode]) ||
			ports.HubKey(s) == h.Key {
			matched = append(matched, s)
		}
	}

	// 3. Fallback to h.Peers if fallbackSocks had 0 matches
	if len(matched) == 0 && len(h.Peers) > 0 {
		for _, p := range h.Peers {
			matched = append(matched, ports.Socket{
				Protocol:      h.Protocol,
				LocalPort:     p.LocalPort,
				State:         p.State,
				BytesSent:     p.BytesSent,
				BytesReceived: p.BytesReceived,
				BytesKnown:    p.BytesKnown,
				SegsOut:       p.SegsOut,
				SegsIn:        p.SegsIn,
				RTTMicrosec:   p.RTTMicrosec,
			})
		}
	}

	// 4. Query live netlink sockets directly for high-resolution real-time stats
	if netlinkSocks, err := dumpSockets(); err == nil {
		liveMap := make(map[string]ports.Socket)
		for _, ns := range netlinkSocks {
			k := fmt.Sprintf("%d->%d", ns.LocalPort, ns.RemotePort)
			liveMap[k] = ns
		}
		for i := range matched {
			k := fmt.Sprintf("%d->%d", matched[i].LocalPort, matched[i].RemotePort)
			if live, ok := liveMap[k]; ok {
				if live.BytesSent > matched[i].BytesSent {
					matched[i].BytesSent = live.BytesSent
				}
				if live.BytesReceived > matched[i].BytesReceived {
					matched[i].BytesReceived = live.BytesReceived
				}
				if live.SegsOut > matched[i].SegsOut {
					matched[i].SegsOut = live.SegsOut
				}
				if live.SegsIn > matched[i].SegsIn {
					matched[i].SegsIn = live.SegsIn
				}
				if live.RTTMicrosec > 0 {
					matched[i].RTTMicrosec = live.RTTMicrosec
				}
				if live.State != "" {
					matched[i].State = live.State
				}
			}
		}
	}

	if ui.livePrevStats == nil {
		ui.livePrevStats = make(map[string]socketStats)
	}

	// 5. First time initialization: populate baseline entries immediately!
	if len(ui.livePrevStats) == 0 {
		for _, s := range matched {
			remoteStr := s.RemoteIP.String() + ":" + strconv.Itoa(int(s.RemotePort))
			if s.RemotePort == 0 {
				for _, p := range h.Peers {
					if p.LocalPort == s.LocalPort {
						remoteStr = p.Remote
						break
					}
				}
			}
			sockKey := strconv.Itoa(int(s.LocalPort)) + " -> " + remoteStr
			ui.livePrevStats[sockKey] = socketStats{
				BytesSent: s.BytesSent,
				BytesRecv: s.BytesReceived,
				SegsOut:   s.SegsOut,
				SegsIn:    s.SegsIn,
				State:     s.State,
				SeenAt:    now,
			}
			payload := s.BytesSent
			if payload == 0 {
				payload = s.BytesReceived
			}
			ui.addLivePacket(LivePacketEvent{
				Timestamp: now,
				Direction: "ACTIVE",
				Proto:     s.Protocol,
				Local:     ":" + strconv.Itoa(int(s.LocalPort)),
				Remote:    remoteStr,
				Bytes:     payload,
				Packets:   s.SegsOut + s.SegsIn,
				TotalSent: s.BytesSent,
				TotalRecv: s.BytesReceived,
				RTT:       s.RTTMicrosec,
				State:     s.State,
				Summary:   fmt.Sprintf("Active connection on :%d", s.LocalPort),
			})
		}
		return
	}

	if ui.livePaused {
		return
	}

	var totalDeltaTx, totalDeltaRx uint64

	for _, s := range matched {
		remoteStr := s.RemoteIP.String() + ":" + strconv.Itoa(int(s.RemotePort))
		if s.RemotePort == 0 {
			for _, p := range h.Peers {
				if p.LocalPort == s.LocalPort {
					remoteStr = p.Remote
					break
				}
			}
		}
		sockKey := strconv.Itoa(int(s.LocalPort)) + " -> " + remoteStr
		localAddr := ":" + strconv.Itoa(int(s.LocalPort))

		prev, exists := ui.livePrevStats[sockKey]
		if !exists {
			ui.addLivePacket(LivePacketEvent{
				Timestamp: now,
				Direction: "ACTIVE",
				Proto:     s.Protocol,
				Local:     localAddr,
				Remote:    remoteStr,
				TotalSent: s.BytesSent,
				TotalRecv: s.BytesReceived,
				RTT:       s.RTTMicrosec,
				State:     s.State,
				Summary:   "New connection established",
			})
			ui.livePrevStats[sockKey] = socketStats{
				BytesSent: s.BytesSent,
				BytesRecv: s.BytesReceived,
				SegsOut:   s.SegsOut,
				SegsIn:    s.SegsIn,
				State:     s.State,
				SeenAt:    now,
			}
			continue
		}

		// Outbound data
		if s.BytesSent > prev.BytesSent {
			deltaBytes := s.BytesSent - prev.BytesSent
			deltaPkts := uint32(1)
			if s.SegsOut > prev.SegsOut {
				deltaPkts = s.SegsOut - prev.SegsOut
			}
			totalDeltaTx += deltaBytes
			ui.addLivePacket(LivePacketEvent{
				Timestamp: now,
				Direction: "TX",
				Proto:     s.Protocol,
				Local:     localAddr,
				Remote:    remoteStr,
				Bytes:     deltaBytes,
				Packets:   deltaPkts,
				TotalSent: s.BytesSent,
				TotalRecv: s.BytesReceived,
				RTT:       s.RTTMicrosec,
				State:     s.State,
				Summary:   "Outbound payload",
			})
		}

		// Inbound data
		if s.BytesReceived > prev.BytesRecv {
			deltaBytes := s.BytesReceived - prev.BytesRecv
			deltaPkts := uint32(1)
			if s.SegsIn > prev.SegsIn {
				deltaPkts = s.SegsIn - prev.SegsIn
			}
			totalDeltaRx += deltaBytes
			ui.addLivePacket(LivePacketEvent{
				Timestamp: now,
				Direction: "RX",
				Proto:     s.Protocol,
				Local:     localAddr,
				Remote:    remoteStr,
				Bytes:     deltaBytes,
				Packets:   deltaPkts,
				TotalSent: s.BytesSent,
				TotalRecv: s.BytesReceived,
				RTT:       s.RTTMicrosec,
				State:     s.State,
				Summary:   "Inbound payload",
			})
		}

		// State changes
		if s.State != prev.State {
			ui.addLivePacket(LivePacketEvent{
				Timestamp: now,
				Direction: "STATE",
				Proto:     s.Protocol,
				Local:     localAddr,
				Remote:    remoteStr,
				State:     s.State,
				Summary:   fmt.Sprintf("State: %s -> %s", prev.State, s.State),
			})
		}

		ui.livePrevStats[sockKey] = socketStats{
			BytesSent: s.BytesSent,
			BytesRecv: s.BytesReceived,
			SegsOut:   s.SegsOut,
			SegsIn:    s.SegsIn,
			State:     s.State,
			SeenAt:    now,
		}
	}

	if dt > 0 {
		ui.liveTxRate = float64(totalDeltaTx) / dt
		ui.liveRxRate = float64(totalDeltaRx) / dt
	}
}

func (ui *UI) addLivePacket(ev LivePacketEvent) {
	ui.livePackets = append([]LivePacketEvent{ev}, ui.livePackets...)
	if len(ui.livePackets) > 300 {
		ui.livePackets = ui.livePackets[:300]
	}
}

func formatRTT(us uint32) string {
	if us == 0 {
		return "-"
	}
	if us < 1000 {
		return strconv.Itoa(int(us)) + " µs"
	}
	ms := float64(us) / 1000.0
	return strconv.FormatFloat(ms, 'f', 1, 64) + " ms"
}

func formatOctets(n uint64) string {
	if n < 1024 {
		return strconv.FormatUint(n, 10) + " B"
	}
	units := []string{"KB", "MB", "GB", "TB"}
	v := float64(n)
	for i, unitName := range units {
		v /= 1024
		if v < 1024 || i == len(units)-1 {
			return strconv.FormatFloat(v, 'f', 1, 64) + " " + unitName
		}
	}
	return strconv.FormatUint(n, 10) + " B"
}

type geoFlowItem struct {
	socket   ports.Socket
	destLoc  geoip.Location
	remoteIP string
	portName string
}

func wellKnownPort(proto string, port uint16) string {
	switch port {
	case 80:
		return "HTTP"
	case 443:
		return "HTTPS"
	case 53:
		return "DNS"
	case 22:
		return "SSH"
	case 123:
		return "NTP"
	case 8080:
		return "HTTP-ALT"
	case 8443:
		return "HTTPS-ALT"
	case 3306:
		return "MYSQL"
	case 5432:
		return "POSTGRES"
	case 6379:
		return "REDIS"
	case 27017:
		return "MONGODB"
	case 51820:
		return "WIREGUARD"
	default:
		return strconv.Itoa(int(port))
	}
}

func (ui *UI) worldGeolocationView(gtx layout.Context, view app.View) layout.Dimensions {
	var flows []geoFlowItem
	countryCounts := make(map[string]int)
	countryNames := make(map[string]string)
	countryFlags := make(map[string]string)
	var totalSent, totalRecv uint64
	localIP := "127.0.0.1"

	for _, ifc := range view.Interfaces {
		if ifc.Up && ifc.Name != "lo" && len(ifc.Addresses) > 0 {
			localIP = ifc.Addresses[0].String()
			break
		}
	}

	for _, s := range view.Sockets {
		if !ports.HasPeer(s) {
			continue
		}
		loc := geoip.Lookup(s.RemoteIP)
		destIP := s.RemoteIP.String()
		flows = append(flows, geoFlowItem{
			socket:   s,
			destLoc:  loc,
			remoteIP: destIP,
			portName: wellKnownPort(s.Protocol, s.RemotePort),
		})

		countryCounts[loc.CountryCode]++
		countryNames[loc.CountryCode] = loc.CountryName
		countryFlags[loc.CountryCode] = loc.Flag
		if s.BytesKnown {
			totalSent += s.BytesSent
			totalRecv += s.BytesReceived
		}
	}

	countryCodes := make([]string, 0, len(countryCounts))
	for code := range countryCounts {
		countryCodes = append(countryCodes, code)
	}
	sort.SliceStable(countryCodes, func(i, j int) bool {
		if countryCounts[countryCodes[i]] != countryCounts[countryCodes[j]] {
			return countryCounts[countryCodes[i]] > countryCounts[countryCodes[j]]
		}
		return countryCodes[i] < countryCodes[j]
	})

	var foreignCount, localCount int
	for _, f := range flows {
		if f.destLoc.IsLocal {
			localCount++
		} else {
			foreignCount++
		}
	}

	if len(ui.geoCountryBtns) < len(countryCodes)+1 {
		ui.geoCountryBtns = make([]widget.Clickable, len(countryCodes)+1)
	}
	if ui.geoScopeBtns[0].Clicked(gtx) {
		ui.geoScopeFilter = ""
	}
	if ui.geoScopeBtns[1].Clicked(gtx) {
		ui.geoScopeFilter = "FOREIGN"
	}
	if ui.geoScopeBtns[2].Clicked(gtx) {
		ui.geoScopeFilter = "INTERNAL"
	}

	if ui.geoCountryBtns[0].Clicked(gtx) {
		ui.geoSelectedCountry = ""
		ui.geoZoom = 1.0
		ui.geoPanX = 0
		ui.geoPanY = 0
	}
	for i, code := range countryCodes {
		if ui.geoCountryBtns[i+1].Clicked(gtx) {
			if ui.geoSelectedCountry == code {
				ui.geoSelectedCountry = ""
				ui.geoZoom = 1.0
				ui.geoPanX = 0
				ui.geoPanY = 0
			} else {
				ui.geoSelectedCountry = code
				c := GetMapCountry(code)
				if c != nil {
					ui.geoZoom = 2.4
					ui.geoPanX = -float64(c.Center[0]-0.5) * 600.0 * ui.geoZoom
					ui.geoPanY = -float64(c.Center[1]-0.5) * 350.0 * ui.geoZoom
				}
			}
		}
	}

	search := strings.ToLower(strings.TrimSpace(ui.geoSearch.Text()))
	var filtered []geoFlowItem
	for _, f := range flows {
		if ui.geoScopeFilter == "FOREIGN" && f.destLoc.IsLocal {
			continue
		}
		if ui.geoScopeFilter == "INTERNAL" && !f.destLoc.IsLocal {
			continue
		}
		if ui.geoSelectedCountry != "" && f.destLoc.CountryCode != ui.geoSelectedCountry {
			continue
		}
		if search != "" {
			match := strings.Contains(strings.ToLower(f.remoteIP), search) ||
				strings.Contains(strconv.Itoa(int(f.socket.LocalPort)), search) ||
				strings.Contains(strconv.Itoa(int(f.socket.RemotePort)), search) ||
				strings.Contains(strings.ToLower(f.socket.Process), search) ||
				strings.Contains(strings.ToLower(f.destLoc.CountryName), search) ||
				strings.Contains(strings.ToLower(f.destLoc.CountryCode), search) ||
				strings.Contains(strings.ToLower(f.portName), search)
			if !match {
				continue
			}
		}
		filtered = append(filtered, f)
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// 1. Globe Dashboard Banner Card
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return drawIconMap(gtx, unit.Dp(36), ui.th.Stamp)
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(16), font.Bold, theme.Sans, ui.th.Ink, "GLOBAL OUTBOUND TOPOLOGY")
								}),
								layout.Rigid(gap(unit.Dp(3))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(11.5), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("SOURCE: LOCAL HOST (%s) ·", localIP))
										}),
										layout.Rigid(hgap(unit.Dp(6))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.flagBadge(gtx, "IR", unit.Dp(16), unit.Dp(11))
										}),
										layout.Rigid(hgap(unit.Dp(4))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(11.5), font.Bold, theme.Mono, ui.th.Olive, "IRAN")
										}),
										layout.Rigid(hgap(unit.Dp(6))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return drawIconArrowRight(gtx, unit.Dp(12), ui.th.Mute)
										}),
										layout.Rigid(hgap(unit.Dp(6))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(11.5), font.Bold, theme.Mono, ui.th.Olive, "FOREIGN DESTINATIONS")
										}),
									)
								}),
							)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
						// Stats: Active Flows
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "ACTIVE FLOWS")
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(18), font.Bold, theme.Sans, ui.th.Ink, strconv.Itoa(len(flows)))
								}),
							)
						}),
						layout.Rigid(hgap(unit.Dp(20))),
						// Stats: Countries
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "COUNTRIES")
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(18), font.Bold, theme.Sans, ui.th.Olive, strconv.Itoa(len(countryCounts)))
								}),
							)
						}),
						layout.Rigid(hgap(unit.Dp(20))),
						// Stats: Traffic
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "TOTAL TRAFFIC")
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Stamp, fmt.Sprintf("▲ %s  ▼ %s", formatOctets(totalSent), formatOctets(totalRecv)))
								}),
							)
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 2. Earth Globe & World Geolocation Map Canvas
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.globeView(gtx, flows, localIP)
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 3. Traffic Scope Categorization (WAN vs LAN/Loopback)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					allLabel := fmt.Sprintf("ALL DESTINATIONS (%d)", len(flows))
					return ui.iconButton(gtx, &ui.geoScopeBtns[0], drawIconNetwork, allLabel, ui.geoScopeFilter == "")
				}),
				layout.Rigid(hgap(unit.Dp(6))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					wanLabel := fmt.Sprintf("FOREIGN WAN (%d)", foreignCount)
					return ui.iconButton(gtx, &ui.geoScopeBtns[1], drawIconMap, wanLabel, ui.geoScopeFilter == "FOREIGN")
				}),
				layout.Rigid(hgap(unit.Dp(6))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lanLabel := fmt.Sprintf("LOCAL / LAN (%d)", localCount)
					return ui.iconButton(gtx, &ui.geoScopeBtns[2], drawIconTarget, lanLabel, ui.geoScopeFilter == "INTERNAL")
				}),
			)
		}),
		layout.Rigid(gap(unit.Dp(6))),

		// 4. Scrollable Country Filter Pills Bar (Clean Horizontal Scroll)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			totalPills := len(countryCodes) + 1
			ui.geoCountryList.Axis = layout.Horizontal
			return material.List(ui.th.Material, &ui.geoCountryList).Layout(gtx, totalPills, func(gtx layout.Context, i int) layout.Dimensions {
				return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if i == 0 {
						allLabel := fmt.Sprintf("ALL COUNTRIES (%d)", len(flows))
						return ui.countryPillButton(gtx, &ui.geoCountryBtns[0], "", allLabel, ui.geoSelectedCountry == "")
					}
					code := countryCodes[i-1]
					selected := ui.geoSelectedCountry == code
					name := countryNames[code]
					if name == "" {
						name = code
					}
					lbl := fmt.Sprintf("%s (%d)", name, countryCounts[code])
					return ui.countryPillButton(gtx, &ui.geoCountryBtns[i], code, lbl, selected)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 3. Search Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ed := material.Editor(ui.th.Material, &ui.geoSearch, "Search destination IP, port, process, or country...")
			ed.TextSize = unit.Sp(13)
			ed.Color = ui.th.Ink
			ed.HintColor = ui.th.Mute
			ed.Font.Typeface = theme.Mono
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, ed.Layout)
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 4. Table Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(230))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(230))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "LOCAL SOURCE (PROCESS · PORT)")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(180))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(180))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "TRAFFIC & DIRECTION")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(240))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(240))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "REMOTE DESTINATION (IP : PORT)")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "TARGET COUNTRY / DESTINATION")
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(4))),

		// 5. Flows Stream List
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(filtered) == 0 {
				return ui.notice(gtx, "No outbound flows matching the current filter.")
			}
			return ui.vlist(gtx, &ui.geoList, len(filtered), func(gtx layout.Context, i int) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.geoFlowRow(gtx, filtered[i])
				})
			})
		}),
	)
}

func (ui *UI) geoFlowRow(gtx layout.Context, item geoFlowItem) layout.Dimensions {
	s := item.socket
	procName := s.Process
	if procName == "" {
		if s.PID > 0 {
			procName = fmt.Sprintf("PID %d", s.PID)
		} else if s.User != "" {
			procName = fmt.Sprintf("system (%s)", s.User)
		} else {
			procName = "outbound flow"
		}
	}

	trafficTxt := "byte count unavailable"
	if s.BytesKnown {
		trafficTxt = fmt.Sprintf("▲ %s  ▼ %s", formatOctets(s.BytesSent), formatOctets(s.BytesReceived))
	}

	stateTxt := s.State
	if stateTxt == "" {
		stateTxt = "ESTABLISHED"
	}

	countryCol := ui.th.Ink
	countryBorder := ui.th.Rule
	if !item.destLoc.IsLocal {
		countryBorder = ui.th.Olive
	}

	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// 1. Local Source (Process & Local Port)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(230))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(230))
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(13), font.Bold, theme.Mono, ui.th.Ink, procName)
						}),
						layout.Rigid(gap(unit.Dp(2))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11.5), font.Normal, theme.Mono, ui.th.Mute, fmt.Sprintf("%s:%d", s.LocalIP.String(), s.LocalPort))
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(12))),

				// 2. Traffic Flow & Direction
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(180))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(180))
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Stamp, trafficTxt)
						}),
						layout.Rigid(gap(unit.Dp(2))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Signal, "────────►")
								}),
								layout.Rigid(hgap(unit.Dp(4))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Olive, stateTxt)
								}),
							)
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(12))),

				// 3. Remote Destination (IP : Port)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(240))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(240))
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(13), font.Bold, theme.Mono, ui.th.Ink, fmt.Sprintf("%s:%d", item.remoteIP, s.RemotePort))
						}),
						layout.Rigid(gap(unit.Dp(2))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(5), Right: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(9.5), font.Bold, theme.Mono, ui.th.Mute, item.portName)
										})
									})
								}),
							)
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(12))),

				// 4. Target Country Badge
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return plate(gtx, ui.th.Raised, countryBorder, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.flagBadge(gtx, item.destLoc.CountryCode, unit.Dp(16), unit.Dp(11))
								}),
								layout.Rigid(hgap(unit.Dp(8))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									cTxt := fmt.Sprintf("%s [%s]", item.destLoc.CountryName, item.destLoc.CountryCode)
									return ui.label(gtx, unit.Sp(11.5), font.Bold, theme.Mono, countryCol, cTxt)
								}),
							)
						})
					})
				}),
			)
		})
	})
}
