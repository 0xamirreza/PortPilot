// Package firewall inspects the host and applies only the backend Decide allows.
package firewall

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	core "github.com/0xamirreza/portpilot/internal/core/firewall"
	"github.com/0xamirreza/portpilot/internal/firewall/nftables"
	"github.com/0xamirreza/portpilot/internal/firewall/ufw"
)

// Runner executes one fixed binary. Implementations must not start a shell.
type Runner interface {
	Run(name string, args ...string) (string, error)
}

// Preview reads files that do not require root.
func Preview() core.Report {
	r := core.Report{
		UFWInstalled:    exists("/usr/sbin/ufw"),
		NftInstalled:    exists("/usr/sbin/nft") || exists("/sbin/nft"),
		LegacyInstalled: exists("/usr/sbin/iptables-legacy") || exists("/sbin/iptables-legacy"),
		ReadAt:          time.Now(),
	}
	r.UFWEnabledOnBoot = enabledOnBoot("/etc/ufw/ufw.conf")
	r.Mode, r.Reason = core.Decide(r)
	return r
}

func exists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func enabledOnBoot(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "ENABLED=") {
			return strings.TrimPrefix(line, "ENABLED=") == "yes"
		}
	}
	return false
}

// Inspect runs read-only commands. Call it from the privileged helper.
func Inspect(run Runner) (core.Report, error) {
	r := Preview()
	r.Authorized = true
	var problems []string
	if r.UFWInstalled {
		out, err := run.Run("ufw", "status", "verbose")
		if err != nil {
			problems = append(problems, "ufw status: "+err.Error())
		} else {
			st := ufw.ParseStatus(out)
			r.UFWKnown = st.Known
			r.UFWActive = st.Active
			r.DefaultInbound = st.DefaultInbound
			r.DefaultOutbound = st.DefaultOutbound
			r.Rules = append(r.Rules, st.Rules...)
		}
	}
	if r.NftInstalled {
		out, err := run.Run("nft", "-j", "list", "ruleset")
		if err != nil {
			problems = append(problems, "nft list: "+err.Error())
		} else {
			sum, err := nftables.ParseJSON(out)
			if err != nil {
				problems = append(problems, err.Error())
			} else {
				r.NftReadable = true
				r.OtherNFTTables = nftables.OtherTables(sum)
				for _, rule := range sum.Rules {
					if rule.Table == "portpilot" {
						r.Rules = append(r.Rules, rule)
					}
				}
			}
		}
	}
	if r.LegacyInstalled {
		out, err := run.Run("iptables-legacy", "-S")
		if err == nil {
			r.LegacyRules = countLegacy(out)
		}
	}
	if len(problems) > 0 && !r.UFWKnown && !r.NftReadable {
		r.Error = strings.Join(problems, "; ")
	}
	r.Mode, r.Reason = core.Decide(r)
	r.ReadAt = time.Now()
	return r, nil
}

func countLegacy(text string) int {
	n := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-A ") {
			n++
		}
	}
	return n
}

// Apply adds one rule through the decided backend.
// The bool is true when nftables itself carries the expiration.
func Apply(run Runner, report core.Report, req core.Request, comment string) (bool, error) {
	if report.Mode != core.ModeUFW && report.Mode != core.ModeNFTables {
		return false, fmt.Errorf("%s", report.Reason)
	}
	switch report.Mode {
	case core.ModeUFW:
		args := ufw.Command(req.Action, strings.ToLower(req.Protocol), req.Port, req.Source, comment, false)
		if _, err := run.Run("ufw", args...); err != nil {
			return false, err
		}
	case core.ModeNFTables:
		if err := ensurePortPilotTable(run); err != nil {
			return false, err
		}
		id := core.Rule{Comment: comment}.ID()
		if cmds, ok := nftables.TimeoutCommands(id, strings.ToLower(req.Protocol), req.Action, req.Port, req.DurationSec); ok && req.Source == "" && req.Interface == "" && (req.Family == "" || req.Family == "both") {
			for _, args := range cmds {
				if _, err := run.Run("nft", args...); err != nil {
					return false, err
				}
			}
			return true, nil
		}
		args := nftables.AddRuleArgs(strings.ToLower(req.Protocol), req.Port, req.Family, req.Source, req.Interface, req.Action, comment)
		if _, err := run.Run("nft", args...); err != nil {
			return false, err
		}
	}
	return false, nil
}

// DeleteOwned removes one PortPilot rule. It refuses rules without the comment prefix.
func DeleteOwned(run Runner, report core.Report, rule core.Rule) error {
	if !rule.Owned || rule.ID() == "" {
		return fmt.Errorf("only rules created by PortPilot can be deleted")
	}
	switch rule.Backend {
	case core.BackendUFW:
		if report.Mode != core.ModeUFW && report.UFWActive {
			return fmt.Errorf("UFW is not the active writer")
		}
		args := ufw.Command(rule.Action, rule.Protocol, rule.Port, rule.Source, rule.Comment, true)
		_, err := run.Run("ufw", args...)
		return err
	case core.BackendNFTables:
		if rule.Table != "portpilot" || rule.Handle == 0 {
			return fmt.Errorf("nft delete needs a portpilot handle")
		}
		_, err := run.Run("nft", nftables.DeleteHandleArgs(rule.Handle)...)
		if err != nil {
			return err
		}
		if args := nftables.DeleteSetArgs(rule.ID()); args != nil {
			_, _ = run.Run("nft", args...)
		}
		return nil
	default:
		return fmt.Errorf("unknown backend %s", rule.Backend)
	}
}

func ensurePortPilotTable(run Runner) error {
	if _, err := run.Run("nft", "list", "table", "inet", "portpilot"); err != nil {
		if _, err := run.Run("nft", "add", "table", "inet", "portpilot"); err != nil {
			return err
		}
	}
	out, _ := run.Run("nft", "list", "table", "inet", "portpilot")
	if !strings.Contains(out, "chain input") {
		_, err := run.Run("nft", "add", "chain", "inet", "portpilot", "input", "{ type filter hook input priority 0 ; policy accept ; }")
		if err != nil {
			return err
		}
	}
	return nil
}

// FindOwned returns the owned rule with this id.
func FindOwned(rules []core.Rule, id string) (core.Rule, bool) {
	for _, rule := range rules {
		if rule.ID() == id {
			return rule, true
		}
	}
	return core.Rule{}, false
}

// LegacyCount is exported for tests of the counter.
func LegacyCount(text string) int { return countLegacy(text) }

// FormatHandle is a small helper for logs.
func FormatHandle(h uint64) string { return strconv.FormatUint(h, 10) }
