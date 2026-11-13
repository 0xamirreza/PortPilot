package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/0xamirreza/portpilot/internal/config"
	"github.com/0xamirreza/portpilot/internal/core/events"
	"github.com/0xamirreza/portpilot/internal/discovery/containers"
	"github.com/0xamirreza/portpilot/internal/discovery/services"
	"github.com/0xamirreza/portpilot/internal/discovery/sockets"
	fw "github.com/0xamirreza/portpilot/internal/firewall"
	"github.com/0xamirreza/portpilot/internal/logging"
	"github.com/0xamirreza/portpilot/internal/storage/repositories"
	"github.com/0xamirreza/portpilot/internal/storage/sqlite"
	"github.com/0xamirreza/portpilot/internal/system/distro"
	"github.com/0xamirreza/portpilot/internal/system/privilege"
)

// App is the unprivileged process: config, database, and discovery.
type App struct {
	paths      config.Paths
	cfg        config.File
	log        *slog.Logger
	logFile    *os.File
	db         *sql.DB
	history    *repositories.History
	rules      *repositories.Rules
	settings   *repositories.Settings
	store      *Store
	engine     *sockets.Engine
	userName   string
	invalidate func()
	svcTick    int
	wantBoxes  bool
}

// Open prepares directories, the database, and logging. It does not scan.
func Open() (*App, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, err
	}
	if err := config.EnsureDirs(paths); err != nil {
		return nil, err
	}
	logger, logFile, logErr := logging.New(paths.Log)
	cfg, _, cfgErr := config.Load(paths.Config)
	banner := ""
	if cfgErr != nil {
		banner = cfgErr.Error()
		cfg = config.Default()
		logger.Error("config", "err", cfgErr)
	} else if logErr != nil {
		banner = "log file was not opened: " + logErr.Error()
	}
	if _, statErr := os.Stat(paths.Config); os.IsNotExist(statErr) && cfgErr == nil {
		if err := config.Save(paths.Config, cfg); err != nil {
			return nil, err
		}
	}
	ctx := context.Background()
	db, err := sqlite.Open(ctx, paths.Database)
	if err != nil {
		return nil, err
	}
	history := repositories.NewHistory(db)
	rules := repositories.NewRules(db)
	settings := repositories.NewSettings(db)
	_ = settings.Put(ctx, "theme", cfg.Theme)
	_ = settings.Put(ctx, "refresh_seconds", strconv.Itoa(cfg.RefreshSeconds))
	name := currentUser()
	target := "gui"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	if err := history.Insert(ctx, repositories.Entry{
		Action: events.ActionAppStart,
		Result: events.ResultSuccess,
		Target: target,
		Detail: "unprivileged process",
		User:   name,
	}); err != nil {
		logger.Error("audit", "err", err)
	}
	if cfg.LogRetentionDays > 0 {
		_ = logging.DefaultStore.Rotate(cfg.LogRetentionDays)
	}
	logger.Info("open", "config", paths.Config, "db", paths.Database, "theme", cfg.Theme, "retention_days", cfg.LogRetentionDays)
	return &App{
		paths:    paths,
		cfg:      cfg,
		log:      logger,
		logFile:  logFile,
		db:       db,
		history:  history,
		rules:    rules,
		settings: settings,
		store:    NewStore(cfg.Theme, cfg.Refresh(), distro.Detect(), paths.Config, paths.DataDir, banner, paths.Log, cfg.LogRetentionDays),
		engine:   &sockets.Engine{},
		userName: name,
	}, nil
}

// Store is the view model.
func (a *App) Store() *Store { return a.store }

// Close releases the database and the log file.
func (a *App) Close() {
	if a.db != nil {
		_ = a.db.Close()
	}
	if a.logFile != nil {
		_ = a.logFile.Close()
	}
}

// LiveScanInterval is the continuous real-time refresh rate for live monitoring.
const LiveScanInterval = 300 * time.Millisecond

// Loop scans until ctx is cancelled. invalidate is called after each scan.
func (a *App) Loop(ctx context.Context, invalidate func()) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.store.Rescan():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(0)
		case <-timer.C:
			a.scan(ctx)
			if invalidate != nil {
				invalidate()
			}
			timer.Reset(LiveScanInterval)
		}
	}
}

