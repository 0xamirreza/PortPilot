package ufw

import "testing"

const sample = `
Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)

	To                         Action      From
--                         ------      ----
22/tcp                     ALLOW IN    Anywhere                   # OpenSSH
5900/tcp                   DENY IN     10.0.0.0/8                 # portpilot:abcd1234
22/tcp (v6)                ALLOW IN    Anywhere (v6)              # OpenSSH
80/tcp                     REJECT IN   Anywhere
`

func TestParseStatus(t *testing.T) {
	st := ParseStatus(sample)
	if !st.Known || !st.Active {
		t.Fatalf("active %+v", st)
	}
	if st.DefaultInbound != "deny" || st.DefaultOutbound != "allow" {
		t.Fatalf("defaults in=%q out=%q", st.DefaultInbound, st.DefaultOutbound)
	}
	if len(st.Rules) != 4 {
		t.Fatalf("rules %d", len(st.Rules))
	}
	if st.Rules[1].Port != 5900 || !st.Rules[1].Owned || st.Rules[1].Source != "10.0.0.0/8" || st.Rules[1].Action != "deny" {
		t.Fatalf("owned %+v", st.Rules[1])
	}
	if st.Rules[2].Family != "ipv6" {
		t.Fatal(st.Rules[2].Family)
	}
	if st.Rules[3].Action != "reject" || st.Rules[3].Port != 80 {
		t.Fatalf("%+v", st.Rules[3])
	}
}

func TestInactive(t *testing.T) {
	st := ParseStatus("Status: inactive\n")
	if !st.Known || st.Active {
		t.Fatalf("%+v", st)
	}
}

func TestCommandIsArgs(t *testing.T) {
	args := Command("deny", "tcp", 5900, "", "portpilot:abcd1234", false)
	if args[0] != "deny" || args[len(args)-1] != "portpilot:abcd1234" {
		t.Fatal(args)
	}
	for _, a := range args {
		if stringsContainsShell(a) {
			t.Fatal(a)
		}
	}
}

func stringsContainsShell(s string) bool {
	return stringsIndex(s, ";") >= 0 || stringsIndex(s, "|") >= 0 || stringsIndex(s, "`") >= 0
}

func stringsIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
