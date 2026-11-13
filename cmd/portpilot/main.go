package main

import (
	"fmt"
	"os"

	"github.com/0xamirreza/portpilot/internal/app"
	"github.com/0xamirreza/portpilot/internal/desktop"
	"github.com/0xamirreza/portpilot/internal/gui"
	"github.com/0xamirreza/portpilot/internal/system/privilege"

	gioapp "gioui.org/app"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "doctor":
			if err := app.Doctor(os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, "portpilot:", err)
				return 1
			}
			return 0
		case "__privileged":
			id := ""
			if len(args) >= 3 && args[1] == "expire" {
				id = args[2]
			}
			if err := privilege.Serve(os.Stdin, os.Stdout, id); err != nil {
				fmt.Fprintln(os.Stderr, "portpilot:", err)
				return 1
			}
			return 0
		case "install-launcher":
			exe, err := os.Executable()
			if err != nil {
				fmt.Fprintln(os.Stderr, "portpilot:", err)
				return 1
			}
			if err := desktop.Install(exe); err != nil {
				fmt.Fprintln(os.Stderr, "portpilot:", err)
				return 1
			}
			fmt.Println("installed the PortPilot menu entry")
			return 0
		case "install-policy":
			if os.Geteuid() != 0 {
				fmt.Fprintln(os.Stderr, "portpilot: install-policy must run as root, through pkexec")
				return 2
			}
			exe, err := os.Executable()
			if err != nil {
				fmt.Fprintln(os.Stderr, "portpilot:", err)
				return 1
			}
			if err := privilege.InstallPolicy(exe); err != nil {
				fmt.Fprintln(os.Stderr, "portpilot:", err)
				return 1
			}
			fmt.Println("installed /usr/share/polkit-1/actions/dev.portpilot.policy")
			return 0
		case "help", "-h", "--help":
			usage()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "portpilot: unknown command %q\n", args[0])
			usage()
			return 2
		}
	}
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "portpilot: the window does not run as root")
		return 2
	}
	if exe, err := os.Executable(); err == nil {
		if err := desktop.Install(exe); err != nil {
			fmt.Fprintln(os.Stderr, "portpilot:", err)
		}
	}
	application, err := app.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "portpilot:", err)
		return 1
	}
	defer application.Close()
	go func() {
		err := gui.Run(application)
		application.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, "portpilot:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	gioapp.Main()
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n  portpilot\n  portpilot doctor\n  portpilot install-launcher\n  portpilot install-policy\n  portpilot __privileged\n")
}