func (a *App) scan(ctx context.Context) {
	a.store.MarkScanning()
	snap, err := a.engine.Scan()
	msg := ""
	if err != nil {
		msg = "Socket scan failed: " + err.Error()
		a.log.Error("scan", "err", err)
	} else {
		a.log.Debug("discovery", "sockets", len(snap.Sockets), "processes", len(snap.Processes), "source", snap.Source, "duration", snap.Duration.String())
	}
	history, herr := a.history.Recent(ctx, 200)
	if herr != nil {
		a.log.Error("history", "err", herr)
		if msg == "" {
			msg = "Activity log could not be read: " + herr.Error()
		}
	}
	a.store.UpdateScan(snap, history, msg)
	a.refreshSide(ctx)
}

func (a *App) refreshSide(ctx context.Context) {
	view := a.store.Copy()
	if !view.Firewall.Authorized {
		a.store.SetFirewall(fw.Preview())
	}
	a.svcTick++
	if a.svcTick == 1 || a.svcTick%10 == 0 {
		if units, err := services.List(a.store.Copy().Host.Init); err == nil {
			a.store.SetServices(units)
		}
	}
	if a.wantBoxes {
		current := a.store.Copy()
		if len(current.Containers) == 0 || a.svcTick%10 == 0 {
			items := containers.List()
			var hosts []containers.HostSocket
			for _, s := range current.Sockets {
				if s.Container == "" || !s.IsBoundService() {
					continue
				}
				hosts = append(hosts, containers.HostSocket{
					Container: s.Container, Process: s.Process, Proto: s.Protocol,
					Addr: s.LocalIP.String() + ":" + strconv.Itoa(int(s.LocalPort)), Port: int(s.LocalPort),
				})
			}
			a.store.SetContainers(containers.Attach(items, hosts))
		}
	}
	if owned, err := a.rules.List(ctx); err == nil {
		a.store.SetOwned(owned)
	}
}

// SetInvalidate lets a finished privileged call redraw the window.
func (a *App) SetInvalidate(fn func()) { a.invalidate = fn }

// WantContainers loads Docker, LXC, and libvirt on the next scans.
func (a *App) WantContainers() { a.wantBoxes = true }

// ToggleAuthorization requests or revokes root authorization.
func (a *App) ToggleAuthorization() {
	if a.store.Copy().Authorized {
		a.RevokeAuthorization()
	} else {
		a.Authorize()
	}
}

// Authorize requests root credentials via pkexec and loads privileged process/firewall data.
func (a *App) Authorize() {
	go func() {
		a.store.SetBusy("Authorizing with system...")
		if a.invalidate != nil {
			a.invalidate()
		}
		res, err := privilege.Call(privilege.Message{Op: "authorize"})
		if err != nil {
			a.store.SetBusy("")
			a.store.SetBanner("Authorization cancelled: " + err.Error())
			a.log.Warn("authorize_failed", "err", err)
			if a.invalidate != nil {
				a.invalidate()
			}
			return
		}
		a.store.SetBanner("")
		a.engine.SetPrivilegedProcs(res.Procs)
		a.store.SetAuthorized(true, res.Report)
		a.log.Info("authorized", "procs", len(res.Procs), "firewall_mode", res.Report.Mode)
		a.store.RequestRescan()
		if a.invalidate != nil {
			a.invalidate()
		}
	}()
}

// RevokeAuthorization returns the session to unprivileged user mode.
func (a *App) RevokeAuthorization() {
	a.engine.SetPrivilegedProcs(nil)
	a.store.SetAuthorized(false, fw.Preview())
	a.log.Info("authorization_revoked")
	a.store.RequestRescan()
	if a.invalidate != nil {
		a.invalidate()
	}
}

// Privileged runs one helper operation off the UI goroutine.
func (a *App) Privileged(msg privilege.Message) {
	go func() {
		a.store.SetBusy("Waiting for authorization")
		if a.invalidate != nil {
			a.invalidate()
		}
		res, err := privilege.Call(msg)
		action := events.ActionFWRead
		switch msg.Op {
		case "apply":
			action = events.ActionFWApply
		case "delete":
			action = events.ActionFWDelete
		case "reconcile":
			action = events.ActionFWReconcile
		case "rollback":
			action = events.ActionFWDelete
		case "service":
			action = events.ActionService
		case "signal":
			action = events.ActionSignal
		}
		result := events.ResultSuccess
		detail := res.Detail
		if err != nil {
			result = events.ResultFailed
			detail = err.Error()
			a.store.SetBanner(err.Error())
			a.log.Error("privileged", "op", msg.Op, "rule", msg.RuleID, "service", msg.Service, "err", err)
		} else {
			a.store.SetBanner("")
			a.log.Info("privileged", "op", msg.Op, "rule", msg.RuleID, "service", msg.Service, "detail", detail)
		}
		if len(res.Procs) > 0 {
			a.engine.SetPrivilegedProcs(res.Procs)
		}
		if res.Report.Authorized || res.Report.Mode != "" {
			a.store.SetFirewall(res.Report)
		} else {
			a.store.SetBusy("")
		}
		a.rememberRule(msg, res.Detail, err == nil)
		_ = a.history.Insert(context.Background(), repositories.Entry{
			Action:   action,
			Backend:  res.Report.Mode,
			Protocol: msg.Rule.Protocol,
			Port:     msg.Rule.Port,
			Target:   msg.Service + msg.RuleID,
			Result:   result,
			Detail:   detail,
			User:     a.userName,
		})
		a.store.RequestRescan()
		if a.invalidate != nil {
			a.invalidate()
		}
	}()
}

