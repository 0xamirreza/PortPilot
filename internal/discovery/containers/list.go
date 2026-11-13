// Package containers reads Docker, LXC, and libvirt. It never changes them.
package containers

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"strings"
)

// Item is one container, VM, or virtual network.
type Item struct {
	Kind      string
	ID        string
	Name      string
	Status    string
	Image     string
	Ports     string
	Network   string
	Host      string
	Published []Publish
}

// Publish is one published container port, separate from a host socket.
type Publish struct {
	Host      string
	Container string
	Proto     string
}

// ParsePublish splits a docker Ports field into host and container endpoints.
func ParsePublish(text string) []Publish {
	var out []Publish
	for _, part := range strings.Split(text, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		left, right, ok := strings.Cut(part, "->")
		if !ok {
			continue
		}
		cont, proto, ok := strings.Cut(right, "/")
		if !ok {
			proto = "tcp"
			cont = right
		}
		out = append(out, Publish{Host: strings.TrimSpace(left), Container: strings.TrimSpace(cont), Proto: strings.ToLower(strings.TrimSpace(proto))})
	}
	return out
}

// ParseDocker reads one JSON object per line from `docker ps`.
func ParseDocker(text string) []Item {
	var out []Item
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row struct {
			ID     string
			Names  string
			Status string
			Image  string
			Ports  string
		}
		if json.Unmarshal([]byte(line), &row) != nil || row.ID == "" {
			continue
		}
		id := row.ID
		if len(id) > 12 {
			id = id[:12]
		}
		out = append(out, Item{Kind: "docker", ID: id, Name: row.Names, Status: row.Status, Image: row.Image, Ports: row.Ports, Published: ParsePublish(row.Ports)})
	}
	return out
}

// ParseVirsh reads `virsh -r list --all` or `virsh -r net-list --all`.
func ParseVirsh(kind, text string) []Item {
	var out []Item
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Id") || strings.HasPrefix(line, "Name") {
			continue
		}
		fields := strings.Fields(line)
		if kind == "libvirt-net" && len(fields) >= 2 {
			out = append(out, Item{Kind: kind, Name: fields[0], Status: strings.Join(fields[1:], " ")})
			continue
		}
		if len(fields) >= 3 && fields[0] != "Id" {
			out = append(out, Item{Kind: kind, ID: fields[0], Name: fields[1], Status: strings.Join(fields[2:], " ")})
		}
	}
	return out
}

// List asks the tools that exist. A missing tool is skipped.
func List() []Item {
	var out []Item
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.Command("docker", "ps", "-a", "--format", "{{json .}}")
		cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C"}
		if raw, err := cmd.Output(); err == nil {
			out = append(out, ParseDocker(string(raw))...)
		}
	}
	if path, err := exec.LookPath("lxc-ls"); err == nil {
		cmd := exec.Command(path, "-1")
		if raw, err := cmd.Output(); err == nil {
			for _, name := range strings.Fields(string(raw)) {
				item := Item{Kind: "lxc", Name: name}
				if safeName(name) {
					if info, err := exec.Command("lxc-info", "-n", name).Output(); err == nil {
						item.Status, item.Network = parseLXCInfo(string(info))
					}
				}
				out = append(out, item)
			}
		}
	}
	if path, err := exec.LookPath("virsh"); err == nil {
		if raw, err := exec.Command(path, "-r", "list", "--all").Output(); err == nil {
			out = append(out, ParseVirsh("libvirt", string(raw))...)
		}
		if raw, err := exec.Command(path, "-r", "net-list", "--all").Output(); err == nil {
			out = append(out, ParseVirsh("libvirt-net", string(raw))...)
		}
	}
	return out
}

func safeName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

func parseLXCInfo(text string) (state, ip string) {
	for _, line := range strings.Split(text, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		switch key {
		case "state":
			state = val
		case "ip":
			if ip == "" {
				ip = val
			}
		}
	}
	return state, ip
}

// HostSocket is the part of a scanned socket used to relate a container.
type HostSocket struct {
	Container string
	Process   string
	Proto     string
	Addr      string
	Port      int
}

// Attach notes host sockets whose cgroup matches a container id or name.
func Attach(items []Item, socks []HostSocket) []Item {
	for i := range items {
		var hits []string
		for _, s := range socks {
			if s.Container == "" {
				continue
			}
			idHit := items[i].ID != "" && strings.Contains(s.Container, items[i].ID)
			nameHit := items[i].Name != "" && strings.Contains(s.Container, items[i].Name)
			if idHit || nameHit {
				hits = append(hits, s.Proto+" "+s.Addr)
			}
		}
		if len(hits) > 0 {
			items[i].Host = strings.Join(hits, ", ")
		}
	}
	return items
}
