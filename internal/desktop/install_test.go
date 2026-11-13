package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWritesMenuEntry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", home)
	t.Setenv("HOME", home)
	exe := filepath.Join(home, "bin", "portpilot")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(exe); err != nil {
		t.Fatal(err)
	}
	desk, err := os.ReadFile(filepath.Join(home, "applications", "portpilot.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(desk)
	if !strings.Contains(text, "Exec="+exe) || !strings.Contains(text, "StartupWMClass=portpilot") {
		t.Fatalf("desktop:\n%s", text)
	}
	icon := filepath.Join(home, "icons", "hicolor", "256x256", "apps", "portpilot.png")
	if !strings.Contains(text, "Icon="+icon) {
		t.Fatalf("icon line:\n%s", text)
	}
	st, err := os.Stat(icon)
	if err != nil || st.Size() < 32 {
		t.Fatalf("png: %v size %v", err, st)
	}
	if _, err := os.Stat(filepath.Join(home, "icons", "hicolor", "index.theme")); err == nil {
		t.Fatal("install must not replace the hicolor index")
	}
}

func TestInstallAppImageOverridesExecutable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", home)
	t.Setenv("HOME", home)
	appimagePath := filepath.Join(home, "PortPilot-x86_64.AppImage")
	t.Setenv("APPIMAGE", appimagePath)

	tempMountExe := "/tmp/.mount_portpi1234/usr/bin/portpilot"
	if err := Install(tempMountExe); err != nil {
		t.Fatal(err)
	}

	desk, err := os.ReadFile(filepath.Join(home, "applications", "portpilot.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(desk)
	if !strings.Contains(text, "Exec="+appimagePath) {
		t.Fatalf("expected Exec=%s, got:\n%s", appimagePath, text)
	}
}
