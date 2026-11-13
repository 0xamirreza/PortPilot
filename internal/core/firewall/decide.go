package firewall

import (
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/ports"
)

var ifaceName = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,15}$`)

// Decide chooses the only backend allowed to change rules.
// An ambiguous host stays read-only. UFW, when active, is the only writer.
func Decide(r Report) (mode, reason string) {
	if !r.Authorized {
		if r.Error != "" {
			return ModeUnavailable, r.Error
		}
		return ModeUnavailable, "Live firewall rules have not been read. Authorization is required."
	}
	if r.Error != "" && !r.UFWKnown && !r.NftReadable {
		return ModeUnavailable, r.Error
	}
	if r.LegacyRules > 0 && (r.UFWActive || len(r.OtherNFTTables) > 0) {
		return ModeReadOnly, "iptables-legacy has rules beside another firewall. PortPilot will not choose between them."
	}
	if r.UFWActive {
		return ModeUFW, "UFW is active, so only UFW may change rules."
	}
	if r.UFWKnown && !r.UFWActive && len(r.OtherNFTTables) > 0 {
		return ModeReadOnly, "UFW is inactive and other nftables tables exist. Ownership is ambiguous, so changes stay disabled."
	}
	if r.NftReadable && r.NftInstalled && len(r.OtherNFTTables) == 0 && r.LegacyRules == 0 {
		return ModeNFTables, "UFW is not active and no other nftables table was found. Changes stay inside inet portpilot."
	}
	if !r.UFWInstalled && !r.NftInstalled {
		return ModeUnavailable, "Neither UFW nor nft was found."
	}
	return ModeReadOnly, "The firewall layout is not one PortPilot can change safely."
}

// SSHPorts collects TCP listen ports that belong to SSH, plus port 22.
func SSHPorts(socks []ports.Socket) map[int]bool {
	out := map[int]bool{22: true}
	for _, s := range socks {
		if s.Protocol != ports.TCP || !s.IsBoundService() {
			continue
		}
		blob := strings.ToLower(s.Process + " " + s.Service + " " + s.Executable)
		if strings.Contains(blob, "sshd") || strings.Contains(blob, "ssh.service") {
			out[int(s.LocalPort)] = true
		}
	}
	return out
}

// Validate checks a change before any command is built.
func Validate(req Request, ssh map[int]bool) error {
	switch req.Action {
	case ActionAllow, ActionDeny, ActionReject:
	default:
		return fmt.Errorf("action must be allow, deny, or reject")
	}
	switch strings.ToLower(req.Protocol) {
	case "tcp", "udp":
	default:
		return fmt.Errorf("protocol must be tcp or udp")
	}
	if req.Port < 1 || req.Port > 65535 {
		return fmt.Errorf("port must be from 1 to 65535")
	}
	switch req.Family {
	case "", "ipv4", "ipv6", "both":
	default:
		return fmt.Errorf("family must be ipv4, ipv6, or both")
	}
	if req.Source != "" {
		if _, err := netip.ParsePrefix(req.Source); err != nil {
			addr, err2 := netip.ParseAddr(req.Source)
			if err2 != nil {
				return fmt.Errorf("source must be an IP address or CIDR")
			}
			if addr.IsLoopback() {
				return fmt.Errorf("loopback addresses are not firewall targets")
			}
		} else if p, _ := netip.ParsePrefix(req.Source); p.Addr().IsLoopback() {
			return fmt.Errorf("loopback addresses are not firewall targets")
		}
	}
	if req.Interface != "" {
		if !ifaceName.MatchString(req.Interface) || req.Interface == "lo" {
			return fmt.Errorf("interface %q is not accepted", req.Interface)
		}
	}
	if req.DurationSec != 0 && (req.DurationSec < 60 || req.DurationSec > 7*24*3600) {
		return fmt.Errorf("duration must be from 1 minute to 7 days")
	}
	if req.Action != ActionAllow && ssh[req.Port] && !req.ConfirmSSH {
		return fmt.Errorf("port %d looks like SSH; confirm that blocking it may disconnect this session", req.Port)
	}
	return nil
}

// Expires returns the absolute expiry for a duration, or the zero time.
func Expires(now time.Time, seconds int) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(seconds) * time.Second)
}

// OwnedComment reports whether the comment was written by PortPilot.
func OwnedComment(comment string) bool {
	return strings.HasPrefix(comment, CommentPrefix) && Rule{Comment: comment}.ID() != ""
}

// FormatPort is the decimal port used in commands.
func FormatPort(port int) string { return strconv.Itoa(port) }
