# PortPilot

PortPilot is a Linux desktop tool for bound TCP and UDP sockets, the processes that own them, firewall rules, services, and container port listings. The window runs as a normal user. Firewall, service, and signal changes go through the same binary as `portpilot __privileged`, started with pkexec for one operation.

The module path is `github.com/0xamirreza/portpilot`.

The executable is dynamically linked. That is the intended form. Gio loads the host X11, Wayland, and EGL libraries, and a fully static file cannot open the window. It does not embed Node.js, a web server, or a database service. It needs glibc 2.34 or newer. Ubuntu 20.04 and Debian 11 are older than that and cannot start it.

Install the packages for your system before `make build`, `make test`, or the first launch. The table below is the set of systems this tree was actually run on.

## Systems this tree ran on

| System | glibc | What ran |
| --- | --- | --- |
| Manjaro, XFCE, X11 | 2.44 | `make test`, `doctor`, and the window |
| Ubuntu 22.04.5 LTS | 2.35 | the window and `doctor`, on the build machine |
| Debian 12 (bookworm) | 2.36 | `doctor`, with no display |
| Fedora Linux 41 | 2.40 | `doctor`, with no display |
| Arch Linux 20261101.0.620205 | 2.44 | `doctor`, with no display |
| openSUSE Leap 15.6 | 2.38 | `doctor`, with no display |

Debian, Fedora, Arch, and openSUSE ran the same `linux/amd64` binary that was built on Ubuntu 22.04. Those runs had no display, so the window was not opened there. Manjaro was built and opened on the desktop. Linux Mint was not run; it uses the Debian package names below.

Go on the Ubuntu build machine is 1.25.1. The Manjaro run used Go 1.27. Gio is 0.8.0.

## Before you build or run

Every system that compiles the window needs:

- Go 1.25 or newer
- a C compiler, `make`, and `pkg-config` (`pkgconf` on Arch and Manjaro)
- the X11, Wayland, EGL, and Vulkan libraries Gio links against, including `vulkan/vulkan.h`
- polkit (`pkexec`) before a firewall, service, or signal change

SQLite is pure Go and adds no package. A running X11 or Wayland session is required only for the window. `doctor` does not need a display.

After the packages are installed, this check must succeed before `make`:

```sh
pkg-config --exists x11 xkbcommon xkbcommon-x11 x11-xcb xcursor xfixes wayland-client wayland-cursor egl vulkan && echo "graphics modules found"
test -f /usr/include/vulkan/vulkan.h && echo "vulkan headers found"
go version
```

A machine that only runs a binary built elsewhere needs the runtime libraries and glibc 2.34 or newer. It does not need the Go toolchain or the `-dev` packages.

### Manjaro

Built and opened on XFCE with X11. Arch Linux uses the same package names. The Arch run in the table above was `doctor` only, from a binary built on Ubuntu, after the runtime libraries were present. `--needed` skips packages that are already installed.

Build and run:

```sh
sudo pacman -S --needed go gcc make pkgconf \
  libx11 libxcb libxcursor libxfixes \
  libxkbcommon libxkbcommon-x11 \
  wayland libglvnd vulkan-headers vulkan-icd-loader \
  polkit
```

`vulkan-icd-loader` provides `vulkan.pc` and does not provide `vulkan/vulkan.h`. Gio includes that header, so `vulkan-headers` is required even when `pkg-config --exists vulkan` already succeeds. The other libraries ship their headers in the same package. There is no separate `-dev` package.

Run a binary only, which is what the Arch `doctor` run did:

```sh
sudo pacman -S --needed libglvnd wayland libxkbcommon libxkbcommon-x11 \
  libx11 libxcursor libxfixes
```

`polkit` is the package that provides `pkexec` when a firewall, service, or signal change is needed. The Arch `doctor` run did not need it.

Optional, and only for the screen that uses them:

```sh
sudo pacman -S --needed nftables ufw docker lxc libvirt
```

The window and `doctor` run without those five. Opening Containers asks Docker, LXC, or libvirt for a list and does not change their networks.

### Ubuntu 22.04

This is the machine that built the `linux/amd64` binary and opened the window.

```sh
sudo apt install gcc make pkg-config \
  libx11-dev libx11-xcb-dev libxcursor-dev libxfixes-dev \
  libxkbcommon-dev libxkbcommon-x11-dev \
  libwayland-dev libegl1-mesa-dev libvulkan-dev \
  policykit-1
```

Go 1.25.1 was installed separately. The Ubuntu 22.04 `golang-go` package is older than 1.25. If `go version` is older, install a newer official Go toolchain and leave the C and graphics packages in place.

