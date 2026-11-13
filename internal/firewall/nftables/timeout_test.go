package nftables

import (
	"strings"
	"testing"
)

func TestTimeoutCommandsStayInPortPilot(t *testing.T) {
	cmds, ok := TimeoutCommands("abcd1234", "tcp", "deny", 5900, 30*60)
	if !ok || len(cmds) != 3 {
		t.Fatal(cmds, ok)
	}
	for _, cmd := range cmds {
		joined := strings.Join(cmd, " ")
		if !strings.Contains(joined, "inet portpilot") || strings.Contains(joined, "flush") {
			t.Fatal(cmd)
		}
	}
	if cmds[2][len(cmds[2])-1] != `"portpilot:abcd1234"` {
		t.Fatal(cmds[2])
	}
}

func TestTimeoutRejectsForeignShape(t *testing.T) {
	if _, ok := TimeoutCommands("../x", "tcp", "deny", 1, 60); ok {
		t.Fatal("bad id accepted")
	}
}
