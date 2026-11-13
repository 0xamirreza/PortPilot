package gui

import (
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/config"
	"github.com/0xamirreza/portpilot/internal/core/firewall"
	"github.com/0xamirreza/portpilot/internal/core/network"
	"github.com/0xamirreza/portpilot/internal/core/ports"
	"github.com/0xamirreza/portpilot/internal/core/processes"
	"github.com/0xamirreza/portpilot/internal/discovery/services"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
	"github.com/0xamirreza/portpilot/internal/system/privilege"
	"github.com/0xamirreza/portpilot/internal/version"
)

func (ui *UI) visibleSockets(view app.View) []ports.Socket {
	ui.query.Text = ui.search.Text()
	return ports.Apply(view.Sockets, ui.query)
}

func (ui *UI) overview(gtx layout.Context, view app.View) layout.Dimensions {
	sum := ports.Summarize(view.Sockets, view.Interfaces)
	items := []struct {
		value, label, kind string
	}{
		{strconv.Itoa(sum.TCPListen), "TCP LISTEN", "ink"},
		{strconv.Itoa(sum.UDPBound), "UDP BOUND", "ink"},
		{strconv.Itoa(sum.Processes), "PROCESSES", "ink"},
		{strconv.Itoa(sum.OwnerUnknown), "OWNER UNKNOWN", "stamp"},
		{strconv.Itoa(sum.Wildcard4), "WILDCARD V4", "signal"},
		{strconv.Itoa(sum.Wildcard6), "WILDCARD V6", "signal"},
		{strconv.Itoa(sum.Localhost), "LOCALHOST", "olive"},
		{strconv.Itoa(sum.InterfacesUp), "LINKS UP", "ink"},
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.stampGrid(gtx, items) }),
		layout.Rigid(gap(unit.Dp(10))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.firewallOverviewCard(gtx, view.Firewall)
		}),
		layout.Rigid(gap(unit.Dp(12))),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.interfaceList(gtx, view.Interfaces)
		}),
	)
}

func (ui *UI) firewallOverviewCard(gtx layout.Context, fw firewall.Report) layout.Dimensions {
	mode := strings.ToUpper(fw.Mode)
	if mode == "" {
		mode = "UNAVAILABLE"
	}
	statusText := "INACTIVE"
	statusCol := ui.th.Mute
	if fw.UFWActive || fw.Mode == firewall.ModeUFW || fw.Mode == firewall.ModeNFTables {
		statusText = "ACTIVE"
		statusCol = ui.th.Olive
	}

	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// 1. Firewall Mode badge
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "FIREWALL BACKEND")
						}),
						layout.Rigid(gap(unit.Dp(2))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(15), font.Bold, theme.Sans, ui.th.Ink, mode)
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(24))),
				// 2. Status badge
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "STATUS")
						}),
						layout.Rigid(gap(unit.Dp(3))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return plate(gtx, ui.th.Raised, statusCol, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, statusCol, statusText)
								})
							})
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(24))),
				// 3. Active Rules
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "ACTIVE RULES")
						}),
						layout.Rigid(gap(unit.Dp(2))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(15), font.Bold, theme.Mono, ui.th.Ink, strconv.Itoa(len(fw.Rules)))
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(24))),
				// 4. Temporary Schedules
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "TEMPORARY RULES")
						}),
						layout.Rigid(gap(unit.Dp(2))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							col := ui.th.Mute
							if len(fw.Schedules) > 0 {
								col = ui.th.Stamp
							}
							return ui.label(gtx, unit.Sp(15), font.Bold, theme.Mono, col, strconv.Itoa(len(fw.Schedules)))
						}),
					)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{}
				}),
				// 5. On Boot indicator
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					bootTxt := "BOOT ENABLED"
					bootCol := ui.th.Olive
					if !fw.UFWEnabledOnBoot {
						bootTxt = "BOOT DISABLED"
						bootCol = ui.th.Mute
					}
					return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, bootCol, bootTxt)
						})
					})
				}),
			)
		})
	})
}

func (ui *UI) stampGrid(gtx layout.Context, items []struct{ value, label, kind string }) layout.Dimensions {
	cols := 4
	if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(760)) {
		cols = 2
	}
	var rows []layout.FlexChild
	for start := 0; start < len(items); start += cols {
		start, end := start, start+cols
		if end > len(items) {
			end = len(items)
		}
		chunk := append([]struct{ value, label, kind string }{}, items[start:end]...)
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			kids := make([]layout.FlexChild, 0, len(chunk)*2)
			for i, item := range chunk {
				item := item
				if i > 0 {
					kids = append(kids, layout.Rigid(hgap(unit.Dp(8))))
				}
				kids = append(kids, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					col := ui.th.Ink
					switch item.kind {
					case "signal":
						col = ui.th.Signal
					case "olive":
						col = ui.th.Olive
					case "stamp":
						col = ui.th.Stamp
					}
					return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(28), font.Bold, theme.Sans, col, item.value)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, item.label)
								}),
							)
						})
					})
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
		}))
		rows = append(rows, layout.Rigid(gap(unit.Dp(8))))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (ui *UI) interfaceList(gtx layout.Context, ifaces []network.Interface) layout.Dimensions {
	upCount := 0
	for _, ifc := range ifaces {
		if ifc.Up {
			upCount++
		}
	}
	downCount := len(ifaces) - upCount

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Header Stats Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, "NETWORK INTERFACES")
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("TOTAL: %d", len(ifaces)))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, fmt.Sprintf("UP: %d", upCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("DOWN: %d", downCount))
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(6))),
		// Table Columns Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(65))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(65))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "STATUS")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(140))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(140))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "INTERFACE")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(130))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(130))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "TYPE")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "MTU")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "ASSIGNED IP ADDRESSES (IPv4 / IPv6)")
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(6))),
		// Scrollable List of Interfaces
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(ifaces) == 0 {
				return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(24), Bottom: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(13), font.Normal, theme.Mono, ui.th.Mute, "No network interfaces were returned.")
						})
					})
				})
			}
			return ui.vlist(gtx, &ui.ifaceList, len(ifaces), func(gtx layout.Context, i int) layout.Dimensions {
				return ui.interfaceRow(gtx, ifaces[i], i)
			})
		}),
	)
}