Runtime libraries, including the set used when a binary built on Ubuntu was copied elsewhere:

```sh
sudo apt install libegl1 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon0 libxkbcommon-x11-0 libx11-6 libx11-xcb1 \
  libxcursor1 libxfixes3
```

`policykit-1` provides `pkexec` for a firewall, service, or signal change.

### Debian 12

`doctor` ran here with no display, using a binary built on Ubuntu 22.04. Install the runtime libraries before starting that file:

```sh
sudo apt install libegl1 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon0 libxkbcommon-x11-0 libx11-6 libx11-xcb1 \
  libxcursor1 libxfixes3
```

`policykit-1` provides `pkexec` for a privileged action. To compile on Debian 12, use the Ubuntu build line above. Debian's `golang-go` may also be older than 1.25, so check `go version` and install a newer toolchain when it is.

### Fedora 41

`doctor` ran here with no display. Runtime packages:

```sh
sudo dnf install mesa-libEGL libwayland-egl libwayland-cursor \
  libxkbcommon libxkbcommon-x11 libX11 libX11-xcb libXcursor libXfixes
```

`polkit` provides `pkexec` for a privileged action. The Fedora `doctor` run did not need it.

To compile on Fedora:

```sh
sudo dnf install golang gcc make pkgconf-pkg-config \
  libX11-devel libXcursor-devel libXfixes-devel \
  libxkbcommon-devel libxkbcommon-x11-devel \
  wayland-devel mesa-libEGL-devel vulkan-loader-devel \
  polkit
```

Confirm `go version` is 1.25 or newer. `vulkan-loader-devel` provides `vulkan.pc` and pulls the Vulkan headers.

### openSUSE Leap 15.6

`doctor` ran here with no display. Runtime packages from that run:

```sh
sudo zypper install Mesa-libEGL1 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon0 libxkbcommon-x11-0 libX11-6 libX11-xcb1 \
  libXcursor1 libXfixes3
```

`polkit` provides `pkexec` for a privileged action. The Leap `doctor` run did not need it.

To compile, the devel names below match current openSUSE package indexes. On Leap, if one name does not resolve, install the package that provides the missing pkg-config module from the check at the top of this file.

```sh
sudo zypper install go gcc make pkgconf-pkg-config \
  libX11-devel libXcursor-devel libXfixes-devel \
  libxkbcommon-devel libxkbcommon-x11-devel \
  wayland-devel Mesa-libEGL-devel vulkan-devel \
  polkit
```

## Test

From the repository, after the packages for that system:

```sh
make test
make doctor
./bin/portpilot
```

`make test` is the Go test suite. It uses fixtures. It does not call UFW or nftables on the live host.

`make doctor` builds `bin/portpilot` and prints a read-only report. On a desktop session the useful lines are:

- `Host:` names the distribution
- `Init: systemd` on the systems in the table above
- `Display: present` when the command runs inside the graphical session
- `Procfs: /proc/net/tcp is readable`
- `pkexec: found` after polkit is installed
- `Linking:` says the executable is dynamically linked
- `Socket scan:` reports a count and either `netlink` or `procfs`
- `Firewall changes:` states that this command does not change rules

On the Manjaro desktop this report showed XFCE's X11 session, a netlink scan, and dynamic linking. The window then opened with `./bin/portpilot`.

The sidebar lists the live sockets. A wildcard address such as `0.0.0.0` or `::` means the socket is bound on every local address. It does not mean another machine can reach that port. A listening socket whose process cannot be read is incomplete data.

Topology and Diag are on the rail. Each row in Topology is one local process and port, directional flow arrows, and remote addresses, with separate traffic lanes for sent and received byte counters. Clicking any source card opens an interactive drill-down live stream monitor for that socket, tracking real-time throughput rate, payload, RTT, segments, and TCP state transitions. Closing or orphaned kernel sockets (`FIN-WAIT-1`, `FIN-WAIT-2`, `TIME-WAIT`, `LAST-ACK`, `CLOSE-WAIT`) are aggregated into a single unified `kernel (closing sockets)` card.

Diag is the internal diagnostic logging console (`Ctrl+0`). It streams application events in real time from an in-memory ring buffer, provides quick filters (`ALL`, `INFO`, `WARN`, `ERROR`, `DEBUG`), interactive search, log file rotation, and a full-detail view (`VIEW FULL →`) with structured attributes, raw logfmt, and selectable indented JSON.

Opening the window writes the menu entry and icon:

- `$XDG_DATA_HOME/applications/portpilot.desktop` (`~/.local/share/applications/` when that variable is unset)
- `$XDG_DATA_HOME/icons/hicolor/256x256/apps/portpilot.png`

