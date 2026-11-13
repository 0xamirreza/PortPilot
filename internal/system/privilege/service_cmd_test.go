package privilege

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOpenRCCommand(t *testing.T) {
	bin, args, err := serviceCommand("openrc-init", "restart", "sshd.service")
	if err != nil || bin != "rc-service" || args[0] != "sshd" || args[1] != "restart" {
		t.Fatalf("%s %v %v", bin, args, err)
	}
	bin, args, err = serviceCommand("openrc-init", "enable", "sshd")
	if err != nil || bin != "rc-update" || args[0] != "add" {
		t.Fatalf("%s %v %v", bin, args, err)
	}
}

func TestOpenRCCommandRuns(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "ran")
	script := filepath.Join(dir, "rc-service")
	body := "#!/bin/sh\nprintf '%s\\n' \"$1 $2\" > " + logPath + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	bin, args, err := serviceCommand("openrc-init", "restart", "sshd.service")
	if err != nil || bin != "rc-service" {
		t.Fatal(bin, args, err)
	}
	cmd := exec.Command(script, args...)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(logPath)
	if err != nil || string(got) != "sshd restart\n" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestSysVScriptRuns(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "ran")
	script := filepath.Join(dir, "demo")
	body := "#!/bin/sh\nprintf '%s\\n' \"$1\" > " + logPath + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	_, args, err := serviceCommand("init", "stop", "demo")
	if err == nil {
		t.Fatal("missing init script was accepted")
	}
	cmd := exec.Command(script, "stop")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(logPath)
	if err != nil || string(got) != "stop\n" {
		t.Fatalf("%q %v args %v", got, err, args)
	}
}

func TestSysVRejectsEnable(t *testing.T) {
	if _, _, err := serviceCommand("init", "enable", "ssh"); err == nil {
		t.Fatal("sysv enable was accepted")
	}
	if _, _, err := serviceCommand("init", "start", "ssh;id"); err == nil {
		t.Fatal("bad name accepted")
	}
}