func (ui *UI) ifaceType(name string) (string, color.NRGBA) {
	lower := strings.ToLower(name)
	switch {
	case lower == "lo":
		return "LOOPBACK", ui.th.Mute
	case strings.HasPrefix(lower, "wl") || strings.HasPrefix(lower, "wifi"):
		return "WI-FI", ui.th.Stamp
	case strings.HasPrefix(lower, "en") || strings.HasPrefix(lower, "eth"):
		return "ETHERNET", ui.th.Olive
	case strings.HasPrefix(lower, "docker") || strings.HasPrefix(lower, "br-") || strings.HasPrefix(lower, "veth"):
		return "DOCKER / VETH", ui.th.Ink
	case strings.HasPrefix(lower, "virbr") || strings.HasPrefix(lower, "vnet"):
		return "VM BRIDGE", ui.th.Ink
	case strings.HasPrefix(lower, "lxc"):
		return "LXC BRIDGE", ui.th.Ink
	case strings.HasPrefix(lower, "tun") || strings.HasPrefix(lower, "tap") || strings.HasPrefix(lower, "wg") || strings.Contains(lower, "tun") || strings.Contains(lower, "tailscale"):
		return "VPN / TUNNEL", ui.th.Signal
	default:
		return "INTERFACE", ui.th.Mute
	}
}

func (ui *UI) interfaceRow(gtx layout.Context, ifc network.Interface, i int) layout.Dimensions {
	gtx.Constraints.Min.Y = 0

	statusText := "DOWN"
	statusCol := ui.th.Mute
	statusBg := ui.th.Plate
	if ifc.Up {
		statusText = "UP"
		statusCol = ui.th.Olive
		statusBg = ui.th.Raised
	}

	typeName, typeCol := ui.ifaceType(ifc.Name)

	var ipv4s, ipv6s []string
	for _, addr := range ifc.Addresses {
		if addr.Is4() {
			ipv4s = append(ipv4s, addr.String())
		} else if addr.Is6() {
			ipv6s = append(ipv6s, addr.String())
		}
	}

	click := ui.rowClick(&ui.ifaceRow, i)
	bg := ui.th.Plate
	border := ui.th.Rule
	if click.Hovered() {
		bg = ui.th.Raised
	}

	return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, bg, border, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						// Status Badge (UP/DOWN)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(65))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(65))
							return plate(gtx, statusBg, statusCol, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, statusCol, statusText)
									})
								})
							})
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						// Interface Name
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(140))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(140))
							return ui.label(gtx, unit.Sp(12.5), font.Bold, theme.Mono, ui.th.Ink, ifc.Name)
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						// Type Badge
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(130))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(130))
							return plate(gtx, ui.th.Raised, typeCol, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, typeCol, typeName)
									})
								})
							})
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						// MTU
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
							mtuStr := fmt.Sprintf("MTU %d", ifc.MTU)
							if ifc.MTU <= 0 {
								mtuStr = "—"
							}
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, mtuStr)
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						// Assigned IP Addresses (IPv4 and IPv6)
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if len(ipv4s) == 0 && len(ipv6s) == 0 {
								return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, "— (no IP address assigned)")
							}
							var addrKids []layout.FlexChild
							if len(ipv4s) > 0 {
								addrKids = append(addrKids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "IPv4: ")
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, strings.Join(ipv4s, ", "))
										}),
									)
								}))
							}
							if len(ipv6s) > 0 {
								if len(ipv4s) > 0 {
									addrKids = append(addrKids, layout.Rigid(gap(unit.Dp(2))))
								}
								addrKids = append(addrKids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, "IPv6: ")
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, strings.Join(ipv6s, ", "))
										}),
									)
								}))
							}
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx, addrKids...)
						}),
					)
				})
			})
		})
	})
}

func (ui *UI) ports(gtx layout.Context, view app.View) layout.Dimensions {
	rows := ui.visibleSockets(view)
	wide := gtx.Constraints.Max.X >= gtx.Dp(unit.Dp(980))
	list := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.filterBar(gtx) }),
			layout.Rigid(gap(unit.Dp(8))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ui.socketHead(gtx) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(rows) == 0 {
					return ui.notice(gtx, "No sockets match this filter.")
				}
				return ui.vlist(gtx, &ui.sockList, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
					click := ui.rowClick(&ui.sockRow, i)
					if click.Clicked(gtx) {
						ui.selectedSock = rows[i].Key()
					}
					return ui.socketLine(gtx, rows[i], rows[i].Key() == ui.selectedSock, click)
				})
			}),
		)
	}
	if !wide && ui.selectedSock != "" {
		if ui.back.Clicked(gtx) {
			ui.selectedSock = ""
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &ui.back, "BACK TO LIST", false)
			}),
			layout.Rigid(gap(unit.Dp(8))),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return ui.socketDetail(gtx, findSocket(view.Sockets, ui.selectedSock))
			}),
		)
	}
	if !wide {
		return list(gtx)
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, list),
		layout.Rigid(hgap(unit.Dp(10))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))
			gtx.Constraints.Max.X = gtx.Dp(unit.Dp(320))
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return ui.socketDetail(gtx, findSocket(view.Sockets, ui.selectedSock))
		}),
	)
}