`portpilot install-launcher` writes the same files. `make install` copies the binary to `~/.local/bin/portpilot` and then runs that command. The mark is the menu icon and the X11 window icon. It is not drawn inside the window. On XFCE, close the application menu and open it again if the new entry is not listed yet.

PolicyKit is separate. To let one admin prompt cover the session, run `pkexec ./bin/portpilot install-policy` after the binary exists. Until that policy is installed, each privileged action uses pkexec on its own.

If UFW is inactive and other nftables tables exist, the firewall page stays read-only. That is the safe result when firewall ownership is ambiguous. Allow, deny, and delete stay disabled there. Do not flush nftables to make the page writable.

## Run

```sh
make build
./bin/portpilot
./bin/portpilot doctor
```

`doctor` checks directories, the database, and a read-only socket scan. It does not change firewall rules and does not open the Docker socket.

## What this build does

- Native Gio window, dark by default, with a light plate. Rail screens are Overview, Ports, Processes, Firewall, Rules, Services, Containers, Activity, Topology, Diag, and Settings. Keyboard shortcuts `Ctrl+1` through `Ctrl+9` switch primary screens, `Ctrl+0` opens Diag, and `Ctrl+T` switches themes.
- Global Root Authorization Switch (`🔒 AUTHORIZE` / `🔓 ROOT (ACTIVE)`) in the top window header: one-click elevation via PolicyKit (`pkexec`) to resolve root process ownership, privileged file descriptors, and live firewall states across all screens, with instant one-click revocation.
- Enhanced full-width sidebar branding with crisp, scaled logo integration filling the navigation card.
- Continuous real-time Netlink streaming (sub-second 300ms cycle) with a steady `● LIVE (netlink)` header indicator.
- Live TCP and UDP discovery. Netlink `SOCK_DIAG` is preferred and includes TCP byte counters. `/proc/net` is the fallback.
- Multi-threaded process inode resolution traversing `/proc/<pid>/task/<tid>/fd` and seamless real-time merging of live user process scans with privileged root caches, eliminating `owner unknown` on active sockets.
- Topology screen with dual-mode navigation:
  - **Process Hubs**: Visual directional flow lanes, per-socket live packet/throughput drill-down monitor, and smart aggregation for kernel closing sockets (`FIN-WAIT-1`, `FIN-WAIT-2`, `TIME-WAIT`, `LAST-ACK`, `CLOSE-WAIT`).
  - **🌐 World Geolocation**: Interactive visual 3D Earth Globe (`🌍 3D GLOBE`) with orthographic projection, graticule grid, detailed continent landmasses, smooth auto-rotation, mouse-drag yaw/pitch rotation, and animated great-circle ballistic trajectory arcs from Local Host (Iran 🇮🇷) to remote destination countries, with an alternate flat map mode (`🗺️ 2D MAP`), country focus zooming, offline IP geolocation engine (`internal/core/network/geoip`), country breakdown pills (`DE`, `US`, `FI`, `NL`, `IR`, etc.), and a comprehensive live flow stream table.
- Overview screen with structured metric cards, clean interactive network interfaces table, and a dedicated `FIREWALL BACKEND` status plate (mode, active status, rule counts, temporary schedules, and boot persistence).
- Processes screen split-pane workstation with real-time search, category filters (`ALL`, `USER`, `ROOT`, `SERVICES`), interactive Process Inspector panel, and signal safety controls (`SIGTERM`, `SIGKILL`).
- Services screen with interactive search, status tabs (`ALL`, `RUNNING`, `INACTIVE`, `FAILED`), and robust systemd unit parser supporting distributions with or without the `LOAD` column.
- Diag diagnostic console with structured logfmt/slog streaming, severity filtering (`ALL`, `INFO`, `WARN`, `ERROR`, `DEBUG`), interactive stream pause (`PAUSE` / `RESUME`), searchable in-memory ring buffer, on-disk rotation, and drill-down detail view (`VIEW FULL →`) with structured attributes, raw lines, and formatted JSON.
- Configurable log retention policy in Settings (1 to 365 days, defaulting to 7 days, with quick presets `1d`, `3d`, `7d`, `14d`, `30d`), plus semantic versioning display (`v0.5.0`).
- Native AppImage packaging pipeline (`make appimage`), providing seamless cross-Linux portability without package manager friction.
- Transparent PolicyKit (`pkexec`) elevation from within AppImage mounts via `--appimage-extract-and-run` helper execution, preventing FUSE user-mount permission blocks.
- Process, user, executable, systemd service, and cgroup container hints where `/proc` allows them.
- Confirmed allow, deny, reject, delete, rollback, and temporary rules through UFW or the `inet portpilot` nftables table, when that backend is unambiguous.
- systemd service list, and start, stop, restart, enable, and disable after confirmation.
- SIGTERM and SIGKILL after a start-time check. SIGKILL needs a second confirmation.
- Docker, LXC, and libvirt listings after the Containers screen is opened. Those listings do not change container networks.
- SQLite audit of application actions, plus a root audit file when the helper runs.
- Scrollbars on the lists that extend past the page.

