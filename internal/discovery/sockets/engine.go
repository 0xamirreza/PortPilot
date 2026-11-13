// Package sockets builds a snapshot of bound sockets and their owners.
package sockets

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/network"
	"github.com/0xamirreza/portpilot/internal/core/ports"
	"github.com/0xamirreza/portpilot/internal/core/processes"
	"github.com/0xamirreza/portpilot/internal/discovery/interfaces"
	nl "github.com/0xamirreza/portpilot/internal/discovery/netlink"
	"github.com/0xamirreza/portpilot/internal/discovery/procfs"
	"github.com/0xamirreza/portpilot/internal/discovery/ss"
)

// Snapshot is one completed read of the host. It is not stored as kernel truth.
type Snapshot struct {
	Sockets    []ports.Socket
	Processes  []processes.Process
	Interfaces []network.Interface
	Source     string
	Warnings   []string
	ScannedAt  time.Time
	Duration   time.Duration
}

// Engine remembers the previous CPU sample so a later scan can show a rate.
type Engine struct {
	mu        sync.Mutex
	prev      map[int]cpuSample
	privProcs []procfs.Proc
}

// SetPrivilegedProcs updates the process cache obtained via root authorization.
func (e *Engine) SetPrivilegedProcs(procs []procfs.Proc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.privProcs = procs
}

type cpuSample struct {
	ticks uint64
	start uint64
	at    time.Time
}

// Scan reads sockets, then attaches process identity where /proc allows it.
func (e *Engine) Scan() (Snapshot, error) {
	started := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.prev == nil {
		e.prev = map[int]cpuSample{}
	}

	var warnings []string
	list, source, err := readSockets()
	if err != nil {
		return Snapshot{}, err
	}
	if source != "netlink" {
		warnings = append(warnings, "Netlink socket diagnostics were unavailable. This list was read from /proc.")
	}

	procs, denied, walkErr := procfs.WalkProcesses()
	if walkErr != nil {
		warnings = append(warnings, walkErr.Error())
	}

	privByPID := map[int]procfs.Proc{}
	for _, p := range e.privProcs {
		privByPID[p.PID] = p
	}

	allProcs := make([]procfs.Proc, 0, len(procs)+len(e.privProcs))
	seen := map[int]bool{}
	deniedCount := 0

	for _, p := range procs {
		seen[p.PID] = true
		if p.Denied {
			// Unprivileged read could not access /proc/[pid]/fd.
			// If we have privileged data for it, use that!
			if priv, ok := privByPID[p.PID]; ok && len(priv.Inodes) > 0 {
				allProcs = append(allProcs, priv)
				continue
			}
			deniedCount++
		} else {
			// Unprivileged read succeeded. It has the latest, live inodes!
			if priv, ok := privByPID[p.PID]; ok {
				if p.Service == "" {
					p.Service = priv.Service
				}
				if p.Container == "" {
					p.Container = priv.Container
				}
				if p.Executable == "" {
					p.Executable = priv.Executable
				}
				// Also merge any privileged inodes if present
				if len(priv.Inodes) > 0 {
					inodeSet := make(map[uint64]bool, len(p.Inodes)+len(priv.Inodes))
					for _, in := range p.Inodes {
						inodeSet[in] = true
					}
					for _, in := range priv.Inodes {
						if !inodeSet[in] {
							p.Inodes = append(p.Inodes, in)
							inodeSet[in] = true
						}
					}
				}
			}
		}
		allProcs = append(allProcs, p)
	}

	for _, priv := range e.privProcs {
		if !seen[priv.PID] {
			allProcs = append(allProcs, priv)
		}
	}

	if len(e.privProcs) > 0 {
		denied = 0
	} else {
		denied = deniedCount
	}

	if denied > 0 {
		warnings = append(warnings, fmt.Sprintf("%d process file-descriptor directories were not readable. Ownership is incomplete for those processes.", denied))
	}

	ifaces, ifErr := interfaces.List()
	if ifErr != nil {
		warnings = append(warnings, "Interface list failed: "+ifErr.Error())
	}
	names := map[int]string{}
	for _, ifc := range ifaces {
		names[ifc.Index] = ifc.Name
	}

	owners := map[uint64][]procfs.Proc{}
	for _, proc := range allProcs {
		for _, inode := range proc.Inodes {
			owners[inode] = append(owners[inode], proc)
		}
	}
	users := map[uint32]string{}
	now := time.Now()
	clk := clockTicks()
	seenPID := map[int]struct{}{}

	for i := range list {
		s := &list[i]
		s.User = lookupUser(users, s.UID)
		if s.IfIndex > 0 {
			s.Interface = names[int(s.IfIndex)]
		}
		cands := owners[s.Inode]
		if len(cands) == 0 {
			continue
		}
		owner := chooseOwner(cands, s.UID)
		s.OwnerKnown = true
		s.PID = owner.PID
		s.PPID = owner.PPID
		s.Process = owner.Name
		s.Executable = owner.Executable
		s.Command = owner.Command
		s.Service = owner.Service
		s.NetNS = owner.NetNS
		s.Container = owner.Container
		s.StartTicks = owner.StartTicks
		if owner.User == "" {
			owner.User = lookupUser(users, owner.UID)
		}
		if owner.User != "" {
			s.User = owner.User
		}
		seenPID[owner.PID] = struct{}{}
	}

	procOut := make([]processes.Process, 0, len(seenPID))
	nextPrev := map[int]cpuSample{}
	for _, proc := range allProcs {
		if _, ok := seenPID[proc.PID]; !ok {
			continue
		}
		item := proc.Process
		if item.User == "" {
			item.User = lookupUser(users, item.UID)
		}
		if prev, ok := e.prev[proc.PID]; ok && prev.start == proc.StartTicks && proc.Ticks >= prev.ticks {
			dt := now.Sub(prev.at).Seconds()
			if dt > 0 && clk > 0 {
				item.CPUPercent = float64(proc.Ticks-prev.ticks) / float64(clk) / dt * 100
				item.CPUKnown = true
			}
		}
		nextPrev[proc.PID] = cpuSample{ticks: proc.Ticks, start: proc.StartTicks, at: now}
		procOut = append(procOut, item)
	}
	e.prev = nextPrev
	byPID := map[int]*processes.Process{}
	for i := range procOut {
		byPID[procOut[i].PID] = &procOut[i]
	}
	for i := range list {
		if p, ok := byPID[list[i].PID]; ok && list[i].OwnerKnown {
			p.SocketKeys = append(p.SocketKeys, list[i].Key())
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].LocalPort != list[j].LocalPort {
			return list[i].LocalPort < list[j].LocalPort
		}
		if list[i].Protocol != list[j].Protocol {
			return list[i].Protocol < list[j].Protocol
		}
		return list[i].LocalIP.String() < list[j].LocalIP.String()
	})
	sort.Slice(procOut, func(i, j int) bool { return procOut[i].PID < procOut[j].PID })
	if note := ss.Note(bindKeys(list)); note != "" {
		warnings = append(warnings, note)
	}

	return Snapshot{
		Sockets:    list,
		Processes:  procOut,
		Interfaces: ifaces,
		Source:     source,
		Warnings:   warnings,
		ScannedAt:  now,
		Duration:   time.Since(started),
	}, nil
}