func (ui *UI) filterBar(gtx layout.Context) layout.Dimensions {
	names := []string{"TCP", "UDP", "IPv4", "IPv6", "LOCAL", "WILD", "HOST", "LISTEN", "ACTIVE"}
	flags := []*bool{&ui.query.TCP, &ui.query.UDP, &ui.query.IPv4, &ui.query.IPv6, &ui.query.Localhost, &ui.query.Wildcard, &ui.query.Specific, &ui.query.Listening, &ui.query.Active}
	for i := range names {
		if ui.chips[i].Clicked(gtx) {
			*flags[i] = !*flags[i]
		}
	}
	search := func(gtx layout.Context) layout.Dimensions {
		ed := material.Editor(ui.th.Material, &ui.search, "Search port, process, address")
		ed.Color = ui.th.Ink
		ed.HintColor = ui.th.Mute
		ed.Font.Typeface = theme.Mono
		ed.TextSize = unit.Sp(13)
		return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, ed.Layout)
		})
	}
	chips := func(gtx layout.Context) layout.Dimensions {
		kids := make([]layout.FlexChild, 0, len(names)*2)
		for i := range names {
			i := i
			if i > 0 {
				kids = append(kids, layout.Rigid(hgap(unit.Dp(6))))
			}
			kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.button(gtx, &ui.chips[i], names[i], *flags[i])
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(search),
		layout.Rigid(gap(unit.Dp(6))),
		layout.Rigid(chips),
	)
}

func (ui *UI) socketHead(gtx layout.Context) layout.Dimensions {
	return ui.socketCols(gtx, "PROTO", "ADDRESS", "PORT", "STATE", "PROCESS", "PID", "EXPOSURE", ui.th.Mute, false, nil)
}

func (ui *UI) socketLine(gtx layout.Context, s ports.Socket, selected bool, click *widget.Clickable) layout.Dimensions {
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return ui.socketCols(gtx, s.Protocol, s.LocalIP.String(), strconv.Itoa(int(s.LocalPort)), s.State, empty(s.Process, "—"), pidText(s), strings.ToUpper(s.Exposure), ui.th.Ink, selected, click)
	})
}

func (ui *UI) socketCols(gtx layout.Context, proto, addr, port, state, process, pid, exposure string, col interface{}, selected bool, click *widget.Clickable) layout.Dimensions {
	ink := ui.th.Ink
	bg := ui.th.Void
	if selected {
		bg = ui.th.Raised
	} else if click != nil && click.Hovered() {
		bg = ui.th.Plate
	}
	if col != nil && !selected && click == nil {
		ink = ui.th.Mute
	}
	return plate(gtx, bg, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			cell := func(weight float32, text string) layout.FlexChild {
				return layout.Flexed(weight, func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ink, text)
				})
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				cell(0.7, proto), cell(1.8, addr), cell(0.6, port), cell(1.1, state), cell(1.4, process), cell(0.6, pid), cell(1, exposure),
			)
		})
	})
}

func (ui *UI) socketDetail(gtx layout.Context, s *ports.Socket) layout.Dimensions {
	if s == nil {
		return ui.notice(gtx, "Select a socket. The list is a live read, not a saved inventory.")
	}
	rows := [][2]string{
		{"Protocol", s.Protocol + " " + s.Family},
		{"State", s.State},
		{"Exposure", s.Exposure},
		{"PID", pidText(*s)},
		{"User", empty(s.User, "unknown")},
		{"Process", empty(s.Process, "not available")},
		{"Executable", empty(s.Executable, "not available")},
		{"Service", empty(s.Service, "none detected")},
		{"Command", empty(s.Command, "not available")},
		{"Interface", empty(s.Interface, "none")},
		{"Namespace", empty(s.NetNS, "not available")},
		{"Container", empty(s.Container, "none detected")},
		{"Inode", strconv.FormatUint(s.Inode, 10)},
		{"Source", s.Source},
	}
	if !s.OwnerKnown {
		rows = append(rows, [2]string{"Ownership", "not available. An open port without a visible process is not treated as a threat."})
	}
	kids := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, unit.Sp(18), font.Bold, theme.Mono, ui.th.Ink, s.LocalIP.String()+":"+strconv.Itoa(int(s.LocalPort)))
		}),
		layout.Rigid(gap(unit.Dp(8))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.wrap(gtx, unit.Sp(13), ui.th.Ink, ports.Reachability(*s))
		}),
		layout.Rigid(gap(unit.Dp(10))),
	}
	for _, row := range rows {
		row := row
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, strings.ToUpper(row[0]))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.wrap(gtx, unit.Sp(13), ui.th.Ink, row[1])
					}),
				)
			})
		}))
	}
	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
		})
	})
}

var procFilterLabels = []string{"ALL", "USER", "ROOT", "SERVICES"}