// SetTheme persists and publishes a theme.
func (a *App) SetTheme(theme string) error {
	a.cfg.Theme = theme
	return a.savePrefs("theme")
}

// SetRefresh persists a scan interval in seconds.
func (a *App) SetRefresh(seconds int) error {
	a.cfg.RefreshSeconds = seconds
	return a.savePrefs("refresh")
}

// SetLogRetention persists log retention days and prunes older logs.
func (a *App) SetLogRetention(days int) error {
	a.cfg.LogRetentionDays = days
	a.store.SetLogRetention(days)
	_ = logging.DefaultStore.Rotate(days)
	return a.savePrefs("log_retention")
}

// RotateLogs triggers an immediate on-disk log rotation.
func (a *App) RotateLogs() error {
	days := a.cfg.LogRetentionDays
	if days <= 0 {
		days = config.DefaultLogRetentionDays
	}
	err := logging.DefaultStore.Rotate(days)
	if err != nil {
		a.log.Error("rotate_logs", "days", days, "err", err)
	} else {
		a.log.Info("rotate_logs", "days", days, "status", "rotated")
	}
	return err
}

func (a *App) savePrefs(what string) error {
	if err := a.cfg.Validate(); err != nil {
		a.store.SetPrefs(a.store.Copy().Theme, a.store.Refresh(), err.Error())
		return err
	}
	if err := config.Save(a.paths.Config, a.cfg); err != nil {
		a.store.SetPrefs(a.cfg.Theme, a.cfg.Refresh(), err.Error())
		return err
	}
	ctx := context.Background()
	_ = a.settings.Put(ctx, "theme", a.cfg.Theme)
	_ = a.settings.Put(ctx, "refresh_seconds", strconv.Itoa(a.cfg.RefreshSeconds))
	_ = a.settings.Put(ctx, "log_retention_days", strconv.Itoa(a.cfg.LogRetentionDays))
	_ = a.history.Insert(ctx, repositories.Entry{
		Action: events.ActionSettings,
		Result: events.ResultSuccess,
		Target: what,
		Detail: fmt.Sprintf("theme=%s refresh=%s retention=%dd", a.cfg.Theme, a.cfg.Refresh(), a.cfg.LogRetentionDays),
		User:   a.userName,
	})
	a.store.SetPrefs(a.cfg.Theme, a.cfg.Refresh(), "")
	a.log.Info("settings", "theme", a.cfg.Theme, "refresh", a.cfg.Refresh(), "retention_days", a.cfg.LogRetentionDays)
	return nil
}

func (a *App) rememberRule(msg privilege.Message, detail string, ok bool) {
	if !ok || a.rules == nil {
		return
	}
	ctx := context.Background()
	id := msg.RuleID
	if id == "" && detail != "" {
		id = strings.TrimPrefix(strings.Fields(detail)[0], "portpilot:")
	}
	switch msg.Op {
	case "apply":
		if id == "" {
			return
		}
		_ = a.rules.Upsert(ctx, repositories.OwnedRule{
			Identity: id, Backend: "", Protocol: msg.Rule.Protocol, Port: msg.Rule.Port,
			Action: msg.Rule.Action, Source: msg.Rule.Source, Iface: msg.Rule.Interface,
			Family: msg.Rule.Family, DurationSec: msg.Rule.DurationSec, Status: "active",
		})
	case "delete":
		if id == "" {
			return
		}
		if msg.Keep {
			_ = a.rules.SetStatus(ctx, id, "disabled")
		} else {
			_ = a.rules.Delete(ctx, id)
		}
	}
}

func currentUser() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.Username
}
