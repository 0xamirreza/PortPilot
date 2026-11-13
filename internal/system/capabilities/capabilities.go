// Package capabilities reports which host tools exist. It does not run them.
package capabilities

import (
	"os"
)

// Report is a presence check. Presence is not permission to change the system.
type Report struct {
	Display      string
	Session      string
	ProcNetTCP   bool
	UFW          bool
	Nft          bool
	Systemctl    bool
	Polkit       bool
	Sudo         bool
	DockerSocket bool
	DockerCLI    bool
	LXC          bool
	Virsh        bool
}

// Probe stats known paths and environment variables.
func Probe() Report {
	return Report{
		Display:      os.Getenv("DISPLAY"),
		Session:      os.Getenv("XDG_SESSION_TYPE"),
		ProcNetTCP:   exists("/proc/net/tcp"),
		UFW:          exists("/usr/sbin/ufw"),
		Nft:          exists("/usr/sbin/nft") || exists("/sbin/nft"),
		Systemctl:    exists("/usr/bin/systemctl") || exists("/bin/systemctl"),
		Polkit:       exists("/usr/bin/pkexec"),
		Sudo:         exists("/usr/bin/sudo"),
		DockerSocket: exists("/var/run/docker.sock"),
		DockerCLI:    exists("/usr/bin/docker"),
		LXC:          exists("/usr/bin/lxc") || exists("/usr/bin/lxc-ls"),
		Virsh:        exists("/usr/bin/virsh"),
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