func (ui *UI) processes(gtx layout.Context, view app.View) layout.Dimensions {
	for i := range ui.procFilterBtns {
		if ui.procFilterBtns[i].Clicked(gtx) {
			ui.procFilter = i
		}
	}

	totalCount := len(view.Processes)
	rootCount, userCount, serviceCount := 0, 0, 0
	for _, p := range view.Processes {
		if p.User == "root" || p.UID == 0 {
			rootCount++
		} else {
			userCount++
		}
		if p.Service != "" {
			serviceCount++
		}
	}

	search := strings.ToLower(strings.TrimSpace(ui.procSearch.Text()))
	var rows []processes.Process
	for _, p := range view.Processes {
		switch ui.procFilter {
		case 1: // USER
			if p.User == "root" || p.UID == 0 {
				continue
			}
		case 2: // ROOT
			if !(p.User == "root" || p.UID == 0) {
				continue
			}
		case 3: // SERVICES
			if p.Service == "" {
				continue
			}
		}

		if search != "" {
			match := strings.Contains(strconv.Itoa(p.PID), search) ||
				strings.Contains(strings.ToLower(p.Name), search) ||
				strings.Contains(strings.ToLower(p.User), search) ||
				strings.Contains(strings.ToLower(p.Service), search) ||
				strings.Contains(strings.ToLower(p.Command), search)
			if !match {
				continue
			}
		}
		rows = append(rows, p)
	}

	var chosen *processes.Process
	for i := range view.Processes {
		if view.Processes[i].PID == ui.selectedProc {
			chosen = &view.Processes[i]
			break
		}
	}
	if chosen == nil && len(rows) > 0 && ui.selectedProc != -1 {
		chosen = &rows[0]
		ui.selectedProc = rows[0].PID
	}

	if chosen != nil {
		if ui.termBtn.Clicked(gtx) {
			ui.confirm = &confirm{
				text: fmt.Sprintf("Send SIGTERM to pid %d (%s)? The helper checks the start ticks before the signal. This does not stop a service.", chosen.PID, empty(chosen.Name, "process")),
				msg:  privilege.Message{Op: "signal", PID: chosen.PID, StartTicks: chosen.StartTicks, Signal: "TERM"},
			}
		}
		if ui.killBtn.Clicked(gtx) {
			ui.confirm = &confirm{
				text: fmt.Sprintf("Send SIGKILL to pid %d (%s)? This cannot be caught by the process. The helper still checks the start ticks.", chosen.PID, empty(chosen.Name, "process")),
				msg:  privilege.Message{Op: "signal", PID: chosen.PID, StartTicks: chosen.StartTicks, Signal: "KILL", ConfirmKill: true},
			}
		}
		if chosen.Service != "" && ui.procSvc[0].Clicked(gtx) {
			ui.confirm = &confirm{
				text: "Stop service " + chosen.Service + "? This is not the same as signalling pid " + strconv.Itoa(chosen.PID) + ".",
				msg:  privilege.Message{Op: "service", Service: chosen.Service, ServiceOp: "stop"},
			}
		}
		if chosen.Service != "" && ui.procSvc[1].Clicked(gtx) {
			ui.confirm = &confirm{
				text: "Restart service " + chosen.Service + "?",
				msg:  privilege.Message{Op: "service", Service: chosen.Service, ServiceOp: "restart"},
			}
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// 1. Search Bar & Status Filter Tabs
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					ed := material.Editor(ui.th.Material, &ui.procSearch, "Search processes by name, PID, user, service, or command...")
					ed.TextSize = unit.Sp(13)
					ed.Color = ui.th.Ink
					ed.HintColor = ui.th.Mute
					ed.Font.Typeface = theme.Mono
					return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, ed.Layout)
					})
				}),
				layout.Rigid(hgap(unit.Dp(10))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					kids := make([]layout.FlexChild, 0, len(procFilterLabels)*2)
					for i, lbl := range procFilterLabels {
						i, lbl := i, lbl
						kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.procFilterBtns[i], lbl, ui.procFilter == i)
						}))
						if i < len(procFilterLabels)-1 {
							kids = append(kids, layout.Rigid(hgap(unit.Dp(6))))
						}
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
				}),
			)
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 2. Statistics Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Ink, fmt.Sprintf("TOTAL: %d", totalCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, fmt.Sprintf("USER: %d", userCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Stamp, fmt.Sprintf("ROOT: %d", rootCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("SERVICES: %d", serviceCount))
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 3. Main Split View: Left List + Right Detail Inspector
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				// Left Table
				layout.Flexed(1.2, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// Table Column Headers
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "PID")
										}),
										layout.Rigid(hgap(unit.Dp(8))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(150))
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "PROCESS")
										}),
										layout.Rigid(hgap(unit.Dp(8))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "USER")
										}),
										layout.Rigid(hgap(unit.Dp(8))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "MEMORY")
										}),
										layout.Rigid(hgap(unit.Dp(8))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(55))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(55))
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "CPU")
										}),
										layout.Rigid(hgap(unit.Dp(8))),
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "SOCKETS")
										}),
									)
								})
							})
						}),
						layout.Rigid(gap(unit.Dp(4))),
						// Scrollable rows
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if len(rows) == 0 {
								return ui.notice(gtx, "No processes match the active search or filter.")
							}
							return ui.vlist(gtx, &ui.procList, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
								click := ui.rowClick(&ui.procRow, i)
								if click.Clicked(gtx) {
									ui.selectedProc = rows[i].PID
								}
								return ui.processRow(gtx, rows[i], rows[i].PID == ui.selectedProc, click)
							})
						}),
					)
				}),
				layout.Rigid(hgap(unit.Dp(10))),
				// Right Inspector
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.processDetail(gtx, chosen)
				}),
			)
		}),
	)
}

func (ui *UI) processRow(gtx layout.Context, p processes.Process, selected bool, click *widget.Clickable) layout.Dimensions {
	gtx.Constraints.Min.Y = 0

	bg := ui.th.Plate
	border := ui.th.Rule
	if selected {
		bg = ui.th.Raised
		border = ui.th.Olive
	} else if click.Hovered() {
		bg = ui.th.Raised
	}

	cpu := "—"
	if p.CPUKnown {
		cpu = fmt.Sprintf("%.1f%%", p.CPUPercent)
	}

	userCol := ui.th.Ink
	if p.User == "root" || p.UID == 0 {
		userCol = ui.th.Stamp
	}

	return layout.Inset{Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, bg, border, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						// PID
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
							return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, strconv.Itoa(p.PID))
						}),
						layout.Rigid(hgap(unit.Dp(8))),
						// Process Name
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(150))
							name := p.Name
							if name == "" {
								name = "—"
							}
							return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, name)
						}),
						layout.Rigid(hgap(unit.Dp(8))),
						// User
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
							user := p.User
							if user == "" {
								user = "—"
							}
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, userCol, user)
						}),
						layout.Rigid(hgap(unit.Dp(8))),
						// RSS Memory
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, formatBytes(p.RSSBytes))
						}),
						layout.Rigid(hgap(unit.Dp(8))),
						// CPU
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(55))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(55))
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, cpu)
						}),
						layout.Rigid(hgap(unit.Dp(8))),
						// Sockets Badge
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							count := len(p.SocketKeys)
							sockText := fmt.Sprintf("%d sockets", count)
							if count == 1 {
								sockText = "1 socket"
							}
							return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Olive, sockText)
									})
								})
							})
						}),
					)
				})
			})
		})
	})
}

