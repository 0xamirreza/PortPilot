// Package processes describes socket owners.
package processes

// Process is a process that held at least one socket at scan time.
type Process struct {
	PID        int
	PPID       int
	Name       string
	Executable string
	Command    string
	User       string
	UID        uint32
	Service    string
	NetNS      string
	Container  string
	StartTicks uint64
	CPUPercent float64
	CPUKnown   bool
	RSSBytes   uint64
	SocketKeys []string
}

// Identity is the pair used later to avoid signalling a reused PID.
func (p Process) Identity() (int, uint64) {
	return p.PID, p.StartTicks
}
