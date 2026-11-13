// Package desktop installs the user launcher entry and icon.
package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0xamirreza/portpilot/assets/brand"
)

// Install writes the menu entry and hicolor icons for exe.
// The icon path in the desktop file is absolute so the menu does not depend on an icon-theme index.
func Install(exe string) error {
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		exe = appimage
	}
	exe, err := filepath.Abs(exe)
	if err != nil {
		return err
	}
	if strings.ContainsAny(exe, "\n\r") {
		return fmt.Errorf("executable path cannot contain a newline")
	}
	data, err := dataHome()
	if err != nil {
		return err
	}
	iconRoot := filepath.Join(data, "icons", "hicolor")
	if err := brand.WriteIcons(iconRoot); err != nil {
		return err
	}
	icon := filepath.Join(iconRoot, "256x256", "apps", "portpilot.png")
	appDir := filepath.Join(data, "applications")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return err
	}
	body := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=PortPilot\n" +
		"Comment=Linux ports, firewall, and services\n" +
		"Exec=" + quoteDesktop(exe) + "\n" +
		"Icon=" + quoteDesktop(icon) + "\n" +
		"Terminal=false\n" +
		"Categories=System;Monitor;Security;\n" +
		"StartupWMClass=portpilot\n"
	path := filepath.Join(appDir, "portpilot.desktop")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	if tool, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command(tool, appDir).Run()
	}
	return nil
}

func dataHome() (string, error) {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

func quoteDesktop(s string) string {
	if !strings.ContainsAny(s, " \t\"\\") {
		return s
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