func (ui *UI) processDetail(gtx layout.Context, p *processes.Process) layout.Dimensions {
	if p == nil {
		return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(30), Bottom: unit.Dp(30), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.wrap(gtx, unit.Sp(13), ui.th.Mute, "Select a process from the list to inspect attributes, socket descriptors, and manage signals.")
				})
			})
		})
	}

	cpuStr := "—"
	if p.CPUKnown {
		cpuStr = fmt.Sprintf("%.1f%%", p.CPUPercent)
	}

	propCard := func(label, value string) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Mute, label)
							}),
							layout.Rigid(gap(unit.Dp(2))),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.wrap(gtx, unit.Sp(12), ui.th.Ink, value)
							}),
						)
					})
				})
			})
		}
	}

	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// Detail elements list
			var elements []layout.Widget

			// 1. Process Header
			elements = append(elements, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(18), font.Bold, theme.Sans, ui.th.Ink, empty(p.Name, "Process "+strconv.Itoa(p.PID)))
					}),
					layout.Rigid(gap(unit.Dp(6))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							// PID Badge
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return plate(gtx, ui.th.Raised, ui.th.Olive, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, fmt.Sprintf("PID %d", p.PID))
									})
								})
							}),
							layout.Rigid(hgap(unit.Dp(8))),
							// PPID Badge
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, fmt.Sprintf("PPID %d", p.PPID))
									})
								})
							}),
							layout.Rigid(hgap(unit.Dp(8))),
							// User Badge
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								uCol := ui.th.Ink
								if p.User == "root" || p.UID == 0 {
									uCol = ui.th.Stamp
								}
								return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, uCol, fmt.Sprintf("USER: %s (UID %d)", empty(p.User, "unknown"), p.UID))
									})
								})
							}),
						)
					}),
				)
			})

			elements = append(elements, func(gtx layout.Context) layout.Dimensions { return gap(unit.Dp(10))(gtx) })

			// 2. Process Actions (Signals)
			elements = append(elements, func(gtx layout.Context) layout.Dimensions {
				return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, ui.th.Mute, "PROCESS SIGNALS")
							}),
							layout.Rigid(gap(unit.Dp(6))),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return ui.button(gtx, &ui.termBtn, "SIGTERM", false)
									}),
									layout.Rigid(hgap(unit.Dp(8))),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return ui.dangerButton(gtx, &ui.killBtn, "SIGKILL")
									}),
								)
							}),
						)
					})
				})
			})

			// 3. Service Management if connected to a Systemd unit
			if p.Service != "" {
				elements = append(elements, func(gtx layout.Context) layout.Dimensions { return gap(unit.Dp(6))(gtx) })
				elements = append(elements, func(gtx layout.Context) layout.Dimensions {
					return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, ui.th.Mute, "SERVICE MANAGEMENT: "+p.Service)
								}),
								layout.Rigid(gap(unit.Dp(6))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.procSvc[0], "STOP SERVICE", false)
										}),
										layout.Rigid(hgap(unit.Dp(8))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.procSvc[1], "RESTART SERVICE", false)
										}),
									)
								}),
							)
						})
					})
				})
			}

			elements = append(elements, func(gtx layout.Context) layout.Dimensions { return gap(unit.Dp(10))(gtx) })

			// 4. Properties List
			elements = append(elements, propCard("SYSTEM SERVICE", empty(p.Service, "None detected (standalone process)")))
			elements = append(elements, propCard("CONTAINER", empty(p.Container, "Host system")))
			elements = append(elements, propCard("NETWORK NAMESPACE", empty(p.NetNS, "Default / Host")))
			elements = append(elements, propCard("MEMORY (RSS)", fmt.Sprintf("%s (%d bytes)", formatBytes(p.RSSBytes), p.RSSBytes)))
			elements = append(elements, propCard("CPU USAGE", cpuStr))
			elements = append(elements, propCard("START TICKS", fmt.Sprintf("%d ticks", p.StartTicks)))
			elements = append(elements, propCard("EXECUTABLE PATH", empty(p.Executable, "Not available")))
			elements = append(elements, propCard("COMMAND LINE", empty(p.Command, "Not available")))

			// 5. Open Sockets
			sockCount := len(p.SocketKeys)
			sockSummary := fmt.Sprintf("%d open sockets held by this process", sockCount)
			if sockCount > 0 {
				sockSummary += ":\n" + strings.Join(p.SocketKeys, "\n")
			}
			elements = append(elements, propCard("SOCKET DESCRIPTORS", sockSummary))

			// Render in a scrollable list inside the inspector
			return ui.vlist(gtx, &ui.procDetailList, len(elements), func(gtx layout.Context, i int) layout.Dimensions {
				return elements[i](gtx)
			})
		})
	})
}

var svcFilterLabels = []string{"ALL", "RUNNING", "INACTIVE", "FAILED"}

func isServiceRunning(s services.Unit) bool {
	act := strings.ToLower(strings.TrimSpace(s.Active))
	sub := strings.ToLower(strings.TrimSpace(s.Sub))
	load := strings.ToLower(strings.TrimSpace(s.Load))
	return (act == "active" && sub == "running") || act == "running" || sub == "running" || load == "running"
}

