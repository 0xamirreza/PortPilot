package procfs

import "testing"

func TestParseCgroupService(t *testing.T) {
	svc, box := ParseCgroup("0::/system.slice/ssh.service\n")
	if svc != "ssh.service" || box != "" {
		t.Fatalf("svc=%q box=%q", svc, box)
	}
}

func TestParseCgroupIgnoresUserManager(t *testing.T) {
	svc, _ := ParseCgroup("0::/user.slice/user-1000.slice/user@1000.service/app.slice/portpilot.scope\n")
	if svc != "" {
		t.Fatalf("scope reported as service %q", svc)
	}
}

func TestParseCgroupDockerScope(t *testing.T) {
	_, box := ParseCgroup("0::/system.slice/docker-0123456789abcdef0123.scope\n")
	if box != "docker:0123456789ab" {
		t.Fatal(box)
	}
}

func TestParseCgroupLXC(t *testing.T) {
	_, box := ParseCgroup("0::/lxc/web/init.scope\n")
	if box != "lxc:web" {
		t.Fatal(box)
	}
}

func TestStatStartTime(t *testing.T) {
	stat := "12 (my proc) S 1 12 12 0 -1 4194560 1 2 3 4 10 20 0 0 20 0 1 0 999 123 45 0"
	ticks, start, ok := CPUTicks(stat)
	if !ok || ticks != 30 || start != 999 {
		t.Fatalf("ticks=%d start=%d ok=%v", ticks, start, ok)
	}
}
