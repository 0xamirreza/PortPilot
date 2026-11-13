package procfs

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/0xamirreza/portpilot/internal/core/processes"
)

// Proc is the ownership record read from /proc before sockets are attached.
type Proc struct {
	processes.Process
	Inodes []uint64
	Ticks  uint64
	Denied bool
}

// WalkProcesses reads the processes this user is allowed to inspect.
// Permission errors are counted and do not fail the walk.
func WalkProcesses() ([]Proc, int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, 0, fmt.Errorf("read /proc: %w", err)
	}
	var out []Proc
	denied := 0
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(ent.Name())
		if err != nil || pid <= 0 {
			continue
		}
		proc, fdDenied, ok := readProc(pid)
		if !ok {
			continue
		}
		if fdDenied {
			denied++
			proc.Denied = true
		}
		out = append(out, proc)
	}
	return out, denied, nil
}

func readProc(pid int) (Proc, bool, bool) {
	base := "/proc/" + strconv.Itoa(pid)
	var proc Proc
	proc.PID = pid
	comm, err := os.ReadFile(base + "/comm")
	if err != nil {
		if os.IsNotExist(err) {
			return Proc{}, false, false
		}
		return Proc{}, false, false
	}
	proc.Name = strings.TrimSpace(string(comm))
	if exe, err := os.Readlink(base + "/exe"); err == nil {
		proc.Executable = exe
	}
	if raw, err := os.ReadFile(base + "/cmdline"); err == nil {
		parts := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
		proc.Command = strings.Join(parts, " ")
	}
	if st, err := os.ReadFile(base + "/status"); err == nil {
		proc.PPID, proc.UID, proc.User = parseStatus(string(st))
	}
	if raw, err := os.ReadFile(base + "/stat"); err == nil {
		text := string(raw)
		if ticks, start, ok := CPUTicks(text); ok {
			proc.Ticks = ticks
			proc.StartTicks = start
		}
		proc.RSSBytes = rssBytes(text)
	}
	if raw, err := os.ReadFile(base + "/cgroup"); err == nil {
		proc.Service, proc.Container = ParseCgroup(string(raw))
	}
	if ns, err := os.Readlink(base + "/ns/net"); err == nil {
		proc.NetNS = ns
	}
	inodes, denied := socketInodes(base + "/fd")
	if !denied {
		if tasks, err := os.ReadDir(base + "/task"); err == nil && len(tasks) > 1 {
			seenInodes := make(map[uint64]bool, len(inodes))
			for _, in := range inodes {
				seenInodes[in] = true
			}
			pidStr := strconv.Itoa(pid)
			for _, t := range tasks {
				tid := t.Name()
				if tid == pidStr {
					continue
				}
				if tInodes, _ := socketInodes(base + "/task/" + tid + "/fd"); len(tInodes) > 0 {
					for _, in := range tInodes {
						if !seenInodes[in] {
							inodes = append(inodes, in)
							seenInodes[in] = true
						}
					}
				}
			}
		}
	}
	proc.Inodes = inodes
	return proc, denied, true
}

func socketInodes(fdDir string) ([]uint64, bool) {
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		if os.IsPermission(err) {
			return nil, true
		}
		return nil, false
	}
	var inodes []uint64
	for _, ent := range entries {
		target, err := os.Readlink(fdDir + "/" + ent.Name())
		if err != nil {
			continue
		}
		if inode, ok := socketInode(target); ok {
			inodes = append(inodes, inode)
		}
	}
	return inodes, false
}

func socketInode(link string) (uint64, bool) {
	link = strings.TrimSpace(link)
	if strings.HasPrefix(link, "socket:[") && strings.HasSuffix(link, "]") {
		n, err := strconv.ParseUint(link[len("socket:["):len(link)-1], 10, 64)
		if err == nil {
			return n, true
		}
	}
	if strings.HasPrefix(link, "[0000]:") {
		n, err := strconv.ParseUint(strings.TrimPrefix(link, "[0000]:"), 10, 64)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func parseStatus(text string) (ppid int, uid uint32, user string) {
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "PPid:"):
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "PPid:")), "%d", &ppid)
		case strings.HasPrefix(line, "Uid:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 32)
				uid = uint32(v)
			}
		}
	}
	return ppid, uid, user
}

func rssBytes(text string) uint64 {
	closeParen := strings.LastIndex(text, ")")
	if closeParen < 0 || closeParen+2 >= len(text) {
		return 0
	}
	fields := strings.Fields(text[closeParen+2:])
	if len(fields) <= 21 {
		return 0
	}
	pages, err := strconv.ParseUint(fields[21], 10, 64)
	if err != nil {
		return 0
	}
	return pages * uint64(os.Getpagesize())
}

// CPUTicks returns utime+stime and starttime from /proc/pid/stat.
func CPUTicks(stat string) (ticks uint64, start uint64, ok bool) {
	closeParen := strings.LastIndex(stat, ")")
	if closeParen < 0 || closeParen+2 >= len(stat) {
		return 0, 0, false
	}
	fields := strings.Fields(stat[closeParen+2:])
	if len(fields) <= 19 {
		return 0, 0, false
	}
	utime, err1 := strconv.ParseUint(fields[11], 10, 64)
	stime, err2 := strconv.ParseUint(fields[12], 10, 64)
	start, err3 := strconv.ParseUint(fields[19], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, false
	}
	return utime + stime, start, true
}
