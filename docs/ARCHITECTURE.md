# Architecture

This document records what the code does today. `ARCHITECT.md` remains the product brief and is not edited by implementation work.

## Shape

The window is unprivileged. One privileged invocation of the same binary handles a single JSON operation and exits.

```text
Gio window
    |
    v
App store ---> SQLite (PortPilot records only)
    |             |
    |             +-- pkexec --> portpilot __privileged
    |                              |-- authorize (reads root procs & live firewall)
    |                              |-- ufw or nft (one backend)
    |                              |-- systemctl / systemd-run
    |                              +-- signal by pid and start ticks
    v
Socket engine, GeoIP engine, service list, container list
```

## Discovery

`SOCK_DIAG` is requested for TCP and UDP, IPv4 and IPv6, with the `INET_DIAG_INFO` extension so a TCP socket can carry `tcpi_bytes_sent` and `tcpi_bytes_received`. A failure, or an empty netlink result while `/proc` has rows, uses `/proc/net/tcp`, `tcp6`, `udp`, and `udp6`. The procfs fallback has no per-socket byte counters.

Ownership comes from `socket:[inode]` links under `/proc/<pid>/fd` and `/proc/<pid>/task/<tid>/fd` for multi-threaded processes. Live unprivileged scans continuously merge with privileged root process caches so that active outbound connections (browsers, IDEs, CLI tools) are immediately attributed without stale cache delays.

The Topology screen provides dual-mode views:
- **Process Hubs**: Groups sockets by process and port with directional traffic arrows, per-socket live packet/throughput drill-down monitor, and smart aggregation for kernel closing sockets.
- **World Geolocation**: Utilizes an embedded, fast offline GeoIP engine (`internal/core/network/geoip`) to map public IP destinations to national flags and country codes (e.g. 🇩🇪 Germany, 🇺🇸 United States, 🇫🇮 Finland, 🇳🇱 Netherlands, 🇮🇷 Iran), with one-click country filtering and a comprehensive live flow stream table.

Directories that return permission denied are counted and reported. Service names are the final `.service` path component of a cgroup. Container ids in cgroups are hints. The Containers screen separately lists Docker, LXC, and libvirt without changing their networks.

systemd units come from `systemctl list-units` without root, parsed dynamically to support modern outputs with or without the `LOAD` column. Actions go through the helper and only when pid 1 is systemd.

## Firewall

See `docs/FIREWALL.md`. The user database is not the kernel ruleset. Authorized reads replace the firewall view. Socket scans do not.

## Persistence

SQLite holds settings and `action_history`. A failed integrity check quarantines the database and opens a new one. A one-file backup is written to `portpilot.db.bak` before open when a database already exists.

Temporary firewall expiry lives in `/var/lib/portpilot/scheduled.json`, mode `0600`, written by root.

## Paths

- `$XDG_CONFIG_HOME/portpilot/config.json`
- `$XDG_DATA_HOME/portpilot/portpilot.db`
- `$XDG_CACHE_HOME/portpilot/portpilot.log`
- `$XDG_DATA_HOME/applications/portpilot.desktop` and `$XDG_DATA_HOME/icons/hicolor/.../portpilot.png` for the menu and window icon
- `/var/lib/portpilot/` for the helper schedule, last change, and audit log

Directories created by the user process are mode `0700`. The config and database are mode `0600`.