func isServiceFailed(s services.Unit) bool {
	act := strings.ToLower(strings.TrimSpace(s.Active))
	sub := strings.ToLower(strings.TrimSpace(s.Sub))
	load := strings.ToLower(strings.TrimSpace(s.Load))
	return act == "failed" || sub == "failed" || load == "failed"
}

func isServiceExited(s services.Unit) bool {
	sub := strings.ToLower(strings.TrimSpace(s.Sub))
	return sub == "exited"
}

func (ui *UI) services(gtx layout.Context, view app.View) layout.Dimensions {
	if len(view.Services) == 0 {
		return ui.notice(gtx, "No service list has been read yet. On systemd this uses an unprivileged systemctl list. Start and stop still ask for authorization.")
	}

	for i := range ui.svcFilterBtns {
		if ui.svcFilterBtns[i].Clicked(gtx) {
			ui.svcFilter = i
		}
	}

	ops := []string{"start", "stop", "restart", "enable", "disable"}
	labels := []string{"START", "STOP", "RESTART", "ENABLE", "DISABLE"}
	for i, op := range ops {
		if ui.svcBtn[i].Clicked(gtx) && ui.selectedSvc != "" && view.Busy == "" {
			name := ui.selectedSvc
			ui.confirm = &confirm{
				text: "Run systemctl " + op + " " + name + "? This is a service action, not a process signal.",
				msg:  privilege.Message{Op: "service", Service: name, ServiceOp: op},
			}
		}
	}

	totalCount := len(view.Services)
	runningCount, inactiveCount, failedCount := 0, 0, 0
	var selectedUnit *services.Unit
	for i := range view.Services {
		s := &view.Services[i]
		if s.Name == ui.selectedSvc {
			selectedUnit = s
		}
		if isServiceRunning(*s) {
			runningCount++
		} else if isServiceFailed(*s) {
			failedCount++
		} else {
			inactiveCount++
		}
	}

	search := strings.ToLower(strings.TrimSpace(ui.svcSearch.Text()))
	var filtered []services.Unit
	for _, svc := range view.Services {
		switch ui.svcFilter {
		case 1:
			if !isServiceRunning(svc) {
				continue
			}
		case 2:
			if isServiceRunning(svc) || isServiceFailed(svc) {
				continue
			}
		case 3:
			if !isServiceFailed(svc) {
				continue
			}
		}

		if search != "" {
			match := strings.Contains(strings.ToLower(svc.Name), search) ||
				strings.Contains(strings.ToLower(svc.Description), search) ||
				strings.Contains(strings.ToLower(svc.Sub), search)
			if !match {
				continue
			}
		}
		filtered = append(filtered, svc)
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// 1. Search Bar & Status Filter Tabs
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					ed := material.Editor(ui.th.Material, &ui.svcSearch, "Search services by unit name or description...")
					ed.TextSize = unit.Sp(13)
					ed.Color = ui.th.Ink
					ed.HintColor = ui.th.Mute
					ed.Font.Typeface = theme.Mono
					return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, ed.Layout)
					})
				}),
				layout.Rigid(hgap(unit.Dp(10))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					kids := make([]layout.FlexChild, 0, len(svcFilterLabels))
					for i, lbl := range svcFilterLabels {
						i, lbl := i, lbl
						kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.button(gtx, &ui.svcFilterBtns[i], lbl, ui.svcFilter == i)
						}))
						if i < len(svcFilterLabels)-1 {
							kids = append(kids, layout.Rigid(hgap(unit.Dp(6))))
						}
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
				}),
			)
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 2. Statistics & Management Control Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Ink, fmt.Sprintf("TOTAL: %d", totalCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, fmt.Sprintf("RUNNING: %d", runningCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("INACTIVE: %d", inactiveCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Signal, fmt.Sprintf("FAILED: %d", failedCount))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if selectedUnit != nil {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "SELECTED:")
									}),
									layout.Rigid(hgap(unit.Dp(6))),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, selectedUnit.Name)
									}),
								)
							}
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Sans, ui.th.Mute, "Click a service below to manage")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							kids := make([]layout.FlexChild, 0, len(labels))
							for i, lbl := range labels {
								i, lbl := i, lbl
								kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.button(gtx, &ui.svcBtn[i], lbl, false)
								}))
								if i < len(labels)-1 {
									kids = append(kids, layout.Rigid(hgap(unit.Dp(4))))
								}
							}
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),

		// 3. Table Column Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "STATUS")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(260))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(260))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "SERVICE UNIT")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "SUB-STATE")
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Sans, ui.th.Mute, "DESCRIPTION")
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(4))),

		// 4. Service Rows List
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(filtered) == 0 {
				return ui.notice(gtx, "No services matching the current filter.")
			}
			return ui.vlist(gtx, &ui.svcList, len(filtered), func(gtx layout.Context, i int) layout.Dimensions {
				click := ui.rowClick(&ui.svcRow, i)
				svc := filtered[i]
				if click.Clicked(gtx) {
					if ui.selectedSvc == svc.Name {
						ui.selectedSvc = ""
					} else {
						ui.selectedSvc = svc.Name
					}
				}
				return ui.serviceRow(gtx, svc, svc.Name == ui.selectedSvc, click)
			})
		}),
	)
}

