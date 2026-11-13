package app

import (
	"sync"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/firewall"
	"github.com/0xamirreza/portpilot/internal/core/network"
	"github.com/0xamirreza/portpilot/internal/core/ports"
	"github.com/0xamirreza/portpilot/internal/core/processes"
	"github.com/0xamirreza/portpilot/internal/discovery/containers"
	"github.com/0xamirreza/portpilot/internal/discovery/services"
	"github.com/0xamirreza/portpilot/internal/discovery/sockets"
	"github.com/0xamirreza/portpilot/internal/storage/repositories"
	"github.com/0xamirreza/portpilot/internal/system/distro"
)

// View is a copy of the latest scan plus the preferences the window draws.
type View struct {
	Sockets          []ports.Socket
	Processes        []processes.Process
	Interfaces       []network.Interface
	History          []repositories.Entry
	Source           string
	Warnings         []string
	Banner           string
	ScannedAt        time.Time
	NextScan         time.Time
	Duration         time.Duration
	Theme            string
	Refresh          time.Duration
	Host             distro.Info
	ConfigPath       string
	DataDir          string
	Scanning         bool
	Firewall         firewall.Report
	Services         []services.Unit
	Containers       []containers.Item
	Owned            []repositories.OwnedRule
	Busy             string
	Authorized       bool
	LogPath          string
	LogRetentionDays int
}

// Store publishes the latest view to the GUI.
type Store struct {
	mu     sync.RWMutex
	view   View
	rescan chan struct{}
}

// NewStore starts from the loaded preferences.
func NewStore(theme string, refresh time.Duration, host distro.Info, configPath, dataDir, banner, logPath string, logRetentionDays int) *Store {
	return &Store{
		rescan: make(chan struct{}, 1),
		view: View{
			Theme:            theme,
			Refresh:          refresh,
			Host:             host,
			ConfigPath:       configPath,
			DataDir:          dataDir,
			Banner:           banner,
			LogPath:          logPath,
			LogRetentionDays: logRetentionDays,
		},
	}
}

// Copy returns the view the window may keep for one frame.
func (s *Store) Copy() View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.view
	out.Sockets = append([]ports.Socket(nil), s.view.Sockets...)
	out.Processes = append([]processes.Process(nil), s.view.Processes...)
	out.Interfaces = append([]network.Interface(nil), s.view.Interfaces...)
	out.History = append([]repositories.Entry(nil), s.view.History...)
	out.Warnings = append([]string(nil), s.view.Warnings...)
	out.Services = append([]services.Unit(nil), s.view.Services...)
	out.Containers = append([]containers.Item(nil), s.view.Containers...)
	out.Owned = append([]repositories.OwnedRule(nil), s.view.Owned...)
	out.Firewall.Rules = append([]firewall.Rule(nil), s.view.Firewall.Rules...)
	out.Firewall.Schedules = append([]firewall.Schedule(nil), s.view.Firewall.Schedules...)
	out.Firewall.OtherNFTTables = append([]string(nil), s.view.Firewall.OtherNFTTables...)
	return out
}

// UpdateScan replaces the latest discovery result.
func (s *Store) UpdateScan(snap sockets.Snapshot, history []repositories.Entry, scanErr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.view.Sockets = snap.Sockets
	s.view.Processes = snap.Processes
	s.view.Interfaces = snap.Interfaces
	s.view.Source = snap.Source
	s.view.Warnings = snap.Warnings
	s.view.ScannedAt = snap.ScannedAt
	s.view.NextScan = time.Now().Add(s.view.Refresh)
	s.view.Duration = snap.Duration
	s.view.History = history
	s.view.Scanning = false
	if scanErr != "" {
		s.view.Banner = scanErr
	}
}

// SetPrefs updates theme and refresh shown by the window.
func (s *Store) SetPrefs(theme string, refresh time.Duration, banner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.view.Theme = theme
	s.view.Refresh = refresh
	s.view.NextScan = time.Now().Add(refresh)
	s.view.Banner = banner
}

// SetLogRetention updates the log retention days shown by the window.
func (s *Store) SetLogRetention(days int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.view.LogRetentionDays = days
}

// MarkScanning notes that a read is in progress.
func (s *Store) MarkScanning() {
	s.mu.Lock()
	s.view.Scanning = true
	s.mu.Unlock()
}

// RequestRescan asks the loop to read the host again.
func (s *Store) RequestRescan() {
	select {
	case s.rescan <- struct{}{}:
	default:
	}
}

// Rescan returns the signal channel.
func (s *Store) Rescan() <-chan struct{} {
	return s.rescan
}

// SetFirewall replaces the firewall report without touching the socket scan.
func (s *Store) SetFirewall(rep firewall.Report) {
	s.mu.Lock()
	s.view.Firewall = rep
	if rep.Authorized {
		s.view.Authorized = true
		s.view.Busy = ""
	}
	s.mu.Unlock()
}

// SetAuthorized updates the root authorization status and firewall report.
func (s *Store) SetAuthorized(auth bool, rep firewall.Report) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.view.Authorized = auth
	s.view.Firewall = rep
	s.view.Firewall.Authorized = auth
	s.view.Busy = ""
}

// SetServices replaces the service list.
func (s *Store) SetServices(units []services.Unit) {
	s.mu.Lock()
	s.view.Services = units
	s.mu.Unlock()
}

// SetContainers replaces the container list.
func (s *Store) SetContainers(items []containers.Item) {
	s.mu.Lock()
	s.view.Containers = items
	s.mu.Unlock()
}

// SetOwned replaces remembered PortPilot rules.
func (s *Store) SetOwned(rules []repositories.OwnedRule) {
	s.mu.Lock()
	s.view.Owned = rules
	s.mu.Unlock()
}

// SetBusy shows a short status while an authorized operation is running.
func (s *Store) SetBusy(text string) {
	s.mu.Lock()
	s.view.Busy = text
	s.mu.Unlock()
}

// SetBanner replaces the red status line.
func (s *Store) SetBanner(text string) {
	s.mu.Lock()
	s.view.Banner = text
	s.mu.Unlock()
}

// Refresh returns the current interval.
func (s *Store) Refresh() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.view.Refresh
}
