package distro

import (
	"os"
	"strings"
)

// Info is the host identity doctor and the UI can show.
type Info struct {
	PrettyName string
	ID         string
	Init       string
	Hostname   string
}

// Detect reads os-release, hostname, and PID 1.
func Detect() Info {
	info := Info{Init: initSystem()}
	if name, err := os.Hostname(); err == nil {
		info.Hostname = name
	}
	raw, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return info
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, `"`)
		switch key {
		case "PRETTY_NAME":
			info.PrettyName = val
		case "ID":
			info.ID = val
		}
	}
	return info
}

func initSystem() string {
	comm, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		return "unknown"
	}
	name := strings.TrimSpace(string(comm))
	switch name {
	case "systemd", "init", "openrc-init":
		return name
	default:
		if name == "" {
			return "unknown"
		}
		return name
	}
}