func (ui *UI) serviceRow(gtx layout.Context, svc services.Unit, selected bool, click *widget.Clickable) layout.Dimensions {
	gtx.Constraints.Min.Y = 0

	statusText := "INACTIVE"
	statusCol := ui.th.Mute
	statusBg := ui.th.Plate

	if isServiceRunning(svc) {
		statusText = "RUNNING"
		statusCol = ui.th.Olive
		statusBg = ui.th.Raised
	} else if isServiceFailed(svc) {
		statusText = "FAILED"
		statusCol = ui.th.Signal
		statusBg = ui.th.Raised
	} else if isServiceExited(svc) {
		statusText = "EXITED"
		statusCol = ui.th.Stamp
		statusBg = ui.th.Raised
	}

	bg := ui.th.Plate
	border := ui.th.Rule
	if selected {
		bg = ui.th.Raised
		border = ui.th.Olive
	} else if click.Hovered() {
		bg = ui.th.Raised
	}

	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return plate(gtx, bg, border, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					// Status Badge
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
						return plate(gtx, statusBg, statusCol, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, statusCol, statusText)
								})
							})
						})
					}),
					layout.Rigid(hgap(unit.Dp(12))),
					// Service Unit Name
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(260))
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(260))
						return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, svc.Name)
					}),
					layout.Rigid(hgap(unit.Dp(12))),
					// Sub-state
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(80))
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(80))
						return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, svc.Sub)
					}),
					layout.Rigid(hgap(unit.Dp(12))),
					// Description
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						desc := svc.Description
						if desc == "" {
							desc = "—"
						}
						return ui.label(gtx, unit.Sp(12), font.Normal, theme.Sans, ui.th.Mute, desc)
					}),
				)
			})
		})
	})
}

func (ui *UI) containers(gtx layout.Context, view app.View) layout.Dimensions {
	ui.app.WantContainers()
	if len(view.Containers) == 0 {
		return ui.notice(gtx, "No container or libvirt object was listed. Docker is queried only after this screen is opened, and nothing here changes container networking.")
	}
	return ui.vlist(gtx, &ui.boxList, len(view.Containers), func(gtx layout.Context, i int) layout.Dimensions {
		item := view.Containers[i]
		return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				kids := []layout.FlexChild{
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(14), font.Bold, theme.Sans, ui.th.Ink, strings.ToUpper(item.Kind)+"  "+empty(item.Name, item.ID))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ui.th.Mute, empty(item.Status, "status unknown")+"  "+item.Image)
					}),
					layout.Rigid(gap(unit.Dp(6))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Stamp, "PUBLISHED")
					}),
				}
				if len(item.Published) == 0 {
					kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.wrap(gtx, unit.Sp(13), ui.th.Mute, "No published port. A container port is not a host socket.")
					}))
				}
				for _, pub := range item.Published {
					pub := pub
					kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(13), font.Normal, theme.Mono, ui.th.Ink, "host "+pub.Host+"  container "+pub.Container+"/"+pub.Proto)
					}))
				}
				kids = append(kids,
					layout.Rigid(gap(unit.Dp(4))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, "HOST SOCKETS")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.wrap(gtx, unit.Sp(13), ui.th.Ink, empty(item.Host, "No host socket cgroup matched this container."))
					}),
				)
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
			})
		})
	})
}

func (ui *UI) activity(gtx layout.Context, view app.View) layout.Dimensions {
	show := make([]int, 0, len(view.History))
	q := strings.ToLower(strings.TrimSpace(ui.actFilter.Text()))
	for i, e := range view.History {
		blob := strings.ToLower(e.Action + " " + e.Target + " " + e.Detail + " " + e.Protocol + " " + strconv.Itoa(e.Port))
		if q == "" || strings.Contains(blob, q) {
			show = append(show, i)
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.field(gtx, &ui.actFilter, "filter by action, port, or target")
		}),
		layout.Rigid(gap(unit.Dp(8))),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(show) == 0 {
				return ui.notice(gtx, "No audit rows match.")
			}
			return ui.vlist(gtx, &ui.actList, len(show), func(gtx layout.Context, i int) layout.Dimensions {
				e := view.History[show[i]]
				line := fmt.Sprintf("%s  %s  %s  %s  %s", e.RecordedAt.Local().Format(time.RFC3339), e.Action, e.Result, e.Target, e.Detail)
				return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ui.th.Ink, line)
					})
				})
			})
		}),
	)
}

