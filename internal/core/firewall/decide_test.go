package firewall

import "testing"

func TestDecideUFWWinsWhenActive(t *testing.T) {
	mode, reason := Decide(Report{Authorized: true, UFWKnown: true, UFWActive: true, UFWInstalled: true, OtherNFTTables: []string{"inet filter"}})
	if mode != ModeUFW {
		t.Fatalf("mode %s (%s)", mode, reason)
	}
}

func TestDecideAmbiguousTablesAreReadOnly(t *testing.T) {
	mode, _ := Decide(Report{Authorized: true, UFWKnown: true, UFWInstalled: true, NftReadable: true, NftInstalled: true, OtherNFTTables: []string{"inet filter"}})
	if mode != ModeReadOnly {
		t.Fatal(mode)
	}
}

func TestDecideOwnTableWhenHostIsEmpty(t *testing.T) {
	mode, _ := Decide(Report{Authorized: true, UFWKnown: true, NftInstalled: true, NftReadable: true})
	if mode != ModeNFTables {
		t.Fatal(mode)
	}
}

func TestDecideLegacyConflict(t *testing.T) {
	mode, _ := Decide(Report{Authorized: true, UFWKnown: true, UFWActive: true, LegacyRules: 3})
	if mode != ModeReadOnly {
		t.Fatal(mode)
	}
}

func TestUnauthorizedStaysClosed(t *testing.T) {
	mode, _ := Decide(Report{UFWInstalled: true})
	if mode != ModeUnavailable {
		t.Fatal(mode)
	}
}

func TestValidateRejectsLoopbackAndSSH(t *testing.T) {
	err := Validate(Request{Action: ActionDeny, Protocol: "tcp", Port: 22}, map[int]bool{22: true})
	if err == nil {
		t.Fatal("ssh block without confirmation was accepted")
	}
	err = Validate(Request{Action: ActionDeny, Protocol: "tcp", Port: 22, ConfirmSSH: true}, map[int]bool{22: true})
	if err != nil {
		t.Fatal(err)
	}
	err = Validate(Request{Action: ActionDeny, Protocol: "tcp", Port: 9, Interface: "lo"}, nil)
	if err == nil {
		t.Fatal("lo was accepted")
	}
	err = Validate(Request{Action: ActionAllow, Protocol: "tcp", Port: 80, Source: "127.0.0.1"}, nil)
	if err == nil {
		t.Fatal("loopback source was accepted")
	}
}

func TestOwnedComment(t *testing.T) {
	if !OwnedComment("portpilot:abcd1234") {
		t.Fatal("expected owned")
	}
	if OwnedComment("portpilot:nope!") || OwnedComment("openssh") {
		t.Fatal("unexpected owned")
	}
}
