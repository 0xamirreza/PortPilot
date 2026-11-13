// Package privilege is the root helper and the unprivileged client that talks to it.
package privilege

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// allowed binaries the helper may execute. Names are basenames, never paths from the GUI.
var allowed = map[string]struct{}{
	"ufw": {}, "nft": {}, "iptables-legacy": {}, "systemctl": {}, "systemd-run": {},
	"rc-service": {}, "rc-update": {},
}

// Look resolves a basename under the system directories only.
func Look(name string) (string, error) {
	if _, ok := allowed[name]; !ok {
		return "", fmt.Errorf("command %q is not allowed", name)
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", fmt.Errorf("command %q is not allowed", name)
	}
	for _, dir := range []string{"/usr/sbin", "/sbin", "/usr/bin", "/bin"} {
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s was not found", name)
}

// Exec runs one allowed binary with argument separation. It never uses a shell.
func Exec(name string, args ...string) (string, error) {
	path, err := Look(name)
	if err != nil {
		return "", err
	}
	cmd := exec.Command(path, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C"}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", err
		}
		return text, fmt.Errorf("%s: %s", name, text)
	}
	return text, nil
}

// execFile runs one init script by its absolute path. It is not a shell.
func execFile(path string, args ...string) (string, error) {
	if filepath.Dir(path) != "/etc/init.d" {
		return "", fmt.Errorf("command is not allowed")
	}
	cmd := exec.Command(path, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C"}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", err
		}
		return text, fmt.Errorf("%s: %s", filepath.Base(path), text)
	}
	return text, nil
}
