package services

import "testing"

func TestParseSystemd(t *testing.T) {
	text := "accounts-daemon.service loaded active running Accounts Service\n" +
		"ssh.service active running OpenBSD Secure Shell server\n" +
		"alsa-restore.service active exited Save/Restore Sound Card State\n" +
		"● failed.service loaded failed failed Broken Service\n" +
		"not-a-unit loaded active running skip\n"
	units := ParseSystemd(text)
	if len(units) != 4 {
		t.Fatalf("expected 4 units, got %d: %+v", len(units), units)
	}
	if units[0].Name != "accounts-daemon.service" || units[0].Load != "loaded" || units[0].Active != "active" || units[0].Sub != "running" {
		t.Errorf("unexpected units[0]: %+v", units[0])
	}
	if units[1].Name != "ssh.service" || units[1].Load != "loaded" || units[1].Active != "active" || units[1].Sub != "running" {
		t.Errorf("unexpected units[1]: %+v", units[1])
	}
	if units[2].Name != "alsa-restore.service" || units[2].Active != "active" || units[2].Sub != "exited" {
		t.Errorf("unexpected units[2]: %+v", units[2])
	}
	if units[3].Name != "failed.service" || units[3].Active != "failed" || units[3].Sub != "failed" {
		t.Errorf("unexpected units[3]: %+v", units[3])
	}
}
