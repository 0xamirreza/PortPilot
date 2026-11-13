// Package ss compares a diagnostic listing with the socket scan. It does not change the host.
package ss

import (
	"os/exec"
	"strings"
)

// Note reports a short mismatch between ss and the scanned bind list.
// binds are "tcp 127.0.0.1:80". An empty result means ss was unavailable or agreed.
func Note(binds []string) string {
	out, err := exec.Command("ss", "-H", "-ltn").Output()
	udp, err2 := exec.Command("ss", "-H", "-lun").Output()
	if err != nil && err2 != nil {
		return ""
	}
	seen := map[string]struct{}{}
	add(seen, "tcp", string(out))
	add(seen, "udp", string(udp))
	if len(seen) == 0 {
		return ""
	}
	missing := 0
	var sample []string
	for _, bind := range binds {
		if _, ok := seen[bind]; !ok {
			missing++
			if len(sample) < 3 {
				sample = append(sample, bind)
			}
		}
	}
	if missing == 0 {
		return ""
	}
	return "ss does not list every TCP listen from the scan (" + strings.Join(sample, ", ") + "). The scan stays on netlink or procfs; ss is only a cross-check."
}

func add(seen map[string]struct{}, proto, text string) {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		seen[proto+" "+normalizeSS(fields[3])] = struct{}{}
	}
}

func normalizeSS(addr string) string {
	if i := strings.Index(addr, "%"); i >= 0 {
		addr = addr[:i] + addr[strings.LastIndex(addr, ":"):]
	}
	if host, port, ok := strings.Cut(strings.TrimPrefix(addr, "["), "]:"); ok && strings.HasPrefix(addr, "[") {
		return host + ":" + port
	}
	if strings.HasPrefix(addr, "*:") {
		return "0.0.0.0" + addr[1:]
	}
	return addr
}