func (ui *UI) settings(gtx layout.Context, view app.View) layout.Dimensions {
	if !ui.settingsDirty {
		ui.draftTheme = view.Theme
		if ui.draftTheme == "" {
			ui.draftTheme = config.ThemeDark
		}
		if ui.logRetentionDraft == 0 {
			ui.logRetentionDraft = view.LogRetentionDays
			if ui.logRetentionDraft <= 0 {
				ui.logRetentionDraft = config.DefaultLogRetentionDays
			}
			ui.logRetentionEditor.SetText(strconv.Itoa(ui.logRetentionDraft))
		}
	}

	if n, err := strconv.Atoi(strings.TrimSpace(ui.logRetentionEditor.Text())); err == nil && n >= 1 && n <= 365 {
		if n != ui.logRetentionDraft {
			ui.logRetentionDraft = n
			ui.settingsDirty = true
		}
	}

	presets := []int{1, 3, 7, 14, 30}
	for i, p := range presets {
		if ui.logPresetBtns[i].Clicked(gtx) {
			ui.logRetentionDraft = p
			ui.logRetentionEditor.SetText(strconv.Itoa(p))
			ui.settingsDirty = true
		}
	}

	if ui.logRotateNowBtn.Clicked(gtx) {
		if ui.app != nil {
			_ = ui.app.RotateLogs()
		}
	}

	if ui.saveBtn.Clicked(gtx) {
		_ = ui.app.SetTheme(ui.draftTheme)
		if ui.logRetentionDraft > 0 {
			_ = ui.app.SetLogRetention(ui.logRetentionDraft)
		}
		ui.settingsDirty = false
	}

	logSizeStr := "0 B"
	if view.LogPath != "" {
		if fi, err := os.Stat(view.LogPath); err == nil {
			logSizeStr = formatOctets(uint64(fi.Size()))
		}
	}

	kids := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.wrap(gtx, unit.Sp(14), ui.th.Ink, "Changes on this screen are kept until you press Save.")
		}),
		layout.Rigid(gap(unit.Dp(12))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Mute, "THEME")
		}),
		layout.Rigid(gap(unit.Dp(6))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ui.themeDraft[0].Clicked(gtx) && ui.draftTheme != config.ThemeDark {
				ui.draftTheme = config.ThemeDark
				ui.settingsDirty = true
			}
			if ui.themeDraft[1].Clicked(gtx) && ui.draftTheme != config.ThemeLight {
				ui.draftTheme = config.ThemeLight
				ui.settingsDirty = true
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.button(gtx, &ui.themeDraft[0], "DARK", ui.draftTheme == config.ThemeDark)
				}),
				layout.Rigid(hgap(unit.Dp(6))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.button(gtx, &ui.themeDraft[1], "LIGHT", ui.draftTheme == config.ThemeLight)
				}),
			)
		}),
		layout.Rigid(gap(unit.Dp(16))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Mute, "DATA STREAMING")
		}),
		layout.Rigid(gap(unit.Dp(6))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(13), font.Bold, theme.Mono, ui.th.Olive, "● LIVE REALTIME STREAMING (CONTINUOUS)")
						}),
						layout.Rigid(gap(unit.Dp(4))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.wrap(gtx, unit.Sp(12), ui.th.Mute, "All sockets, bandwidth counters, topology connections, and processes update continuously in sub-second real-time.")
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(16))),
		// Log Retention & Rotation
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Mute, "LOG RETENTION & ROTATION")
		}),
		layout.Rigid(gap(unit.Dp(6))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.wrap(gtx, unit.Sp(12), ui.th.Mute, "Automatically prune internal PortPilot application logs older than the retention period.")
						}),
						layout.Rigid(gap(unit.Dp(8))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								// Quick Presets
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									kids := make([]layout.FlexChild, 0, len(presets))
									for i, p := range presets {
										i, p := i, p
										lbl := fmt.Sprintf("%d DAYS", p)
										if p == 1 {
											lbl = "1 DAY"
										}
										kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.logPresetBtns[i], lbl, ui.logRetentionDraft == p)
										}))
										if i < len(presets)-1 {
											kids = append(kids, layout.Rigid(hgap(unit.Dp(6))))
										}
									}
									return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
								}),
								layout.Rigid(hgap(unit.Dp(12))),
								// Custom Day Input
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(70))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(70))
									ed := material.Editor(ui.th.Material, &ui.logRetentionEditor, "Days")
									ed.TextSize = unit.Sp(13)
									ed.Color = ui.th.Ink
									ed.HintColor = ui.th.Mute
									ed.Font.Typeface = theme.Mono
									return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, ed.Layout)
									})
								}),
								layout.Rigid(hgap(unit.Dp(6))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ui.th.Mute, "days")
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return ui.button(gtx, &ui.logRotateNowBtn, "ROTATE NOW", false)
									})
								}),
							)
						}),
						layout.Rigid(gap(unit.Dp(6))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							infoText := fmt.Sprintf("Current log file: %s (%s)", view.LogPath, logSizeStr)
							return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, infoText)
						}),
					)
				})
			})
		}),
	}
	label := "SAVED"
	if ui.settingsDirty {
		label = "SAVE"
	}
	kids = append(kids,
		layout.Rigid(gap(unit.Dp(16))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.button(gtx, &ui.saveBtn, label, ui.settingsDirty)
		}),
		layout.Rigid(gap(unit.Dp(20))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Mute, "SYSTEM & STORAGE")
		}),
		layout.Rigid(gap(unit.Dp(6))),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// Version Row
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(130))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(130))
									return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Mute, "Version")
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, "PortPilot "+version.Number)
										})
									})
								}),
							)
						}),
						layout.Rigid(gap(unit.Dp(10))),
						// Config Path Row
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(130))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(130))
									return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Mute, "Config File")
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ui.th.Ink, view.ConfigPath)
								}),
							)
						}),
						layout.Rigid(gap(unit.Dp(10))),
						// Data Dir Row
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(130))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(130))
									return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Mute, "Data Directory")
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(12), font.Normal, theme.Mono, ui.th.Ink, view.DataDir)
								}),
							)
						}),
					)
				})
			})
		}),
	)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
}

func (ui *UI) stringList(gtx layout.Context, list *widget.List, lines []string) layout.Dimensions {
	return ui.vlist(gtx, list, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
		return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return ui.label(gtx, unit.Sp(13), font.Normal, theme.Mono, ui.th.Ink, lines[i])
			})
		})
	})
}

func (ui *UI) rowClick(rows *[]widget.Clickable, i int) *widget.Clickable {
	for len(*rows) <= i {
		*rows = append(*rows, widget.Clickable{})
	}
	return &(*rows)[i]
}

func findSocket(rows []ports.Socket, key string) *ports.Socket {
	if key == "" {
		return nil
	}
	for i := range rows {
		if rows[i].Key() == key {
			return &rows[i]
		}
	}
	return nil
}

func pidText(s ports.Socket) string {
	if !s.OwnerKnown || s.PID <= 0 {
		return "—"
	}
	return strconv.Itoa(s.PID)
}

func empty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func formatBytes(n uint64) string {
	if n == 0 {
		return "—"
	}
	const kb = 1024
	if n < 10*kb {
		return strconv.FormatUint(n, 10) + " B"
	}
	if n < 10*kb*kb {
		return strconv.FormatUint(n/kb, 10) + " KB"
	}
	return strconv.FormatUint(n/(kb*kb), 10) + " MB"
}
