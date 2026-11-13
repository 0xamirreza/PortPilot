# Changelog

All notable changes to PortPilot are documented in this file.

## [v0.5.0] - 2026-11-14

### Added
- **Universal Linux AppImage Packaging**:
  - Full AppImage bundling pipeline with `make appimage` and `scripts/build-appimage.sh`.
  - Automatic `APPIMAGE` environment detection for transparent privilege elevation via `pkexec "$APPIMAGE" --appimage-extract-and-run __privileged`.
  - Desktop integration installer updating `.desktop` launcher to reference the host `.AppImage` path.
- **Interactive 3D Earth Globe & 2D World Map Topology**:
  - Full orthographic 3D Earth Globe (`internal/gui/globe.go`) with graticule lat/long grid, detailed continent landmass vectors (`internal/gui/map_data.go`), smooth continuous auto-rotation, and interactive mouse-drag yaw/pitch rotation.
  - Animated great-circle ballistic trajectory arcs visualizing outbound network traffic streams from Local Host to remote destination countries with animated traveling particle pulses.
  - Alternate flat projection mode (`🗺️ 2D MAP`) with country focus zooming and click-to-center navigation.
  - High-performance vector icons engine (`internal/gui/icons.go`) rendering crisp resolution-independent UI icons across all screen scales.
  - Comprehensive country coordinate mapping and spatial boundary calculations (`internal/core/network/geoip/coordinates.go`).

## [v1.2.0] - 2026-11-08

### Added
- **Diagnostic Logging Screen (`Diag`)**:
  - Added the 10th rail screen: `Diag` (`Ctrl+0`).
  - High-performance in-memory ring buffer capturing all application events formatted according to logfmt and structured `log/slog` standards.
  - Multi-level filtering: `ALL`, `INFO`, `WARN`, `ERROR`, and `DEBUG`.
  - Real-time statistics summary bar displaying total events, per-level counts, active log file path, and live disk size.
  - Interactive stream pause: added `PAUSE` / `RESUME` action button to freeze the diagnostic log view in-place for calm inspection and selection without jumping.
  - Immediate log rotation (`ROTATE` button) and memory buffer clearing (`CLEAR`).
  - Full-detail log inspection (`VIEW FULL →`): dedicated drill-down view (`diagDetailView`) presenting complete un-truncated timestamps, event message, severity badge, structured attribute key-value cards, selectable raw log line, and formatted/indented JSON view.
- **Log Retention & Rotation Policy**:
  - Configurable retention period in `Settings` (1 to 365 days, defaulting to 7 days).
  - Quick-select preset buttons (`1d`, `3d`, `7d`, `14d`, `30d`) with on-disk pruning of rotated log files (`portpilot.log.YYYYMMDD`).
- **Real-Time Live Socket Traffic Monitor in Topology**:
  - Clicking any Topology source card opens an interactive drill-down monitor tracking live packet and byte movements in real time.
  - Real-time throughput metrics (TX Rate B/s, RX Rate B/s, total bytes sent/received, active peers count).
  - Live socket event log tracking delta byte transfers, segment counters, round-trip time (RTT in µs/ms), and TCP state transitions.
- **Kernel Socket Teardown Aggregation**:
  - Automatically identifies closing and orphaned TCP sockets (`FIN-WAIT-1`, `FIN-WAIT-2`, `TIME-WAIT`, `LAST-ACK`, `CLOSE-WAIT`) in the Linux kernel network stack.
  - Groups unattributed closing sockets into a single unified `kernel (closing sockets)` hub in Topology instead of cluttering the view with dozens of duplicate `owner unknown` cards.
- **World Geolocation & Outbound Flows Topology Tab (`🌐 WORLD GEOLOCATION`)**:
  - Added dedicated global geolocation tab in the Topology screen.
  - Interactive offline IP geolocation engine (`internal/core/network/geoip`) resolving public IPv4/IPv6 destination addresses to country codes, country names, and national flags (e.g. 🇩🇪 Germany, 🇺🇸 United States, 🇫🇮 Finland, 🇳🇱 Netherlands, 🇮🇷 Iran, 🇫🇷 France, 🇬🇧 UK).
  - Global overview dashboard displaying local machine origin (`SOURCE: LOCAL HOST (IP) · 🇮🇷 IRAN`), active outbound flow counters, unique country counts, and total external traffic (▲ Sent / ▼ Received).
  - One-click interactive country filter pills (`ALL COUNTRIES`, `🇩🇪 Germany (N)`, `🇺🇸 United States (N)`, etc.) for instant destination filtering.
  - Comprehensive live flows stream table listing every active connection: Local process name and PID, Local IP and port, Directional animated transfer arrow with delta bytes, Remote destination IP and port with well-known protocol tags (`HTTPS`, `DNS`, `SSH`, etc.), and highlighted Target Country badge.
