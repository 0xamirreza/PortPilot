// Package services lists init units. It does not start or stop them.
package services

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Unit is one service the window can show.
type Unit struct {
	Name        string
	Load        string
	Active      string
	Sub         string
	Description string
	Init        string
}

var splitSpace = regexp.MustCompile(`\s{2,}`)

// ParseSystemd reads `systemctl list-units --type=service --plain` output.
func ParseSystemd(text string) []Unit {
	var out []Unit
	knownLoads := map[string]bool{
		"loaded":      true,
		"not-found":   true,
		"bad-setting": true,
		"error":       true,
		"masked":      true,
	}

	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "UNIT ") || strings.HasPrefix(line, "Legend:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Strip leading status bullet if present (e.g. ●, ○, *)
		if fields[0] == "●" || fields[0] == "○" || fields[0] == "*" {
			fields = fields[1:]
			if len(fields) < 3 {
				continue
			}
		}

		name := fields[0]
		if !strings.HasSuffix(name, ".service") {
			continue
		}

		load := "loaded"
		active := ""
		sub := ""
		desc := ""

		if len(fields) >= 4 && knownLoads[strings.ToLower(fields[1])] {
			load = fields[1]
			active = fields[2]
			sub = fields[3]
			if len(fields) > 4 {
				desc = strings.Join(fields[4:], " ")
			}
		} else {
			// Output without LOAD column: UNIT ACTIVE SUB [DESCRIPTION...]
			active = fields[1]
			if len(fields) > 2 {
				sub = fields[2]
			}
			if len(fields) > 3 {
				desc = strings.Join(fields[3:], " ")
			}
		}

		out = append(out, Unit{
			Name:        name,
			Load:        load,
			Active:      active,
			Sub:         sub,
			Description: desc,
			Init:        "systemd",
		})
	}
	return out
}

// List reads the init system that is actually running.
func List(init string) ([]Unit, error) {
	switch init {
	case "systemd":
		cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager", "--no-legend", "--plain")
		cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C"}
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return nil, fmt.Errorf("systemctl: %w", err)
		}
		return ParseSystemd(string(out)), nil
	case "openrc-init":
		return listDir("/etc/init.d", "openrc"), nil
	case "init":
		if _, err := os.Stat("/usr/bin/systemctl"); err == nil {
			return List("systemd")
		}
		return listDir("/etc/init.d", "sysv"), nil
	default:
		return nil, fmt.Errorf("service listing is not available for init %s", init)
	}
}

func listDir(dir, init string) []Unit {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Unit
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.Contains(name, " ") {
			continue
		}
		out = append(out, Unit{Name: name, Init: init})
	}
	return out
}