## Distribution & Packaging

PortPilot has standardized on **AppImage** as its primary Linux desktop distribution format. Instead of maintaining disparate distro-specific packages (`.deb`, `.rpm`, AUR), a single self-contained AppImage runs across modern Linux distributions (Ubuntu, Debian, Fedora, Arch/Manjaro, openSUSE).

To build the universal AppDir and AppImage:

```sh
make appimage
```

This compiles the binary, extracts embedded high-resolution brand icons, generates the desktop entry, configures `AppRun`, and packages the output into `dist/PortPilot-x86_64.AppImage`.

## Roadmap

### Phase 1: Universal Linux AppImage & Core Refinement (v0.5.0 — Current)
- [x] **Primary Distribution**: Transition focus to cross-Linux **AppImage** distribution, deprecating standalone raw binary / `.deb` prerequisites.
- [x] **Privileged Helper in AppImage**: Automatic detection of `$APPIMAGE` and privileged root helper invocation via `pkexec "$APPIMAGE" --appimage-extract-and-run __privileged` to bypass FUSE mount restrictions.
- [x] **Desktop Integration**: Launcher installation (`portpilot install-launcher`) automatically pointing to `$APPIMAGE` when run inside an AppImage container.
- [x] **3D Earth Globe & 2D World Map Topology**: Interactive live connection visualization with great-circle trajectory arcs and offline GeoIP mapping.
- [x] **Sub-second Netlink Discovery**: Continuous real-time socket and byte counters streaming with zero UI jitter.
- [x] **Diagnostic Logging Console (Diag)**: Live structured slog streaming, in-memory ring buffer, and formatted JSON inspector.

### Phase 2: Release Automation & Advanced Linux Observability (v0.6.0)
- [ ] **Automated GitHub Releases**: GitHub Actions CI/CD matrix compiling and publishing signed `PortPilot-x86_64.AppImage` and SHA256 checksums on tag creation.
- [ ] **Container Network Namespaces**: Deep socket inspection across Docker, Podman, and Kubernetes CNI network namespaces (`/var/run/netns`).
- [ ] **Extended BPF Socket Tracing**: Optional eBPF probes for sub-millisecond connection latency, retransmissions, and TCP backlog drops.
- [ ] **Firewall Preset Profiles**: Quick one-click firewall hardening profiles (Workstation, Public Wi-Fi, Dev Server).

### Phase 3: Platform Abstraction Layer (PAL) & Cross-Platform Exploration (v1.0.0)
- [ ] **Core Engine Decoupling**: Isolate discovery, firewall, and process inspection into an OS-agnostic engine interface (`internal/core/engine`).
- [ ] **Windows Native Backend (Exploration)**:
  - Windows Filtering Platform (WFP) and Win32 IP Helper API (`GetExtendedTcpTable`).
  - Windows UAC elevation service.
  - Distribution via standard Windows Installer (`.msi`).
- [ ] **macOS Native Backend (Exploration)**:
  - BSD `libproc` socket inspection and Packet Filter (`pfctl`) integration.
  - Authorization Services elevation (`authopen`).
  - Distribution via Apple Disk Image (`.dmg`).

## Limits

- Live UFW and nftables reads need root. An unprivileged `ufw status` or `nft list` is denied, so the window shows that until you authorize a read.
- If UFW is active, only UFW is used. Other nftables tables or iptables-legacy rules beside another manager keep the host read-only.
- The nftables chain policy is accept. PortPilot adds explicit rules. It does not flush tables.
- Temporary rules are removed by a systemd timer that runs the helper. If that timer cannot be created, the new rule is deleted and the window says expiration was not guaranteed.
- OpenRC and SysV units can be listed when that init is running. Actions are sent only when pid 1 is systemd.
- The optional PolicyKit file is installed only by `portpilot install-policy`.
- `make release` writes `linux/amd64` and attempts `linux/arm64`. A missing arm64 toolchain leaves that file out of `dist/SHA256SUMS`. `linux/arm64` was not part of the runs in the table above.

Build details are in `docs/BUILD.md`.

The Liberation fonts embedded in the binary stay under the SIL Open Font License. The text is in `assets/fonts/LICENSE-OFL.txt`.
