// Package ufw reads and builds UFW commands. It does not invoke a shell.
package ufw

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/0xamirreza/portpilot/internal/core/firewall"
)

var ruleLine = regexp.MustCompile(`^(?:\[\s*\d+\])?\s*(\d+)(?:/([A-Za-z0-9]+))?\s+(?:\(v6\)\s+)?(ALLOW|DENY|REJECT)\s+(\S+)\s+(\S+)(?:\s+\(v6\))?(?:\s+#\s*(.*))?$`)

// Status is the result of `ufw status verbose`.
type Status struct {
	Active          bool
	Known           bool
	DefaultInbound  string
	DefaultOutbound string
	Rules           []firewall.Rule
}

// ParseStatus reads `ufw status verbose` or `ufw status numbered` text.
func ParseStatus(text string) Status {
	var st Status
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		low := strings.ToLower(line)
		switch {
		case strings.HasPrefix(low, "status:"):
			st.Known = true
			st.Active = strings.Contains(low, "active") && !strings.Contains(low, "inactive")
		case strings.HasPrefix(low, "default:"):
			st.DefaultInbound, st.DefaultOutbound = parseDefault(low)
		default:
			if rule, ok := parseRule(line); ok {
				st.Rules = append(st.Rules, rule)
			}
		}
	}
	return st
}

func parseDefault(line string) (in, out string) {
	// default: deny (incoming), allow (outgoing), disabled (routed)
	if i := strings.Index(line, "("); i > 0 {
		in = strings.TrimSpace(strings.TrimPrefix(line[:i], "default:"))
	}
	if strings.Contains(line, "incoming") {
		in = wordBefore(line, "(incoming)")
	}
	if strings.Contains(line, "outgoing") {
		out = wordBefore(line, "(outgoing)")
	}
	return in, out
}

func wordBefore(line, marker string) string {
	i := strings.Index(line, marker)
	if i < 0 {
		return ""
	}
	prev := strings.TrimSpace(line[:i])
	if c := strings.LastIndex(prev, " "); c >= 0 {
		prev = prev[c+1:]
	}
	return strings.Trim(prev, ",")
}

func parseRule(line string) (firewall.Rule, bool) {
	m := ruleLine.FindStringSubmatch(line)
	if m == nil {
		return firewall.Rule{}, false
	}
	port, err := strconv.Atoi(m[1])
	if err != nil {
		return firewall.Rule{}, false
	}
	proto := strings.ToLower(m[2])
	if proto == "" {
		proto = "tcp"
	}
	family := "ipv4+ipv6"
	if strings.Contains(line, "(v6)") {
		family = "ipv6"
	}
	comment := strings.TrimSpace(m[6])
	return firewall.Rule{
		Backend:  firewall.BackendUFW,
		Action:   strings.ToLower(m[3]),
		Protocol: proto,
		Port:     port,
		Family:   family,
		Source:   fromField(m[5]),
		Comment:  comment,
		Owned:    firewall.OwnedComment(comment),
		Raw:      line,
	}, true
}

func fromField(from string) string {
	low := strings.ToLower(from)
	if low == "anywhere" || strings.HasPrefix(low, "anywhere") {
		return ""
	}
	return from
}

// Command builds a fixed-argument ufw invocation for one change.
func Command(action, proto string, port int, source, comment string, delete bool) []string {
	args := []string{}
	if delete {
		args = append(args, "delete")
	}
	args = append(args, action, "proto", proto)
	if source == "" {
		args = append(args, "from", "any")
	} else {
		args = append(args, "from", source)
	}
	args = append(args, "to", "any", "port", strconv.Itoa(port), "comment", comment)
	return args
}
