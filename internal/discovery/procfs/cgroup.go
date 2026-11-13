package procfs

import "strings"

// ParseCgroup extracts a systemd service name and a container reference.
// A scope or a user manager is not reported as the service that owns a port.
func ParseCgroup(text string) (service, container string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		_, path, ok := strings.Cut(line, ":")
		if !ok {
			path = line
		}
		// cgroup v2 lines look like "0::/system.slice/ssh.service".
		if _, rest, ok := strings.Cut(path, ":"); ok {
			path = rest
		}
		if svc := serviceFromPath(path); svc != "" {
			service = svc
		}
		if kind, id := containerFromPath(path); id != "" {
			container = kind + ":" + id
		}
	}
	return service, container
}

func serviceFromPath(path string) string {
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	if strings.HasSuffix(base, ".service") {
		return base
	}
	return ""
}

func containerFromPath(path string) (kind, id string) {
	markers := []struct {
		kind   string
		prefix string
		suffix string
	}{
		{"docker", "docker-", ".scope"},
		{"podman", "libpod-", ".scope"},
		{"lxc", "lxc.payload.", ""},
	}
	for _, part := range strings.Split(path, "/") {
		for _, m := range markers {
			if m.suffix != "" && strings.HasPrefix(part, m.prefix) && strings.HasSuffix(part, m.suffix) {
				id = strings.TrimSuffix(strings.TrimPrefix(part, m.prefix), m.suffix)
				if id != "" {
					return m.kind, shorten(id)
				}
			}
			if m.suffix == "" && strings.HasPrefix(part, m.prefix) {
				id = strings.TrimPrefix(part, m.prefix)
				if id != "" {
					return m.kind, id
				}
			}
		}
		if part == "docker" || part == "lxc" || part == "libpod" {
			continue
		}
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if (part == "docker" || part == "lxc" || part == "libpod") && i+1 < len(parts) && parts[i+1] != "" {
			kind := part
			if kind == "libpod" {
				kind = "podman"
			}
			return kind, shorten(parts[i+1])
		}
	}
	return "", ""
}

func shorten(id string) string {
	id = strings.TrimSuffix(id, ".scope")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
