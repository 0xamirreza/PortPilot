package ss

import "testing"

func TestNormalizeSS(t *testing.T) {
	if got := normalizeSS("[::]:22"); got != ":::22" {
		t.Fatal(got)
	}
	if got := normalizeSS("127.0.0.53%lo:53"); got != "127.0.0.53:53" {
		t.Fatal(got)
	}
}