- **Enhanced Full-Width Sidebar Logo Branding**:
  - Expanded the sidebar logo (`ui.mark`) from a fixed 80dp box to dynamically fill the entire 152dp usable width of the sidebar card, eliminating awkward dead whitespace and creating a commanding, high-definition visual brand presence.
- **High-Definition Branding Assets**:
  - Replaced legacy graphics with the updated high-resolution logo from `logo.jpg`.
  - Exported multi-resolution crisp transparent icons (16x16 through 512x512) for X11 `_NET_WM_ICON` dynamic hints and system desktop menu launchers.
  - Enhanced sidebar menu branding with increased padding and crisp image scaling.
- **Global Root Authorization Header Switch (`🔒 AUTHORIZE` / `🔓 ROOT`)**:
  - Added a dedicated authorization button in the top window header beside the Live Netlink indicator.
  - One-click privilege elevation via PolicyKit (`pkexec`): when authorized, reads root process file descriptors and full live firewall rules.
  - Automatically resolves all system/root process names (such as `mariadbd`, `containerd`, `cupsd`, `systemd-resolve`, `nxd`, and `dnsmasq`) across all screens (Ports, Processes, Overview, Topology, and Firewall).
  - One-click lock toggle: clicking the unlocked button revokes authorization, returning the session to unprivileged mode with zero lingering elevated cache.
- **Application Versioning in Settings**:
  - Integrated version identification (`v1.2.0`) in `internal/version/` and displayed it cleanly in the Settings screen.

### Changed
- **Firewall & Rules Screens Complete Redesign**:
  - Removed redundant `AUTHORIZE READ` buttons from both Firewall and Rules screens, unifying root privileges under the global header authorization switch.
  - Redesigned `Firewall` overview: added backend status indicator (`UFW (ACTIVE)`, `NFTABLES`), detailed reason banner, live metrics bar (`ACTIVE RULES`, `SCHEDULES`, `UFW ACTIVE`, `UFW ON BOOT`, `NFT TABLES`), and action controls (`RECONCILE`, `ROLLBACK`).
  - Redesigned `Rules` editor: eliminated the cluttered single-column vertical button/input stack, grouping rule creation into a compact 4-row form (Port/Protocol/Family, Source/Interface, Duration presets, and color-coded action buttons `ALLOW`, `DENY`, `REJECT`).
  - Added dedicated structured table for active rules with columns: `BACKEND`, `ACTION` (colored pills: `ALLOW` green, `DENY` red, `REJECT` amber), `PROTOCOL`, `PORT`, `SOURCE IP / CIDR`, and `SCOPE / OWNERSHIP`.
  - Added visual countdown pills for temporary schedules and an organized bottom section for disabled rules.
- **Processes Screen Complete Redesign**:
  - Transformed the `Processes` screen into a split-pane workstation with real-time search, category filters (`ALL`, `USER`, `ROOT`, `SERVICES`), and live statistics counters (`TOTAL`, `USER`, `ROOT`, `SERVICES`).
  - Structured process table columns: `PID` (70dp), `PROCESS` (150dp), `USER` (75dp), `MEMORY` (80dp), `CPU` (55dp), and `SOCKETS` badge. Fixed PID and process text truncation.
  - Dedicated interactive Process Inspector panel on the right with formatted property cards: System Service, Container runtime, Network Namespace, Memory RSS, CPU usage, Start ticks, Executable path, Command line, and held Socket Descriptors.
  - Distinct signal controls with safety confirmation: polite termination (`SIGTERM`) and danger-highlighted force kill (`SIGKILL`), plus integrated Systemd service management (`STOP SERVICE`, `RESTART SERVICE`).
- **Services Screen Complete Redesign & Parser Fix**:
  - Full restructuring of the `Services` screen with an interactive search bar, status filter tabs (`ALL`, `RUNNING`, `INACTIVE`, `FAILED`), and live status counters.
  - Fixed systemd unit parsing for modern Linux distributions (such as Arch/Manjaro systemd 250+) where `systemctl list-units --plain` omits the `LOAD` column or prepends bullet characters (`●`, `○`), ensuring all running services are correctly recognized and counted instead of showing `RUNNING: 0`.
  - Added robust service status matching helpers (`isServiceRunning`, `isServiceFailed`, `isServiceExited`).
  - Dedicated service row layout featuring colored status badges (`RUNNING`, `EXITED`, `FAILED`, `INACTIVE`), bold monospace unit names, sub-state indicators, and clear service descriptions.
  - Interactive selection bar with dedicated actions (`START`, `STOP`, `RESTART`, `ENABLE`, `DISABLE`).
