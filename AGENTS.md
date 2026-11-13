# Agents

PortPilot is a Linux administration tool. Treat `ARCHITECT.md` as the unchanged product brief. Living engineering notes belong in `docs/`.

## Current build

The window, socket discovery, firewall read and confirmed changes, temporary rules, systemd actions, process signals, read-only container listings, and the Topology screen are implemented. The logo is the menu and X11 window icon. Tests must not execute UFW or nftables against the live host. The README lists the systems this tree ran on and the packages for each.

## Rules that stay in force

- The GUI process stays unprivileged. Root may run only `__privileged`.
- No shell interpolation and no arbitrary commands.
- UFW, when it is active, is the only writer.
- Never flush nftables. Never modify tables other than `inet portpilot`.
- Ambiguous firewall ownership stays read-only.
- Do not run destructive firewall tests on the developer machine.
- Do not present placeholder data as a live scan.
- A listening socket without a readable process is incomplete data, not an incident.
- A wildcard bind is not external reachability.
- Do not open Docker, LXC, or libvirt until the Containers screen asks for a list.

## Module path

`github.com/0xamirreza/portpilot` is the module path.

## Checks

```sh
make test
make build
./bin/portpilot doctor
```

Report implemented behavior, files, dependencies, tests, limitations, and what remains.
