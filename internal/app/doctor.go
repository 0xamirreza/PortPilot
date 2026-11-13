package app

import (
	"debug/elf"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/0xamirreza/portpilot/internal/config"
	"github.com/0xamirreza/portpilot/internal/system/capabilities"
	"github.com/0xamirreza/portpilot/internal/system/distro"
)

// Doctor prints a read-only compatibility report and returns an error
// when the database or the socket scan cannot run.
func Doctor(w io.Writer) error {
	host := distro.Detect()
	caps := capabilities.Probe()
	fmt.Fprintf(w, "PortPilot doctor\n")
	fmt.Fprintf(w, "Host: %s (%s)\n", fallback(host.PrettyName, "unknown"), fallback(host.Hostname, "unknown"))
	fmt.Fprintf(w, "Init: %s\n", host.Init)
	fmt.Fprintf(w, "Session: %s\n", fallback(caps.Session, "unset"))
	if caps.Display == "" && caps.Session != "wayland" {
		fmt.Fprintf(w, "Display: missing. The window needs X11 or Wayland. This command does not.\n")
	} else {
		fmt.Fprintf(w, "Display: present\n")
	}
	paths, err := config.ResolvePaths()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Config: %s\n", paths.Config)
	fmt.Fprintf(w, "Data: %s\n", paths.DataDir)
	if !caps.ProcNetTCP {
		fmt.Fprintf(w, "Procfs: /proc/net/tcp is not readable\n")
		return fmt.Errorf("/proc/net/tcp is not readable")
	}
	fmt.Fprintf(w, "Procfs: /proc/net/tcp is readable\n")
	fmt.Fprintf(w, "UFW binary: %s\n", found(caps.UFW))
	fmt.Fprintf(w, "nft binary: %s\n", found(caps.Nft))
	fmt.Fprintf(w, "systemctl: %s\n", found(caps.Systemctl))
	fmt.Fprintf(w, "pkexec: %s\n", found(caps.Polkit))
	fmt.Fprintf(w, "sudo: %s\n", found(caps.Sudo))
	fmt.Fprintf(w, "Docker CLI: %s\n", found(caps.DockerCLI))
	if caps.DockerSocket {
		fmt.Fprintf(w, "Docker socket: present. It is opened only from the Containers screen.\n")
	} else {
		fmt.Fprintf(w, "Docker socket: absent\n")
	}
	fmt.Fprintf(w, "LXC tools: %s\n", found(caps.LXC))
	fmt.Fprintf(w, "virsh: %s\n", found(caps.Virsh))
	fmt.Fprintf(w, "Linking: %s. Gio needs cgo and the system graphics libraries, so this executable is not a fully static binary.\n", linkMode())
	fmt.Fprintf(w, "Firewall changes: implemented. They run only through pkexec after a confirmation. This command does not change rules.\n")

	opened, err := Open()
	if err != nil {
		fmt.Fprintf(w, "Database: %v\n", err)
		return err
	}
	defer opened.Close()
	snap, scanErr := opened.engine.Scan()
	if scanErr != nil {
		fmt.Fprintf(w, "Socket scan: %v\n", scanErr)
		return scanErr
	}
	fmt.Fprintf(w, "Socket scan: %d sockets via %s in %s\n", len(snap.Sockets), snap.Source, snap.Duration.Round(time.Millisecond))
	fmt.Fprintf(w, "Processes with sockets: %d\n", len(snap.Processes))
	fmt.Fprintf(w, "Interfaces: %d\n", len(snap.Interfaces))
	for _, warning := range snap.Warnings {
		fmt.Fprintf(w, "Warning: %s\n", warning)
	}
	fmt.Fprintf(w, "Database: %s\n", opened.paths.Database)
	return nil
}

func linkMode() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	f, err := elf.Open(path)
	if err != nil {
		return "unknown"
	}
	defer f.Close()
	for _, prog := range f.Progs {
		if prog.Type == elf.PT_INTERP {
			return "dynamic"
		}
	}
	return "static"
}

func found(v bool) string {
	if v {
		return "found"
	}
	return "not found"
}

func fallback(v, other string) string {
	if v == "" {
		return other
	}
	return v
}
