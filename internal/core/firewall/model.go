// Package firewall is the rule model shared by UFW, nftables, and the window.
package firewall

import "time"

const (
	BackendUFW      = "ufw"
	BackendNFTables = "nftables"
	BackendNone     = "none"

	ModeUFW         = "ufw"
	ModeNFTables    = "nftables"
	ModeReadOnly    = "read-only"
	ModeUnavailable = "unavailable"

	ActionAllow  = "allow"
	ActionDeny   = "deny"
	ActionReject = "reject"

	CommentPrefix = "portpilot:"
)

// Rule is one firewall rule the window can show. It is not kernel state by itself.
type Rule struct {
	Backend   string `json:"backend"`
	Action    string `json:"action"`
	Protocol  string `json:"protocol"`
	Port      int    `json:"port"`
	Family    string `json:"family"`
	Source    string `json:"source,omitempty"`
	Interface string `json:"interface,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Owned     bool   `json:"owned"`
	Handle    uint64 `json:"handle,omitempty"`
	Table     string `json:"table,omitempty"`
	Chain     string `json:"chain,omitempty"`
	Raw       string `json:"raw,omitempty"`
}

// ID is the PortPilot comment token, when this process created the rule.
func (r Rule) ID() string {
	if len(r.Comment) <= len(CommentPrefix) || r.Comment[:len(CommentPrefix)] != CommentPrefix {
		return ""
	}
	id := r.Comment[len(CommentPrefix):]
	for _, c := range id {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
		default:
			return ""
		}
	}
	if len(id) < 8 || len(id) > 32 {
		return ""
	}
	return id
}

// Report is a point-in-time firewall reading.
type Report struct {
	UFWInstalled     bool       `json:"ufw_installed"`
	NftInstalled     bool       `json:"nft_installed"`
	LegacyInstalled  bool       `json:"legacy_installed"`
	UFWEnabledOnBoot bool       `json:"ufw_enabled_on_boot"`
	UFWActive        bool       `json:"ufw_active"`
	UFWKnown         bool       `json:"ufw_known"`
	DefaultInbound   string     `json:"default_inbound,omitempty"`
	DefaultOutbound  string     `json:"default_outbound,omitempty"`
	NftReadable      bool       `json:"nft_readable"`
	OtherNFTTables   []string   `json:"other_nft_tables,omitempty"`
	LegacyRules      int        `json:"legacy_rules"`
	Mode             string     `json:"mode"`
	Reason           string     `json:"reason"`
	Rules            []Rule     `json:"rules,omitempty"`
	Schedules        []Schedule `json:"schedules,omitempty"`
	ReadAt           time.Time  `json:"read_at"`
	Authorized       bool       `json:"authorized"`
	Error            string     `json:"error,omitempty"`
}

// Schedule is a temporary rule tracked by the privileged helper.
type Schedule struct {
	ID       string    `json:"id"`
	Backend  string    `json:"backend"`
	Protocol string    `json:"protocol"`
	Port     int       `json:"port"`
	Action   string    `json:"action"`
	Expires  time.Time `json:"expires"`
}

// Request is a validated change. The helper rejects anything outside this shape.
type Request struct {
	Action      string `json:"action"`
	Protocol    string `json:"protocol"`
	Port        int    `json:"port"`
	Family      string `json:"family"`
	Source      string `json:"source,omitempty"`
	Interface   string `json:"interface,omitempty"`
	DurationSec int    `json:"duration_sec,omitempty"`
	ConfirmSSH  bool   `json:"confirm_ssh"`
	RuleID      string `json:"rule_id,omitempty"`
}
