package gui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/core/firewall"
	"github.com/0xamirreza/portpilot/internal/gui/theme"
	"github.com/0xamirreza/portpilot/internal/system/privilege"
)

type confirm struct {
	text string
	msg  privilege.Message
}

var durationChoices = []struct {
	Label string
	Sec   int
}{
	{"PERMANENT", 0},
	{"5M", 5 * 60},
	{"15M", 15 * 60},
	{"30M", 30 * 60},
	{"1H", 3600},
	{"6H", 6 * 3600},
	{"24H", 24 * 3600},
}

func (ui *UI) firewallPage(gtx layout.Context, view app.View, editor bool) layout.Dimensions {
	rep := view.Firewall
	if ui.reconcile.Clicked(gtx) && view.Busy == "" {
		ui.app.Privileged(privilege.Message{Op: "reconcile"})
	}
	if ui.rollback.Clicked(gtx) && view.Busy == "" {
		ui.confirm = &confirm{
			text: "Roll back the whole PortPilot snapshot? Every rule PortPilot added is removed. Rules from other managers stay.",
			msg:  privilege.Message{Op: "rollback"},
		}
	}
	if ui.deleteBtn.Clicked(gtx) && ui.selectedRule != "" && view.Busy == "" {
		ui.confirm = &confirm{
			text: "Delete the PortPilot rule " + ui.selectedRule + "? Rules that PortPilot did not create stay untouched.",
			msg:  privilege.Message{Op: "delete", RuleID: ui.selectedRule},
		}
	}
	if ui.disableBtn.Clicked(gtx) && ui.selectedRule != "" && view.Busy == "" {
		ui.confirm = &confirm{
			text: "Disable " + ui.selectedRule + "? It leaves the firewall and can be enabled again from this list.",
			msg:  privilege.Message{Op: "delete", RuleID: ui.selectedRule, Keep: true},
		}
	}
	if editor {
		ui.ruleClicks(gtx, view)
	}
	status := rep.Mode
	if status == "" {
		status = firewall.ModeUnavailable
	}
	head := strings.ToUpper(status)
	if view.Busy != "" {
		head = view.Busy
	}

	modeCol := ui.th.Ink
	if rep.UFWActive || rep.Mode == firewall.ModeUFW || rep.Mode == firewall.ModeNFTables {
		modeCol = ui.th.Olive
	} else if rep.Mode == firewall.ModeUnavailable {
		modeCol = ui.th.Signal
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// 1. Backend Status & Action Controls Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						// Backend Badge
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return plate(gtx, ui.th.Raised, modeCol, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, modeCol, "FIREWALL: "+head)
								})
							})
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						// Sub reason
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(12), font.Normal, theme.Sans, ui.th.Ink, rep.Reason)
						}),
						layout.Rigid(hgap(unit.Dp(12))),
						// Action Buttons (RECONCILE & ROLLBACK)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.button(gtx, &ui.reconcile, "RECONCILE", false)
								}),
								layout.Rigid(hgap(unit.Dp(8))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.button(gtx, &ui.rollback, "ROLLBACK", false)
								}),
							)
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(6))),
		// 2. Metrics Bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Raised, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Olive, fmt.Sprintf("ACTIVE RULES: %d", len(rep.Rules)))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Stamp, fmt.Sprintf("SCHEDULES: %d", len(rep.Schedules)))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("UFW ACTIVE: %t", rep.UFWActive))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("UFW ON BOOT: %t", rep.UFWEnabledOnBoot))
						}),
						layout.Rigid(hgap(unit.Dp(14))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, fmt.Sprintf("NFT TABLES: %d", len(rep.OtherNFTTables)))
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(6))),
		// 3. Wildcard warning notice
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.wrap(gtx, unit.Sp(12), ui.th.Stamp, "A wildcard bind is not proof that another machine can reach the port. A firewall change is a separate fact.")
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(6))),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if editor {
				return ui.ruleEditor(gtx, view)
			}
			return ui.ruleList(gtx, rep, false)
		}),
	)
}

func (ui *UI) ruleClicks(gtx layout.Context, view app.View) {
	if ui.protoBtn[0].Clicked(gtx) {
		ui.protocol = "tcp"
	}
	if ui.protoBtn[1].Clicked(gtx) {
		ui.protocol = "udp"
	}
	for i := range durationChoices {
		if ui.durBtn[i].Clicked(gtx) {
			ui.duration = durationChoices[i].Sec
			ui.customEdit.SetText("")
		}
	}
	if ui.famBtn[0].Clicked(gtx) {
		ui.family = "both"
	}
	if ui.famBtn[1].Clicked(gtx) {
		ui.family = "ipv4"
	}
	if ui.famBtn[2].Clicked(gtx) {
		ui.family = "ipv6"
	}
	if view.Busy != "" {
		return
	}
	if ui.allowBtn.Clicked(gtx) {
		ui.askRule(view, firewall.ActionAllow)
	}
	if ui.denyBtn.Clicked(gtx) {
		ui.askRule(view, firewall.ActionDeny)
	}
	if ui.rejectBtn.Clicked(gtx) {
		ui.askRule(view, firewall.ActionReject)
	}
}

