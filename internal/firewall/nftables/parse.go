// Package nftables reads nft JSON and builds commands that touch only inet portpilot.
package nftables

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xamirreza/portpilot/internal/core/firewall"
)

// Summary is enough of a ruleset to choose a backend and list rules.
type Summary struct {
	Readable bool
	Tables   []string
	Rules    []firewall.Rule
}

// ParseJSON reads `nft -j list ruleset` output.
func ParseJSON(raw string) (Summary, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Summary{}, fmt.Errorf("empty nft json")
	}
	var doc struct {
		NFTables []map[string]json.RawMessage `json:"nftables"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return Summary{}, fmt.Errorf("nft json: %w", err)
	}
	var sum Summary
	sum.Readable = true
	seen := map[string]struct{}{}
	for _, obj := range doc.NFTables {
		if rawTable, ok := obj["table"]; ok {
			var t struct {
				Family string `json:"family"`
				Name   string `json:"name"`
			}
			if json.Unmarshal(rawTable, &t) == nil && t.Name != "" {
				key := t.Family + " " + t.Name
				if _, dup := seen[key]; !dup {
					seen[key] = struct{}{}
					sum.Tables = append(sum.Tables, key)
				}
			}
		}
		if rawRule, ok := obj["rule"]; ok {
			if rule, ok := parseRule(rawRule); ok {
				sum.Rules = append(sum.Rules, rule)
			}
		}
	}
	return sum, nil
}

func parseRule(raw json.RawMessage) (firewall.Rule, bool) {
	var r struct {
		Family  string          `json:"family"`
		Table   string          `json:"table"`
		Chain   string          `json:"chain"`
		Handle  uint64          `json:"handle"`
		Comment string          `json:"comment"`
		Expr    json.RawMessage `json:"expr"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Table == "" {
		return firewall.Rule{}, false
	}
	rule := firewall.Rule{
		Backend:  firewall.BackendNFTables,
		Family:   r.Family,
		Table:    r.Table,
		Chain:    r.Chain,
		Handle:   r.Handle,
		Comment:  r.Comment,
		Owned:    r.Table == "portpilot" && firewall.OwnedComment(r.Comment),
		Protocol: protoFromExpr(r.Expr),
		Port:     portFromExpr(r.Expr),
		Action:   actionFromExpr(r.Expr),
	}
	return rule, true
}

func protoFromExpr(raw json.RawMessage) string {
	text := strings.ToLower(string(raw))
	switch {
	case strings.Contains(text, `"protocol":"udp"`), strings.Contains(text, `"protocol": "udp"`):
		return "udp"
	case strings.Contains(text, `"protocol":"tcp"`), strings.Contains(text, `"protocol": "tcp"`):
		return "tcp"
	default:
		return ""
	}
}

func portFromExpr(raw json.RawMessage) int {
	var exprs []map[string]json.RawMessage
	if json.Unmarshal(raw, &exprs) != nil {
		return 0
	}
	for _, expr := range exprs {
		match, ok := expr["match"]
		if !ok {
			continue
		}
		var body struct {
			Right json.RawMessage `json:"right"`
			Left  struct {
				Payload struct {
					Field string `json:"field"`
				} `json:"payload"`
			} `json:"left"`
		}
		if json.Unmarshal(match, &body) != nil || body.Left.Payload.Field != "dport" {
			continue
		}
		var n int
		if json.Unmarshal(body.Right, &n) == nil {
			return n
		}
	}
	return 0
}

func actionFromExpr(raw json.RawMessage) string {
	text := strings.ToLower(string(raw))
	switch {
	case strings.Contains(text, `"drop"`):
		return firewall.ActionDeny
	case strings.Contains(text, `"reject"`):
		return firewall.ActionReject
	case strings.Contains(text, `"accept"`):
		return firewall.ActionAllow
	default:
		return ""
	}
}

// OtherTables returns tables that are not inet portpilot.
func OtherTables(sum Summary) []string {
	var out []string
	for _, t := range sum.Tables {
		if t != "inet portpilot" {
			out = append(out, t)
		}
	}
	return out
}

// AddRuleArgs builds an nft add-rule invocation inside inet portpilot only.
func AddRuleArgs(proto string, port int, family, source, iface, action, comment string) []string {
	stmt := map[string]string{firewall.ActionAllow: "accept", firewall.ActionDeny: "drop", firewall.ActionReject: "reject"}[action]
	args := []string{"add", "rule", "inet", "portpilot", "input"}
	switch family {
	case "ipv4":
		args = append(args, "meta", "nfproto", "ipv4")
	case "ipv6":
		args = append(args, "meta", "nfproto", "ipv6")
	}
	if iface != "" {
		args = append(args, "iifname", iface)
	}
	if source != "" {
		if strings.Contains(source, ":") {
			args = append(args, "ip6", "saddr", source)
		} else {
			args = append(args, "ip", "saddr", source)
		}
	}
	args = append(args, proto, "dport", strconv.Itoa(port), stmt, "comment", nftComment(comment))
	return args
}

// nftComment quotes a comment for the nft parser. The colon in portpilot:id
// is otherwise read as syntax. The quotes are part of the argument because
// the helper does not use a shell.
func nftComment(comment string) string {
	return `"` + comment + `"`
}

// DeleteHandleArgs deletes one handle in the PortPilot table.
func DeleteHandleArgs(handle uint64) []string {
	return []string{"delete", "rule", "inet", "portpilot", "input", "handle", strconv.FormatUint(handle, 10)}
}
