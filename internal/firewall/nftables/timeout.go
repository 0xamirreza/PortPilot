package nftables

import (
	"fmt"
	"strconv"
	"strings"
)

// TimeoutCommands builds an nft set whose element expires, inside inet portpilot only.
// Source and interface rules are not expressed as a port set, so the caller keeps the plain rule.
func TimeoutCommands(id, proto, action string, port, seconds int) ([][]string, bool) {
	if !setID(id) || seconds < 60 || (proto != "tcp" && proto != "udp") {
		return nil, false
	}
	stmt := map[string]string{"allow": "accept", "deny": "drop", "reject": "reject"}[action]
	if stmt == "" {
		return nil, false
	}
	name := "pp" + id
	spec := fmt.Sprintf("{ type inet_service ; flags timeout ; }")
	element := fmt.Sprintf("{ %d timeout %s }", port, nftDuration(seconds))
	return [][]string{
		{"add", "set", "inet", "portpilot", name, spec},
		{"add", "element", "inet", "portpilot", name, element},
		{"add", "rule", "inet", "portpilot", "input", proto, "dport", "@" + name, stmt, "comment", nftComment("portpilot:" + id)},
	}, true
}

// DeleteSetArgs removes the timeout set for one owned id.
func DeleteSetArgs(id string) []string {
	if !setID(id) {
		return nil
	}
	return []string{"delete", "set", "inet", "portpilot", "pp" + id}
}

func setID(id string) bool {
	if len(id) < 8 || len(id) > 32 {
		return false
	}
	for _, c := range id {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

func nftDuration(seconds int) string {
	if seconds%3600 == 0 {
		return strconv.Itoa(seconds/3600) + "h"
	}
	if seconds%60 == 0 {
		return strconv.Itoa(seconds/60) + "m"
	}
	return strconv.Itoa(seconds) + "s"
}
