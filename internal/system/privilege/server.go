package privilege

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/firewall"
	"github.com/0xamirreza/portpilot/internal/discovery/procfs"
	"github.com/0xamirreza/portpilot/internal/discovery/sockets"
	fw "github.com/0xamirreza/portpilot/internal/firewall"
)

// Message is one helper request. Op is an allowlisted name.
type Message struct {
	Op          string           `json:"op"`
	Rule        firewall.Request `json:"rule,omitempty"`
	RuleID      string           `json:"rule_id,omitempty"`
	Service     string           `json:"service,omitempty"`
	ServiceOp   string           `json:"service_op,omitempty"`
	PID         int              `json:"pid,omitempty"`
	StartTicks  uint64           `json:"start_ticks,omitempty"`
	Signal      string           `json:"signal,omitempty"`
	ConfirmKill bool             `json:"confirm_kill,omitempty"`
	Keep        bool             `json:"keep,omitempty"`
}

// Result is the helper response.
type Result struct {
	OK     bool            `json:"ok"`
	Error  string          `json:"error,omitempty"`
	Report firewall.Report `json:"report,omitempty"`
	Detail string          `json:"detail,omitempty"`
	Procs  []procfs.Proc   `json:"procs,omitempty"`
}

type hostRunner struct{}

func (hostRunner) Run(name string, args ...string) (string, error) {
	return Exec(name, args...)
}

// Serve reads one JSON message from r and writes one JSON result.
// expireID, when set, deletes that scheduled rule and does not read r.
func Serve(r io.Reader, w io.Writer, expireID string) error {
	if os.Geteuid() != 0 {
		_ = writeResult(w, Result{Error: "the helper is not root"})
		return fmt.Errorf("the helper is not root")
	}
	if expireID != "" {
		return writeResult(w, expireOne(expireID))
	}
	var msg Message
	dec := json.NewDecoder(io.LimitReader(r, 1<<20))
	if err := dec.Decode(&msg); err != nil {
		return writeResult(w, Result{Error: "invalid request"})
	}
	return writeResult(w, dispatch(msg))
}

func dispatch(msg Message) Result {
	switch msg.Op {
	case "authorize", "procs":
		procs, _, _ := procfs.WalkProcesses()
		rec := reconcile(time.Now())
		audit("authorize", true, "root inspection and process ownership")
		return Result{
			OK:     true,
			Report: rec.Report,
			Procs:  procs,
			Detail: "root authorized",
		}
	case "inspect":
		procs, _, _ := procfs.WalkProcesses()
		rec := reconcile(time.Now())
		if rec.Report.ReadAt.IsZero() && rec.Error != "" {
			return rec
		}
		rec.Procs = procs
		audit("inspect", rec.Error == "" && rec.OK, rec.Detail)
		return rec
	case "reconcile":
		return reconcile(time.Now())
	case "apply":
		return applyRule(msg.Rule)
	case "delete":
		return deleteRule(msg.RuleID)
	case "rollback":
		return rollback()
	case "service":
		return serviceAction(msg.Service, msg.ServiceOp)
	case "signal":
		return signal(msg)
	default:
		return Result{Error: "unknown operation"}
	}
}

