// Package config loads PortPilot settings from the XDG config home.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	AppName                 = "portpilot"
	ThemeDark               = "dark"
	ThemeLight              = "light"
	MinRefresh              = 1
	MaxRefresh              = 60
	DefaultRefresh          = 3
	DefaultLogRetentionDays = 7
	MinLogRetentionDays     = 1
	MaxLogRetentionDays     = 365
)

// Paths are the XDG locations owned by PortPilot.
type Paths struct {
	ConfigDir string
	DataDir   string
	CacheDir  string
	Config    string
	Database  string
	Log       string
}

// File is the on-disk configuration. It is not an authorization source.
type File struct {
	Theme            string `json:"theme"`
	RefreshSeconds   int    `json:"refresh_seconds"`
	LogRetentionDays int    `json:"log_retention_days,omitempty"`
}

// Default returns the settings used when no file exists.
func Default() File {
	return File{
		Theme:            ThemeDark,
		RefreshSeconds:   DefaultRefresh,
		LogRetentionDays: DefaultLogRetentionDays,
	}
}

// Refresh is the scan interval.
func (f File) Refresh() time.Duration {
	return time.Duration(f.RefreshSeconds) * time.Second
}

// ResolvePaths honors XDG_CONFIG_HOME, XDG_DATA_HOME, and XDG_CACHE_HOME.
func ResolvePaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("home directory: %w", err)
	}
	configHome := envOr("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dataHome := envOr("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	cacheHome := envOr("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	p := Paths{
		ConfigDir: filepath.Join(configHome, AppName),
		DataDir:   filepath.Join(dataHome, AppName),
		CacheDir:  filepath.Join(cacheHome, AppName),
	}
	p.Config = filepath.Join(p.ConfigDir, "config.json")
	p.Database = filepath.Join(p.DataDir, "portpilot.db")
	p.Log = filepath.Join(p.CacheDir, "portpilot.log")
	return p, nil
}

// EnsureDirs creates PortPilot directories with mode 0700.
func EnsureDirs(p Paths) error {
	for _, dir := range []string{p.ConfigDir, p.DataDir, p.CacheDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("chmod %s: %w", dir, err)
		}
	}
	return nil
}

// Load reads config.json. A missing file returns Default and ok=false.
func Load(path string) (File, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), false, nil
		}
		return File{}, false, fmt.Errorf("read config: %w", err)
	}
	cfg := Default()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return File{}, true, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return File{}, true, err
	}
	return cfg, true, nil
}

// Save writes config.json with mode 0600.
func Save(path string, cfg File) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// Validate rejects themes and intervals the application does not implement.
func (f File) Validate() error {
	switch f.Theme {
	case ThemeDark, ThemeLight:
	default:
		return fmt.Errorf("theme %q is not dark or light", f.Theme)
	}
	if f.RefreshSeconds < MinRefresh || f.RefreshSeconds > MaxRefresh {
		return fmt.Errorf("refresh_seconds must be from %d to %d", MinRefresh, MaxRefresh)
	}
	if f.LogRetentionDays != 0 && (f.LogRetentionDays < MinLogRetentionDays || f.LogRetentionDays > MaxLogRetentionDays) {
		return fmt.Errorf("log_retention_days must be from %d to %d", MinLogRetentionDays, MaxLogRetentionDays)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