- **Network Interfaces Overview & Firewall Card Redesign**:
  - Redesigned the `INTERFACES` section in the `Overview` screen from a raw text dump into an interactive, structured table.
  - Added header statistics bar with real-time counters: `TOTAL`, `UP` (green), and `DOWN` (gray).
  - Categorized interface types with dedicated visual badges: `WI-FI`, `ETHERNET`, `DOCKER / VETH`, `VM BRIDGE`, `LXC BRIDGE`, `VPN / TUNNEL`, and `LOOPBACK`.
  - Transformed the unstyled raw Firewall status line in `Overview` into a dedicated structured card (`firewallOverviewCard`) featuring `FIREWALL BACKEND`, `STATUS` (active/inactive badge), `ACTIVE RULES`, `TEMPORARY RULES`, and `BOOT ENABLED` pill.
  - Removed obsolete and disruptive warning banners ("A wildcard bind is not proof that another machine can reach the port" and "ss does not list every TCP listen...") from Overview.
  - Structured IP address display clearly separating IPv4 (bold) and IPv6 (muted) addresses, eliminating text truncation and messy whitespace.
  - Replaced the arbitrary 12-item limit with a smooth, vertically scrollable list container.
- **Continuous Real-Time Streaming**:
  - Upgraded socket discovery from periodic polling to continuous real-time Netlink streaming (300ms cycle).
  - Replaced the legacy refresh interval configuration with an active live streaming status indicator in Settings.
  - Removed the redundant `RESCAN` button from the top-right header and stabilized the status label to a constant `● LIVE (netlink)`.
- **Topology Visual Hierarchy**:
  - Separated sent and received traffic metrics onto distinct visual lanes above and below directional flow arrows, eliminating text overlap and visual clutter.
- **Keyboard Shortcuts Alignment**:
  - Aligned and verified all desktop shortcut keys (`Ctrl+1` through `Ctrl+9` for primary screens, `Ctrl+0` for Diag, `Ctrl+T` for Theme toggle, and `Ctrl+F` for Search).

### Fixed
- Fixed process inode resolution and eliminated `owner unknown` on active sockets: resolved a critical caching defect where privileged process snapshots (`privProcs`) caused continuous live scans of user processes (`procs`) to be discarded. Live scans now merge seamlessly with privileged data so new outbound connections (e.g. browser, IDE, CLI) immediately attach to their actual process.
- Multi-threaded process inode discovery: added traversal of `/proc/[pid]/task/[tid]/fd` to capture unshared socket descriptors created by worker threads.
- Topology grouping and attribution: separated unattributed sockets by user and type, eliminated false aggregation of unrelated destination flows under a single ephemeral port, and attributed known user/system daemon flows appropriately.
- Fixed root process inode resolution: prioritized privileged process entries over unprivileged permission-denied process stubs, properly attaching root ownership (like NetworkManager, MariaDB, and systemd) to scanned sockets.
- Fixed `ACTION` column alignment in Diag table: pinned `VIEW FULL →` button to the far right edge (95dp fixed container) across all rows regardless of whether structured attributes are present.
- Fixed visual distortion in the Diag log table where vertical rows stretched as full-height columns due to uninitialized list axis orientation.
- Fixed RTT formatting and traffic calculation delta baselines during initial live monitor sampling.

---

## Systems & Test Verification on Manjaro Linux

- **Host Environment**: Manjaro Linux (Kernel Linux 6.x, glibc 2.44, XFCE Desktop, X11 session).
- **Toolchain**: Go 1.25+, GCC, `make`, `pkgconf`.
- **Automated Test Suite**:
  - Executed `make test` across all packages:
    - `internal/config`: validated configuration loading, JSON serialization, and log retention constraints.
    - `internal/core/ports`: validated socket filtering, peer association, and closing socket aggregation.
    - `internal/gui`: validated UI state transitions, live stream tracking, and log detail selection.
    - `internal/logging`: validated ring buffer storage, level filtering, and log rotation pruning.
    - `internal/discovery` (netlink, procfs, ss, sockets, containers, services): verified read-only socket discovery and process ownership mappings.
    - `internal/firewall` (ufw, nftables): verified safety rules and non-destructive namespace isolation.
  - Result: **All tests passed cleanly (`make test` exit code 0)**.
- **Compilation & Linking**:
  - `make build` succeeded with dynamic cgo linking against system X11, Wayland, and EGL libraries, generating `bin/portpilot`.
- **Compatibility Doctor Check**:
  - `./bin/portpilot doctor` ran successfully on Manjaro Linux:
    - Host distribution detected: `Manjaro Linux`.
    - Netlink `SOCK_DIAG` verified and operational.
    - Procfs `/proc/net/tcp` readable.
    - Firewall binaries verified: `ufw` and `nftables` detected.
    - Service managers: `systemctl` detected.
    - Privilege escalation: `pkexec` and `sudo` available.
    - Container engines: `docker`, `lxc`, and `virsh` tools detected.
    - Embedded SQLite database opened and initialized at `$XDG_DATA_HOME/portpilot/portpilot.db`.
