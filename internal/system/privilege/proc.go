package privilege

import (
	"fmt"
	"os"
	"syscall"

	"github.com/0xamirreza/portpilot/internal/discovery/procfs"
)

func startTicks(pid int) (uint64, error) {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, fmt.Errorf("process %d is not readable", pid)
	}
	_, start, ok := procfs.CPUTicks(string(raw))
	if !ok {
		return 0, fmt.Errorf("process %d identity could not be read", pid)
	}
	return start, nil
}

func unixSignal(sig int) syscall.Signal {
	return syscall.Signal(sig)
}
