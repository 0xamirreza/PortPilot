package nftables

import (
	"strings"
	"testing"
)

const sample = `{
  "nftables": [
    {"metainfo": {"version": "1.0.2"}},
    {"table": {"family": "inet", "name": "filter"}},
    {"table": {"family": "inet", "name": "portpilot"}},
    {"rule": {"family": "inet", "table": "portpilot", "chain": "input", "handle": 7, "comment": "portpilot:abcd1234",
      "expr": [{"match": {"left": {"payload": {"protocol": "tcp", "field": "dport"}}, "right": 5900}}, {"drop": null}]}},
    {"rule": {"family": "inet", "table": "filter", "chain": "input", "handle": 3, "comment": "other",
      "expr": [{"match": {"left": {"payload": {"protocol": "tcp", "field": "dport"}}, "right": 22}}, {"accept": null}]}}
  ]
}`

func TestParseJSON(t *testing.T) {
	sum, err := ParseJSON(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Rules) != 2 {
		t.Fatalf("rules %d", len(sum.Rules))
	}
	if !sum.Rules[0].Owned || sum.Rules[0].Port != 5900 || sum.Rules[0].Handle != 7 || sum.Rules[0].Action != "deny" {
		t.Fatalf("%+v", sum.Rules[0])
	}
	if sum.Rules[1].Owned {
		t.Fatal("foreign rule marked owned")
	}
	other := OtherTables(sum)
	if len(other) != 1 || other[0] != "inet filter" {
		t.Fatal(other)
	}
}

func TestAddRuleStaysInPortPilotTable(t *testing.T) {
	args := AddRuleArgs("tcp", 5900, "ipv4", "", "", "deny", "portpilot:abcd1234")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "inet portpilot input") || strings.Contains(joined, "flush") {
		t.Fatal(args)
	}
	if args[0] != "add" || args[1] != "rule" {
		t.Fatal(args)
	}
}