func (ui *UI) askRule(view app.View, action string) {
	port, err := strconv.Atoi(strings.TrimSpace(ui.portEdit.Text()))
	if err != nil {
		ui.app.Store().SetBanner("Enter a port from 1 to 65535.")
		return
	}
	seconds := ui.duration
	if text := strings.TrimSpace(ui.customEdit.Text()); text != "" {
		mins, err := strconv.Atoi(text)
		if err != nil || mins < 1 {
			ui.app.Store().SetBanner("Custom duration is a number of minutes, from 1 to 10080.")
			return
		}
		seconds = mins * 60
	}
	req := firewall.Request{
		Action: action, Protocol: ui.protocol, Port: port, Family: ui.family,
		Source: strings.TrimSpace(ui.sourceEdit.Text()), Interface: strings.TrimSpace(ui.ifaceEdit.Text()),
		DurationSec: seconds,
	}
	ssh := firewall.SSHPorts(view.Sockets)[port]
	text := fmt.Sprintf("%s %s/%d %s", strings.ToUpper(action), ui.protocol, port, ui.family)
	if req.Source != "" {
		text += " from " + req.Source
	}
	if req.Interface != "" {
		text += " on " + req.Interface
	}
	if seconds > 0 {
		text += fmt.Sprintf(" for %s", time.Duration(seconds)*time.Second)
	} else {
		text += " until you delete it"
	}
	text += "? "
	if ssh && action != firewall.ActionAllow {
		text += "This port looks like SSH. Blocking it can disconnect the session that is managing this host. "
		req.ConfirmSSH = true
	}
	if view.Firewall.Mode != firewall.ModeUFW && view.Firewall.Mode != firewall.ModeNFTables {
		text += "The helper will refuse the change unless a safe backend is active. "
	}
	text += "Loopback is never a target."
	ui.confirm = &confirm{text: text, msg: privilege.Message{Op: "apply", Rule: req}}
}

func (ui *UI) ruleEditor(gtx layout.Context, view app.View) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Form Card
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// Row 1: Target Port + Protocol + Family
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(150))
									return ui.field(gtx, &ui.portEdit, "Port (1-65535)")
								}),
								layout.Rigid(hgap(unit.Dp(12))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.protoBtn[0], "TCP", ui.protocol == "tcp")
										}),
										layout.Rigid(hgap(unit.Dp(4))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.protoBtn[1], "UDP", ui.protocol == "udp")
										}),
									)
								}),
								layout.Rigid(hgap(unit.Dp(12))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.famBtn[0], "BOTH", ui.family == "both")
										}),
										layout.Rigid(hgap(unit.Dp(4))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.famBtn[1], "IPv4", ui.family == "ipv4")
										}),
										layout.Rigid(hgap(unit.Dp(4))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.famBtn[2], "IPv6", ui.family == "ipv6")
										}),
									)
								}),
							)
						}),
						layout.Rigid(gap(unit.Dp(8))),
						// Row 2: Source IP/CIDR + Interface
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1.2, func(gtx layout.Context) layout.Dimensions {
									return ui.field(gtx, &ui.sourceEdit, "Source IP or CIDR (leave empty for ANY)")
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return ui.field(gtx, &ui.ifaceEdit, "Interface (e.g. eth0, empty = any except lo)")
								}),
							)
						}),
						layout.Rigid(gap(unit.Dp(8))),
						// Row 3: Duration presets + custom minutes
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, ui.th.Mute, "DURATION: ")
								}),
								layout.Rigid(hgap(unit.Dp(6))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									kids := make([]layout.FlexChild, 0, len(durationChoices)*2)
									for i := range durationChoices {
										i := i
										kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.durBtn[i], durationChoices[i].Label, ui.duration == durationChoices[i].Sec)
										}))
										if i < len(durationChoices)-1 {
											kids = append(kids, layout.Rigid(hgap(unit.Dp(4))))
										}
									}
									return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, kids...)
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
									return ui.field(gtx, &ui.customEdit, "Custom mins")
								}),
							)
						}),
						layout.Rigid(gap(unit.Dp(10))),
						// Row 4: Primary Action Buttons + Selected Rule actions
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.successButton(gtx, &ui.allowBtn, "ALLOW RULE")
								}),
								layout.Rigid(hgap(unit.Dp(8))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.dangerButton(gtx, &ui.denyBtn, "DENY RULE")
								}),
								layout.Rigid(hgap(unit.Dp(8))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return ui.warningButton(gtx, &ui.rejectBtn, "REJECT RULE")
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
								// Selection actions
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if ui.selectedRule == "" {
										return layout.Dimensions{}
									}
									return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.button(gtx, &ui.disableBtn, "DISABLE SELECTED", true)
										}),
										layout.Rigid(hgap(unit.Dp(8))),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return ui.dangerButton(gtx, &ui.deleteBtn, "DELETE SELECTED")
										}),
									)
								}),
							)
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(8))),
		// Table of Active Rules
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.ruleList(gtx, view.Firewall, true)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.disabledRules(gtx, view)
				}),
			)
		}),
	)
}