func applyRule(req firewall.Request) Result {
	ssh := map[int]bool{22: true}
	if snap, err := (&sockets.Engine{}).Scan(); err == nil {
		ssh = firewall.SSHPorts(snap.Sockets)
	}
	if err := firewall.Validate(req, ssh); err != nil {
		audit("apply", false, err.Error())
		return Result{Error: err.Error()}
	}
	rep, err := fw.Inspect(hostRunner{})
	if err != nil {
		return Result{Error: err.Error()}
	}
	if rep.Mode != firewall.ModeUFW && rep.Mode != firewall.ModeNFTables {
		return Result{Error: rep.Reason}
	}
	id, err := newID()
	if err != nil {
		return Result{Error: err.Error()}
	}
	comment := firewall.CommentPrefix + id
	native, err := fw.Apply(hostRunner{}, rep, req, comment)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if req.DurationSec > 0 {
		item := firewall.Schedule{
			ID: id, Backend: rep.Mode, Protocol: strings.ToLower(req.Protocol),
			Port: req.Port, Action: req.Action, Expires: firewall.Expires(time.Now(), req.DurationSec),
		}
		if err := remember(item); err != nil {
			_ = deleteByID(id)
			return Result{Error: "the rule was removed because its expiry could not be stored: " + err.Error()}
		}
		if err := armTimer(id, req.DurationSec); err != nil {
			if native {
				audit("apply", true, comment+" nft-timeout-without-systemd")
				_ = saveLast(id)
				_ = rememberOwned(id, req)
				return Result{OK: true, Detail: comment + " kernel timeout is set; systemd did not arm a timer, so the next reconcile removes the leftover rule", Report: mustInspect()}
			}
			_ = deleteByID(id)
			_ = forget(id)
			audit("apply", false, err.Error())
			return Result{Error: "expiration could not be scheduled, so the rule was removed: " + err.Error()}
		}
	}
	audit("apply", true, comment)
	_ = saveLast(id)
	_ = rememberOwned(id, req)
	return Result{OK: true, Detail: comment, Report: mustInspect()}
}

func deleteRule(id string) Result {
	if (firewall.Rule{Comment: firewall.CommentPrefix + id}).ID() == "" {
		return Result{Error: "rule id is not valid"}
	}
	if err := deleteByID(id); err != nil {
		return Result{Error: err.Error()}
	}
	_ = forget(id)
	_ = forgetOwned(id)
	_, _ = Exec("systemctl", "stop", "portpilot-expire-"+id+".timer")
	audit("delete", true, id)
	return Result{OK: true, Detail: "deleted " + id, Report: mustInspect()}
}

func deleteByID(id string) error {
	rep, err := fw.Inspect(hostRunner{})
	if err != nil {
		return err
	}
	rule, ok := fw.FindOwned(rep.Rules, id)
	if !ok {
		return nil
	}
	return fw.DeleteOwned(hostRunner{}, rep, rule)
}

func reconcile(now time.Time) Result {
	items, err := due(now)
	if err != nil {
		return Result{Error: err.Error()}
	}
	var notes []string
	for _, item := range items {
		if err := deleteByID(item.ID); err != nil {
			notes = append(notes, item.ID+": "+err.Error())
			continue
		}
		_ = forget(item.ID)
		_ = forgetOwned(item.ID)
		notes = append(notes, "expired "+item.ID)
	}
	return Result{OK: len(notes) == 0 || err == nil, Detail: strings.Join(notes, "; "), Report: mustInspect()}
}

func expireOne(id string) Result {
	if (firewall.Rule{Comment: firewall.CommentPrefix + id}).ID() == "" {
		return Result{Error: "rule id is not valid"}
	}
	file, err := loadSchedule()
	if err != nil {
		return Result{Error: err.Error()}
	}
	var item *firewall.Schedule
	for i := range file.Items {
		if file.Items[i].ID == id {
			item = &file.Items[i]
			break
		}
	}
	if item == nil {
		return Result{Error: "no scheduled rule with that id"}
	}
	if item.Expires.After(time.Now().Add(5 * time.Second)) {
		return Result{Error: "that rule is not due"}
	}
	if err := deleteByID(id); err != nil {
		return Result{Error: err.Error()}
	}
	_ = forget(id)
	_ = forgetOwned(id)
	return Result{OK: true, Detail: "expired " + id}
}

func armTimer(id string, seconds int) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	unit := "portpilot-expire-" + id
	_, err = Exec("systemd-run", "--collect", "--unit="+unit, "--on-active="+strconv.Itoa(seconds)+"s",
		"--description=PortPilot expire "+id, exe, "__privileged", "expire", id)
	return err
}