func readSockets() ([]ports.Socket, string, error) {
	list, err := nl.DumpSockets()
	if err == nil && len(list) > 0 {
		return list, "netlink", nil
	}
	fallback, ferr := readProcNet()
	if ferr != nil {
		if err != nil {
			return nil, "", fmt.Errorf("netlink: %v; procfs: %w", err, ferr)
		}
		return nil, "", ferr
	}
	if err == nil && len(list) == 0 && len(fallback) > 0 {
		return fallback, "procfs", nil
	}
	if err != nil {
		return fallback, "procfs", nil
	}
	return list, "netlink", nil
}

func readProcNet() ([]ports.Socket, error) {
	type spec struct {
		path, family, proto string
	}
	specs := []spec{
		{"/proc/net/tcp", ports.IPv4, ports.TCP},
		{"/proc/net/tcp6", ports.IPv6, ports.TCP},
		{"/proc/net/udp", ports.IPv4, ports.UDP},
		{"/proc/net/udp6", ports.IPv6, ports.UDP},
	}
	var out []ports.Socket
	var first error
	opened := 0
	for _, spec := range specs {
		f, err := openProc(spec.path)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		opened++
		batch, err := procfs.ParseProcNet(f, spec.family, spec.proto)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.path, err)
		}
		out = append(out, batch...)
	}
	if opened == 0 {
		return nil, first
	}
	return out, nil
}

func chooseOwner(cands []procfs.Proc, uid uint32) procfs.Proc {
	best := cands[0]
	for _, c := range cands[1:] {
		bestUID := best.UID == uid
		cUID := c.UID == uid
		if cUID && !bestUID {
			best = c
			continue
		}
		if cUID == bestUID && c.PID < best.PID {
			best = c
		}
	}
	return best
}

func lookupUser(cache map[uint32]string, uid uint32) string {
	if name, ok := cache[uid]; ok {
		return name
	}
	id := strconv.FormatUint(uint64(uid), 10)
	name := id
	if u, err := user.LookupId(id); err == nil && u.Username != "" {
		name = u.Username
	}
	cache[uid] = name
	return name
}

func bindKeys(list []ports.Socket) []string {
	var out []string
	for _, s := range list {
		if s.Protocol != ports.TCP || !s.IsBoundService() {
			continue
		}
		out = append(out, strings.ToLower(s.Protocol)+" "+s.LocalIP.String()+":"+strconv.Itoa(int(s.LocalPort)))
	}
	return out
}

func clockTicks() int64 {
	const atClktck = 17
	raw, err := os.ReadFile("/proc/self/auxv")
	if err != nil {
		return 100
	}
	for len(raw) >= 16 {
		key := binary.LittleEndian.Uint64(raw[0:8])
		val := binary.LittleEndian.Uint64(raw[8:16])
		if key == atClktck && val > 0 {
			return int64(val)
		}
		if key == 0 {
			break
		}
		raw = raw[16:]
	}
	return 100
}