func (ui *UI) field(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
	w := material.Editor(ui.th.Material, ed, hint)
	w.Font = font.Font{Typeface: theme.Mono}
	w.Color = ui.th.Ink
	w.HintColor = ui.th.Mute
	return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, w.Layout)
	})
}

func (ui *UI) disabledRules(gtx layout.Context, view app.View) layout.Dimensions {
	var saved []int
	for i := range view.Owned {
		if view.Owned[i].Status == "disabled" {
			saved = append(saved, i)
		}
	}
	if len(saved) == 0 {
		return layout.Dimensions{}
	}
	return plate(gtx, ui.th.Plate, ui.th.Stamp, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			kids := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Stamp, "DISABLED RULES (CLICK TO RE-ENABLE)")
				}),
				layout.Rigid(gap(unit.Dp(6))),
			}
			for n, idx := range saved {
				n, idx := n, idx
				rule := view.Owned[idx]
				click := ui.rowClick(&ui.ownedRow, n)
				if click.Clicked(gtx) && view.Busy == "" {
					req := firewall.Request{
						Action: rule.Action, Protocol: rule.Protocol, Port: rule.Port, Family: rule.Family,
						Source: rule.Source, Interface: rule.Iface, DurationSec: rule.DurationSec,
					}
					text := fmt.Sprintf("Enable %s %s/%d again?", strings.ToUpper(rule.Action), rule.Protocol, rule.Port)
					if firewall.SSHPorts(view.Sockets)[rule.Port] && rule.Action != firewall.ActionAllow {
						req.ConfirmSSH = true
						text += " This port looks like SSH."
					}
					ui.confirm = &confirm{text: text, msg: privilege.Message{Op: "apply", Rule: req}}
				}
				bg := ui.th.Raised
				if click.Hovered() {
					bg = ui.th.Plate
				}
				line := fmt.Sprintf("DISABLED  %s %s/%d  %s", strings.ToUpper(rule.Action), strings.ToUpper(rule.Protocol), rule.Port, rule.Identity)
				kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return plate(gtx, bg, ui.th.Stamp, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Ink, line)
								})
							})
						})
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
		})
	})
}

