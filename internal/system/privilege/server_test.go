package privilege

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCleanUnit(t *testing.T) {
	got, err := cleanUnit("ssh")
	if err != nil || got != "ssh.service" {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := cleanUnit("ssh;reboot"); err == nil {
		t.Fatal("shell metacharacter was accepted")
	}
	if _, err := cleanUnit("../ssh"); err == nil {
		t.Fatal("path was accepted")
	}
}

func TestLookRejectsUnexpected(t *testing.T) {
	if _, err := Look("bash"); err == nil {
		t.Fatal("bash was allowed")
	}
	if _, err := Look("ufw;id"); err == nil {
		t.Fatal("compound name was allowed")
	}
}

func TestServeRefusesNonRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("this check is for an unprivileged process")
	}
	var buf bytes.Buffer
	err := Serve(strings.NewReader(`{"op":"inspect"}`), &buf, "")
	if err == nil {
		t.Fatal("non-root helper returned success")
	}
	if !strings.Contains(buf.String(), "not root") {
		t.Fatal(buf.String())
	}
}

func TestDispatchRejectsUnknown(t *testing.T) {
	res := dispatch(Message{Op: "shell"})
	if res.OK || res.Error == "" {
		t.Fatal(res)
	}
}