func serviceAction(name, op string) Result {
	comm, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		return Result{Error: "init system could not be read"}
	}
	bin, args, err := serviceCommand(strings.TrimSpace(string(comm)), op, name)
	if err != nil {
		return Result{Error: err.Error()}
	}
	var out string
	if strings.Contains(bin, "/") {
		out, err = execFile(bin, args...)
	} else {
		out, err = Exec(bin, args...)
	}
	if err != nil {
		audit("service", false, op+" "+name)
		return Result{Error: err.Error(), Detail: out}
	}
	audit("service", true, op+" "+name)
	return Result{OK: true, Detail: op + " " + name}
}

func signal(msg Message) Result {
	if msg.PID <= 1 {
		return Result{Error: "that pid is not a target"}
	}
	sig := 15
	if msg.Signal == "KILL" {
		if !msg.ConfirmKill {
			return Result{Error: "SIGKILL requires an explicit confirmation"}
		}
		sig = 9
	} else if msg.Signal != "TERM" {
		return Result{Error: "signal must be TERM or KILL"}
	}
	start, err := startTicks(msg.PID)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if start != msg.StartTicks {
		return Result{Error: "the process changed; the pid was not signalled"}
	}
	proc, err := os.FindProcess(msg.PID)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if err := proc.Signal(unixSignal(sig)); err != nil {
		audit("signal", false, err.Error())
		return Result{Error: err.Error()}
	}
	detail := fmt.Sprintf("signal %d to %d", sig, msg.PID)
	audit("signal", true, detail)
	return Result{OK: true, Detail: detail}
}

func rollback() Result {
	file, err := loadOwned()
	if err != nil {
		return Result{Error: err.Error()}
	}
	if len(file.Rules) == 0 {
		if id, err := loadLast(); err == nil && id != "" {
			file.Rules = []ownedRule{{ID: id}}
		}
	}
	if len(file.Rules) == 0 {
		return Result{Error: "there is no PortPilot snapshot to roll back"}
	}
	var notes []string
	var failed int
	for _, rule := range file.Rules {
		if err := deleteByID(rule.ID); err != nil {
			failed++
			notes = append(notes, rule.ID+": "+err.Error())
			continue
		}
		_ = forget(rule.ID)
		notes = append(notes, "removed "+rule.ID)
	}
	_ = saveOwned(ownedFile{})
	_ = saveLast("")
	audit("rollback", failed == 0, strings.Join(notes, "; "))
	res := Result{OK: failed == 0, Detail: strings.Join(notes, "; "), Report: mustInspect()}
	if failed > 0 && res.Detail == "" {
		res.Error = "rollback did not finish"
	}
	if failed > 0 {
		res.Error = res.Detail
	}
	return res
}

func saveLast(id string) error {
	if err := os.MkdirAll("/var/lib/portpilot", 0o700); err != nil {
		return err
	}
	return os.WriteFile("/var/lib/portpilot/last-change", []byte(id), 0o600)
}

func loadLast() (string, error) {
	raw, err := os.ReadFile("/var/lib/portpilot/last-change")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func mustInspect() firewall.Report {
	rep, err := fw.Inspect(hostRunner{})
	if err != nil {
		rep.Error = err.Error()
	}
	file, _ := loadSchedule()
	rep.Schedules = file.Items
	return rep
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func writeResult(w io.Writer, res Result) error {
	enc := json.NewEncoder(w)
	return enc.Encode(res)
}

func cleanUnit(name string) (string, error) {
	name = strings.TrimSpace(name)
	if strings.Contains(name, "/") || strings.Contains(name, "..") || strings.ContainsAny(name, " \t\n;|&$`") {
		return "", fmt.Errorf("service name is not accepted")
	}
	if !strings.HasSuffix(name, ".service") {
		name += ".service"
	}
	if len(name) < len(".service")+1 || len(name) > 256 {
		return "", fmt.Errorf("service name is not accepted")
	}
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '@', c == '.', c == '_', c == '-', c == ':':
		default:
			return "", fmt.Errorf("service name is not accepted")
		}
	}
	return name, nil
}
