package privilege

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func serviceCommand(initName, op, name string) (string, []string, error) {
	switch op {
	case "start", "stop", "restart", "enable", "disable":
	default:
		return "", nil, fmt.Errorf("unsupported service operation")
	}
	switch initName {
	case "systemd":
		unit, err := cleanUnit(name)
		if err != nil {
			return "", nil, err
		}
		return "systemctl", []string{op, unit}, nil
	case "openrc-init":
		base, err := plainName(name)
		if err != nil {
			return "", nil, err
		}
		if op == "enable" {
			return "rc-update", []string{"add", base, "default"}, nil
		}
		if op == "disable" {
			return "rc-update", []string{"del", base, "default"}, nil
		}
		return "rc-service", []string{base, op}, nil
	case "init":
		base, err := plainName(name)
		if err != nil {
			return "", nil, err
		}
		if op == "enable" || op == "disable" {
			return "", nil, fmt.Errorf("SysV on this host has no single enable command")
		}
		path := filepath.Join("/etc/init.d", base)
		st, err := os.Stat(path)
		if err != nil || st.IsDir() {
			return "", nil, fmt.Errorf("init script %s was not found", base)
		}
		return path, []string{op}, nil
	default:
		return "", nil, fmt.Errorf("service actions are not available for init %s", initName)
	}
}

func plainName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".service")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", fmt.Errorf("service name is not accepted")
	}
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '@', c == '.', c == '_', c == '-', c == ':':
		default:
			return "", fmt.Errorf("service name is not accepted")
		}
	}
	return name, nil
}
