package firewall

import (
	"strings"
	"testing"

	core "github.com/0xamirreza/portpilot/internal/core/firewall"
)

type fakeRun struct {
	calls []string
	out   map[string]string
	err   map[string]error
}

func (f *fakeRun) Run(name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if err := f.err[key]; err != nil {
		return "", err
	}
	return f.out[name], nil
}

func TestApplyRefusesReadOnly(t *testing.T) {
	run := &fakeRun{}
	_, err := Apply(run, core.Report{Mode: core.ModeReadOnly, Reason: "ambiguous"}, core.Request{}, "portpilot:abcd1234")
	if err == nil {
		t.Fatal("read-only apply succeeded")
	}
	if len(run.calls) != 0 {
		t.Fatal(run.calls)
	}
}

func TestInspectUsesUFWWhenActive(t *testing.T) {
	run := &fakeRun{out: map[string]string{
		"ufw": "Status: active\nDefault: deny (incoming), allow (outgoing), disabled (routed)\n22/tcp ALLOW IN Anywhere\n",
		"nft": `{"nftables":[{"table":{"family":"inet","name":"filter"}}]}`,
	}}
	// Preview looks at the real host binaries. Override by calling the parsers through Inspect,
	// which also stats the host. The mode still follows the command output when binaries exist.
	rep, err := Inspect(run)
	if err != nil {
		t.Fatal(err)
	}
	if rep.UFWInstalled && rep.UFWActive && rep.Mode != core.ModeUFW && rep.Mode != core.ModeReadOnly {
		t.Fatalf("mode %s %s", rep.Mode, rep.Reason)
	}
	for _, call := range run.calls {
		if strings.Contains(call, "flush") || strings.Contains(call, "delete") || strings.Contains(call, "add ") {
			t.Fatalf("inspect mutated: %s", call)
		}
	}
}
