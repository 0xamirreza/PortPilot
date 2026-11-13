package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingUsesDarkDefault(t *testing.T) {
	dir := t.TempDir()
	cfg, existed, err := Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Fatal("missing file reported as present")
	}
	if cfg.Theme != ThemeDark || cfg.RefreshSeconds != DefaultRefresh || cfg.LogRetentionDays != DefaultLogRetentionDays {
		t.Fatalf("defaults = %+v", cfg)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	in := File{Theme: ThemeLight, RefreshSeconds: 5, LogRetentionDays: 14}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	out, existed, err := Load(path)
	if err != nil || !existed {
		t.Fatalf("load existed=%v err=%v", existed, err)
	}
	if out != in {
		t.Fatalf("got %+v", out)
	}
}

func TestRejectsUnknownTheme(t *testing.T) {
	err := (File{Theme: "sepia", RefreshSeconds: 3}).Validate()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolvePathsHonorsXDG(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))
	p, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	if p.Config != filepath.Join(base, "cfg", "portpilot", "config.json") {
		t.Fatal(p.Config)
	}
	if p.Database != filepath.Join(base, "data", "portpilot", "portpilot.db") {
		t.Fatal(p.Database)
	}
}