func (ui *UI) ruleList(gtx layout.Context, rep firewall.Report, selectable bool) layout.Dimensions {
	if len(rep.Rules) == 0 && len(rep.Schedules) == 0 {
		msg := "No firewall rules observed in this scan."
		if !rep.Authorized {
			msg = "No rules have been read. Click the AUTHORIZE button in the top header to load live UFW and nftables rules."
		} else if rep.UFWInstalled && !rep.UFWActive {
			msg = "No firewall rules observed. UFW is installed but inactive (UFW ACTIVE: false). Run 'sudo ufw enable' in terminal to activate UFW and enable full rule management."
		}
		return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(24), Bottom: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ui.wrap(gtx, unit.Sp(13), ui.th.Mute, msg)
				})
			})
		})
	}

	n := len(rep.Rules) + len(rep.Schedules)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Table Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return plate(gtx, ui.th.Plate, ui.th.Rule, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "BACKEND")
						}),
						layout.Rigid(hgap(unit.Dp(10))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "ACTION")
						}),
						layout.Rigid(hgap(unit.Dp(10))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "PROTOCOL")
						}),
						layout.Rigid(hgap(unit.Dp(10))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "PORT")
						}),
						layout.Rigid(hgap(unit.Dp(10))),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(160))
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "SOURCE IP / CIDR")
						}),
						layout.Rigid(hgap(unit.Dp(10))),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, "SCOPE / OWNERSHIP")
						}),
					)
				})
			})
		}),
		layout.Rigid(gap(unit.Dp(4))),
		// Table Rows
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return ui.vlist(gtx, &ui.fwList, n, func(gtx layout.Context, i int) layout.Dimensions {
				if i < len(rep.Rules) {
					rule := rep.Rules[i]
					click := ui.rowClick(&ui.fwRow, i)
					if selectable && click.Clicked(gtx) && rule.Owned {
						if ui.selectedRule == rule.ID() {
							ui.selectedRule = ""
						} else {
							ui.selectedRule = rule.ID()
						}
					}

					actionCol := ui.th.Olive
					actionText := strings.ToUpper(rule.Action)
					if rule.Action == firewall.ActionDeny {
						actionCol = ui.th.Signal
					} else if rule.Action == firewall.ActionReject {
						actionCol = ui.th.Stamp
					}

					bg := ui.th.Plate
					border := ui.th.Rule
					isSelected := selectable && rule.ID() != "" && rule.ID() == ui.selectedRule
					if isSelected {
						bg = ui.th.Raised
						border = ui.th.Olive
					} else if click.Hovered() {
						bg = ui.th.Raised
					}

					srcText := "ANY (all sources)"
					srcCol := ui.th.Mute
					if rule.Source != "" && rule.Source != "any" {
						srcText = rule.Source
						srcCol = ui.th.Ink
					}

					return layout.Inset{Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return plate(gtx, bg, border, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										// Backend
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
											return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, strings.ToUpper(rule.Backend))
										}),
										layout.Rigid(hgap(unit.Dp(10))),
										// Action Badge
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
											return plate(gtx, ui.th.Raised, actionCol, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, actionCol, actionText)
													})
												})
											})
										}),
										layout.Rigid(hgap(unit.Dp(10))),
										// Protocol
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
											return ui.label(gtx, unit.Sp(11.5), font.Bold, theme.Mono, ui.th.Ink, strings.ToUpper(rule.Protocol))
										}),
										layout.Rigid(hgap(unit.Dp(10))),
										// Port
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
											return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, strconv.Itoa(rule.Port))
										}),
										layout.Rigid(hgap(unit.Dp(10))),
										// Source
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
											gtx.Constraints.Max.X = gtx.Dp(unit.Dp(160))
											return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, srcCol, srcText)
										}),
										layout.Rigid(hgap(unit.Dp(10))),
										// Ownership / Scope
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											if rule.Owned {
												return plate(gtx, ui.th.Raised, ui.th.Olive, func(gtx layout.Context) layout.Dimensions {
													return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
														return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Olive, "PORTPILOT OWNED: "+rule.ID())
													})
												})
											}
											return ui.label(gtx, unit.Sp(11), font.Normal, theme.Mono, ui.th.Mute, "SYSTEM / FOREIGN")
										}),
									)
								})
							})
						})
					})
				}

				// Scheduled temporary rule
				item := rep.Schedules[i-len(rep.Rules)]
				left := time.Until(item.Expires).Round(time.Second)
				if left < 0 {
					left = 0
				}
				return layout.Inset{Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return plate(gtx, ui.th.Plate, ui.th.Stamp, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
									return ui.label(gtx, unit.Sp(11), font.Bold, theme.Mono, ui.th.Mute, strings.ToUpper(item.Backend))
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
									return plate(gtx, ui.th.Raised, ui.th.Stamp, func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return ui.label(gtx, unit.Sp(10.5), font.Bold, theme.Mono, ui.th.Stamp, strings.ToUpper(item.Action))
											})
										})
									})
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(75))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(75))
									return ui.label(gtx, unit.Sp(11.5), font.Bold, theme.Mono, ui.th.Ink, strings.ToUpper(item.Protocol))
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min.X = gtx.Dp(unit.Dp(85))
									gtx.Constraints.Max.X = gtx.Dp(unit.Dp(85))
									return ui.label(gtx, unit.Sp(12), font.Bold, theme.Mono, ui.th.Ink, strconv.Itoa(item.Port))
								}),
								layout.Rigid(hgap(unit.Dp(10))),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return plate(gtx, ui.th.Raised, ui.th.Stamp, func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return ui.label(gtx, unit.Sp(10), font.Bold, theme.Mono, ui.th.Stamp, fmt.Sprintf("TEMPORARY RULE (ID: %s) — EXPIRES IN %s", item.ID, left))
										})
									})
								}),
							)
						})
					})
				})
			})
		}),
	)
}

func (ui *UI) confirmPlate(gtx layout.Context) layout.Dimensions {
	if ui.yes.Clicked(gtx) && ui.confirm != nil {
		msg := ui.confirm.msg
		ui.confirm = nil
		ui.app.Privileged(msg)
	}
	if ui.no.Clicked(gtx) {
		ui.confirm = nil
	}
	text := ""
	if ui.confirm != nil {
		text = ui.confirm.text
	}
	return layout.Inset{Top: unit.Dp(24), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return plate(gtx, ui.th.Plate, ui.th.Signal, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return ui.wrap(gtx, unit.Sp(16), ui.th.Ink, text)
					}),
					layout.Rigid(gap(unit.Dp(14))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.button(gtx, &ui.yes, "YES", true)
							}),
							layout.Rigid(hgap(unit.Dp(8))),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return ui.button(gtx, &ui.no, "NO", false)
							}),
						)
					}),
				)
			})
		})
	})
}
