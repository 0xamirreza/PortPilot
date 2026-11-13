For PortPilot, the architecture is designed so that the final product is a native Linux desktop application packaged as a self-contained executable (Single Binary / AppImage); not a project requiring Node.js, Python, Docker, or a separately installed backend runtime.

A key technical architectural decision: to achieve maximum compatibility across Linux distributions, both the core engine and graphical user interface are built with Go, avoiding any dependency on WebView or WebKitGTK.

# PortPilot — Master Architecture

[Portmaster: A GlassWire Alternative for Linux to Monitor & Secure Network Connections](https://images.openai.com/static-rsc-4/OUy_Z4zw7KiolPcBM-W-gy3mB8FSfF0zK3j6qb8ytfxtRk2n6f5xJ_BI4XhbeT2OfMeOVAQlGEc43sJ941qV1uO5FikgHcxQN1gwP7XiL4esDEgOab6hnAde0ZS4Mhr2WYax8e6Yk4ROjU9jxsp3lu-vqkq8rYQt1vF-Xk-Rjf4?purpose=inline)

[itsfoss.com](https://itsfoss.com/portmaster/?utm_source=chatgpt.com)

# PortPilot

Linux Port & Firewall Management Platform

Single Binary

Native Desktop

Go

Cross-Distro

UFW + nftables

Goal: Discover, monitor, and manage ports, processes, services, and firewall rules in Linux through a unified graphical user interface.

## 1. Architectural Decisions

| Component                | Selection                            |
| ------------------------ | ------------------------------------ |
| Language                 | Go                                   |
| Desktop GUI              | Gio (Immediate Mode GUI)             |
| Application Architecture | Modular Monolith                     |
| Binary                   | Single Executable                    |
| Database                 | SQLite                               |
| SQLite Driver            | Pure Go (modernc.org/sqlite)         |
| Process Discovery        | Linux `/proc` + Netlink              |
| Socket Discovery         | Netlink + `/proc` fallback           |
| Firewall                 | UFW + Native nftables                |
| Privilege Management     | Polkit / pkexec + sudo fallback      |
| Service Management       | systemd + SysV/OpenRC detection      |
| Container Integration    | Docker Socket API (Optional)         |
| Configuration            | XDG Base Directory                   |
| Logging                  | Structured Logging                   |
| Testing                  | Go Test + Integration Tests          |
| Build                    | Go Cross-Compilation where supported |

### Why Gio?

Consider three common desktop GUI options:

| Framework | Advantages                           | Limitations                          |
| --------- | ------------------------------------ | ------------------------------------ |
| Gio       | Native Go rendering, zero WebView    | Smaller UI ecosystem                 |
| Wails     | Modern UI with Vue/Quasar            | Heavy WebKitGTK dependency on Linux  |
| Fyne      | Easy native GUI development          | High system graphics dependencies    |

Gio is chosen because our primary priority is universal portability and standalone execution across diverse Linux distributions.

Note regarding Single Binary / AppImage: A standalone executable does not eliminate OS graphics infrastructure. The GUI depends on X11/Wayland and GPU drivers; firewall management depends on host kernel capabilities and tools. Thus, the goal is a standalone binary with zero external runtime requirements (no Node.js/Electron/Python), rather than an impossible binary decoupled from kernel and display servers.

## 2. High-Level Architecture

PortPilot is a Modular Monolith: a single executable with clean, independent internal packages rather than multiple microservices.

PortPilot Desktop

## Gio GUI

Dashboard · Ports · Processes · Firewall · Services

Application Core

Use Cases · Validation · Authorization · State Management

Discovery

Ports / Process / Network

Firewall

UFW / nftables

System

systemd / Services

Integrations

Docker / LXC / KVM

System Adapter Layer

Netlink · /proc · UFW CLI · nftables Netlink · systemd D-Bus

Linux Kernel & Host Services

TCP/UDP · Firewall · Network Interfaces · Processes

### Project Structure

```
portpilot/
├── cmd/
│   └── portpilot/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── lifecycle.go
│   │   └── state.go
│   │
│   ├── gui/
│   │   ├── window.go
│   │   ├── navigation.go
│   │   ├── theme/
│   │   ├── components/
│   │   └── screens/
│   │       ├── dashboard/
│   │       ├── ports/
│   │       ├── processes/
│   │       ├── firewall/
│   │       ├── services/
│   │       ├── containers/
│   │       ├── activity/
│   │       └── settings/
│   │
│   ├── core/
│   │   ├── ports/
│   │   ├── processes/
│   │   ├── firewall/
│   │   ├── services/
│   │   ├── network/
│   │   └── events/
│   │
│   ├── discovery/
│   │   ├── sockets/
│   │   ├── procfs/
│   │   ├── netlink/
│   │   ├── interfaces/
│   │   └── containers/
│   │
│   ├── firewall/
│   │   ├── manager.go
│   │   ├── detector.go
│   │   ├── ufw/
│   │   ├── nftables/
│   │   ├── rules/
│   │   └── scheduler/
│   │
│   ├── system/
│   │   ├── privilege/
│   │   ├── polkit/
│   │   ├── services/
│   │   ├── distro/
│   │   └── capabilities/
│   │
│   ├── integrations/
│   │   ├── docker/
│   │   ├── lxc/
│   │   └── libvirt/
│   │
│   ├── storage/
│   │   ├── sqlite/
│   │   ├── migrations/
│   │   └── repositories/
│   │
│   ├── config/
│   ├── logging/
│   └── security/
│
├── assets/
│   ├── icons/
│   └── fonts/
│
├── docs/
│   ├── ARCHITECTURE.md
│   ├── SECURITY.md
│   ├── FIREWALL.md
│   ├── BUILD.md
│   └── CHANGELOG.md
│
├── scripts/
│   ├── build.sh
│   └── release.sh
│
├── tests/
│   ├── integration/
│   └── fixtures/
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE
└── AGENTS.md
```

This is the baseline structure; modules should maintain strict separation of concerns.

## 3. Core Module Design

### A. Dashboard

The main screen presents the following telemetry:

- Total TCP Listening Sockets
- Total UDP Sockets
- Total active processes with open sockets
- Ports bound to `0.0.0.0`
- Ports bound to `::`
- Localhost-only bound ports
- Firewall status
- UFW and nftables active state
- Active firewall rules count
- Security warnings and reachability indicators

### B. Port Discovery

The foundational module of the application.

Gathers the following attributes for every socket:

| Field      | Description                      |
| ---------- | -------------------------------- |
| Protocol   | TCP / UDP                        |
| Address    | Bind address                     |
| Port       | Port number                      |
| Family     | IPv4 / IPv6                      |
| State      | LISTEN / UNCONN / ESTABLISHED    |
| PID        | Process identifier               |
| Process    | Executable name                  |
| Executable | Full binary path                 |
| User       | Process owner                    |
| Service    | Associated systemd unit          |
| Namespace  | Network Namespace                |
| Container  | Container ID (if containerized)  |

Essential requirements:

- Support both TCP and UDP.
- Differentiate IPv4 and IPv6.
- Never conflate Loopback with Public/Wildcard binds.
- Preserve distinct sockets sharing ports.
- Handle processes holding multiple sockets.
- Filter out stale PIDs from terminated processes.
- Clearly annotate incomplete data or Permission Denied results.
- Unattributed sockets must be treated as incomplete data, not security incidents.
- Bind addresses must not be conflated with firewall accessibility.

Netlink SOCK_DIAG is the primary discovery mechanism, with `/proc/net` as fallback. `ss` is used strictly as a diagnostic cross-check.

### C. Firewall Manager

Two independent backends:

UFW Backend:
- Detect UFW installation and active status
- Display active rules
- Add rules
- Delete rules
- Enable / disable rules
- Manage IPv4 and IPv6 rules
- Support TCP and UDP
- Filter by source IP / CIDR and interface
- Support rule comments

Native nftables Backend:
- Read active tables and chains
- Manage PortPilot-owned rules
- Maintain dedicated table `inet portpilot`
- Create structured chains and rules
- Delete PortPilot-owned rules
- Utilize comments and rule handles
- Never modify or flush third-party tables

Critical rule:
If UFW is active, PortPilot must not alter host rules through the independent nftables backend, even if UFW utilizes nftables under the hood. Only the UFW backend is permitted to modify rules in this state.

If both systems are active and rule ownership is ambiguous, PortPilot must remain in Read-Only mode and alert the user.

### D. Temporary Firewall Rules

Enables short-lived firewall rules with reliable expiration.

Preset durations:
- 5 minutes
- 15 minutes
- 30 minutes
- 1 hour
- 6 hours
- 24 hours
- Custom duration

Expected behavior:

1. User selects target port and protocol.
2. Selects source restriction and rule scope.
3. Specifies expiration duration.
4. Rule is applied atomically.
5. Remaining countdown is displayed in the GUI.
6. Rule is automatically purged upon expiration.
7. Lifecycle event is logged in the Activity Log.

For nftables, leverage timeout support in sets or equivalent kernel mechanisms.
For UFW (lacking native timeout support), use an independent, persistent systemd timer. On systems without systemd timers, transparently state this limitation and never claim a rule will automatically expire after the window closes.

### E. Process Manager

Features:

- Process listing and search
- Inspect PID and PPID
- Inspect owner user
- Inspect executable path and command line
- List all open sockets owned by the process
- Identify associated systemd unit
- Monitor CPU and RAM consumption when accessible
- Send SIGTERM
- Send SIGKILL with explicit confirmation
- Stop / Restart associated systemd service

Note: Terminating a process and stopping a systemd service are distinct operations and must remain separate.

### F. Service Manager

Supported init systems:

- systemd
- OpenRC
- SysV init

Automatically detect the active init system and enable only supported actions.

Features:

- Start
- Stop
- Restart
- Enable
- Disable
- Status telemetry
- Map services to listening ports

### G. Container & Virtualization

Support for Docker, LXC, and KVM/libvirt:

- Docker: inspect port mappings via Docker Socket API
- LXC: identify containers and network interfaces
- KVM/libvirt: identify bridge networks and virtual switches
- Map host ports to container destinations
- Distinguish host-native sockets from containerized services

Docker access remains Read-Only by default and is queried only on demand.

### H. Activity Log

Every modification is permanently audited:

```
2026-11-08 12:00:00
Action: FIREWALL_BLOCK
Backend: UFW
Protocol: TCP
Port: 5900
Duration: 30m
Result: SUCCESS
```

Filterable by:

- Timestamp
- Process
- Port
- Action
- Backend
- Result

## 4. Security Model & Privilege Management

Privilege separation is paramount.

Never execute the entire application as root.

PortPilot enforces two distinct privilege levels:

1. User Mode: unprivileged GUI execution and data visualization.
2. Privileged Helper: minimal root helper executing sensitive changes.

Architecture:

```
PortPilot (Normal User)
        |
        | Privileged Request
        v
Polkit / pkexec
        |
        v
PortPilot Helper
        |
        +-- Firewall Rules
        +-- Service Control
        +-- Process Signals
        +-- Network Configuration
```

The helper utilizes the same binary via an internal subcommand:

```
portpilot __privileged
```

Operates strictly through structured JSON messages over stdin/stdout.

Security rules:

- No execution of arbitrary shell commands.
- No shell interpolation.
- Strict argument allowlists.
- Robust input validation.
- Audit all sensitive operations.
- Privileged operations must remain auditable.
- The GUI process never maintains persistent root access.
- Fallback to sudo when Polkit is unavailable.
- Deny operations clearly when authorization fails.

## 5. User Interface Design

The interface must be modern, minimal, and tailored for system administration.

[Portmaster: A GlassWire Alternative for Linux to Monitor & Secure Network Connections](https://images.openai.com/static-rsc-4/ogml2LY9kNTYndO7CoXvWRbLJcAdbdTMIh4YgUd6mL_8kR-4tsoHBOW_TRb6fFwNNhjdrRKgomtQVsItVprFBU5K7Xoswc2AaVghwsiKQd5giA79mOt1yxOqkrFDdLuU8gpAXH1eqGmTeP0bWjByyR4AatzxOHC3vYQY74P53WM?purpose=inline)

[itsfoss.com](https://itsfoss.com/portmaster/?utm_source=chatgpt.com)

Visual direction for Dashboard

[Portmaster: Uma alternativa ao Glasswire para monitorar e proteger conexões de rede • é Livre e Aberto!](https://images.openai.com/static-rsc-4/A5DmGsmPcu7bU4g0jyeUF4aREpZEbDWklx-M3Nrk6InSx3du97fv-T1-29-xcOVQrvVql6InTXh1HqfxePMMf5oppeYgOjmGNQg6uJCIIDM93IF-TSdR2fc3VEtvZ8X5ltWqwB7W6ZHIJuZNy8wcpijH_X87xIQOOWMdlIW5F9k?purpose=inline)

[livreeaberto.com](https://livreeaberto.com/portmaster?utm_source=chatgpt.com)

Visual direction for Firewall and Port Management

Main navigation:

| Screen     | Responsibility                     |
| ---------- | ---------------------------------- |
| Overview   | Global system telemetry            |
| Ports      | Socket and port discovery          |
| Processes  | Process manager and signals        |
| Firewall   | UFW and nftables overview          |
| Rules      | Firewall rules editor              |
| Services   | systemd unit management            |
| Containers | Docker / LXC / KVM mappings        |
| Activity   | Audit log history                  |
| Settings   | Application preferences and theme  |

UX Features:

- Dark and Light themes
- Instant search
- TCP / UDP filters
- IPv4 / IPv6 filters
- Localhost vs All Interfaces filters
- Process-based filtering
- Real-time continuous refresh
- Manual refresh trigger
- Detailed side panel inspection
- Confirmation dialogs for dangerous actions
- State diff display before and after changes
- Responsive design for varying display sizes
- Keyboard navigation accessibility

## 6. Storage & Database

SQLite stores PortPilot application data only; it never mirrors kernel network state.

Tables:

```
settings
firewall_rules
scheduled_rules
action_history
process_snapshots
network_interfaces
application_state
schema_migrations
```

Storage paths:

```
~/.config/portpilot/config.json
~/.local/share/portpilot/portpilot.db
~/.cache/portpilot/
```

Files writable by normal users must never be trusted by the root helper.

## 7. Distribution Strategy & Packaging

Target execution:

```
./portpilot
```

Zero runtime prerequisites:

- No Node.js
- No npm
- No Python
- No Docker
- No PostgreSQL
- No Redis
- No external backend daemon
- No web server

Embedded fonts and brand assets reside inside the executable.

Initial build targets:

| Architecture | Output                |
| ------------ | --------------------- |
| Linux AMD64  | portpilot-linux-amd64 |
| Linux ARM64  | portpilot-linux-arm64 |

Target distributions:

- Ubuntu
- Debian
- Fedora
- Arch Linux
- Manjaro
- Linux Mint
- openSUSE

Real-world compatibility testing is required on target distributions; cross-compilation alone does not guarantee runtime compatibility.

Execution modes:

- `portpilot`: graphical desktop interface
- `portpilot doctor`: CLI diagnostics for system compatibility, firewall status, and permissions

In headless environments, the binary outputs an informative message and maintains CLI diagnostics.

# 8. MASTER PROMPT for AI Coding Agent

The following prompt serves as the technical brief for implementation:

# MASTER PROMPT — PORTPILOT

## Linux Native Port & Firewall Management Application

You are acting as a Principal Linux Systems Engineer, Senior Go Architect, Security Engineer, and Desktop Application Developer.

Your mission is to design, implement, test, document, and deliver a production-grade Linux desktop application named **PortPilot**.

Do not build a prototype, a static dashboard, or a mock application.

Build a real, functional, security-conscious Linux administration tool.

---

# 1. PRODUCT DEFINITION

Product name: PortPilot

Product category: Linux Desktop System Administration / Network Security

Primary purpose:

PortPilot provides a graphical interface for discovering, inspecting, monitoring, and managing listening network sockets, processes, firewall rules, Linux services, and container port mappings.

The application must allow users to:

- Discover all TCP and UDP listening sockets.
- Inspect processes owning network sockets.
- Understand which network interfaces expose a service.
- Block and unblock ports.
- Create temporary firewall rules.
- Create permanent firewall rules.
- Manage UFW and native nftables.
- Inspect and control Linux services.
- Inspect Docker, LXC, and KVM network mappings.
- Monitor active network connections.
- Maintain an audit history of all administrative operations.

The application must be safe by default.

# 2. NON-NEGOTIABLE ARCHITECTURAL REQUIREMENTS

## 2.1 Single Binary

The final product must be distributed as a single executable file per target architecture.

Example:

```
./portpilot
```

The user must not need to install:

- Node.js
- npm
- Python
- Docker
- A separate backend
- A separate web server
- A separate application runtime
- PostgreSQL
- Redis

Embed all application-owned resources into the executable.

Do not require a background PortPilot daemon.

Do not require a separate database server.

SQLite must be embedded.

System-provided capabilities such as Linux Kernel APIs, a graphical display stack, systemd, UFW, nftables, and authorization infrastructure may be used when available.

Clearly document unavoidable host dependencies.

## 2.2 Technology Stack

Primary language: Go.

Desktop UI: Gio.

Database: SQLite using a pure-Go driver such as modernc.org/sqlite.

Logging: Go structured logging.

Configuration: XDG Base Directory Specification.

Socket discovery: Linux Netlink with procfs fallback.

Process discovery: Linux /proc.

Firewall engines:

- UFW
- Native nftables

Privilege management:

- Polkit / pkexec
- sudo fallback

Service management:

- systemd
- OpenRC
- SysV init where feasible

Optional integrations:

- Docker Engine API
- LXC
- libvirt / KVM

Use Go modules.

Pin dependencies to stable, compatible versions. Verify actual library APIs and current documentation before implementation. Never invent package functions or unsupported APIs.

Do not introduce a frontend JavaScript runtime.

Do not introduce a WebView dependency.

Do not use a browser-hosted UI.

The application must open as a native desktop window.

## 2.3 Architecture Style

Use a Modular Monolith.

Maintain clear boundaries between:

- Presentation
- Application Services
- Domain Logic
- Infrastructure
- System Adapters
- Privileged Operations
- Persistence

Avoid unnecessary microservices, networking layers, and abstractions.

All core functionality must be independently testable.

# 3. PROJECT STRUCTURE

Use the following initial structure:

```
cmd/portpilot/
    main.go

internal/
    app/
    gui/
        components/
        screens/
        theme/
    core/
        ports/
        processes/
        firewall/
        services/
        network/
        events/
    discovery/
        sockets/
        procfs/
        netlink/
        interfaces/
        containers/
    firewall/
        ufw/
        nftables/
        rules/
        scheduler/
    system/
        privilege/
        polkit/
        services/
        distro/
        capabilities/
    integrations/
        docker/
        lxc/
        libvirt/
    storage/
        sqlite/
        migrations/
        repositories/
    config/
    logging/
    security/

assets/
    icons/
    fonts/

docs/
scripts/
tests/
    integration/
    fixtures/

go.mod
go.sum
Makefile
README.md
LICENSE
AGENTS.md
```

You may improve this structure when technically justified, but do not compromise modular boundaries.

# 4. PORT DISCOVERY ENGINE

Implement a real Linux socket discovery engine.

It must discover:

- TCP LISTEN sockets.
- UDP bound sockets.
- IPv4 sockets.
- IPv6 sockets.
- Loopback sockets.
- Wildcard-bound sockets.
- Interface-specific sockets.
- Multiple sockets using the same port.
- Sockets across supported network namespaces where permissions allow.

For each socket, collect:

- Protocol
- Address family
- Local address
- Local port
- Remote address when applicable
- Socket state
- PID
- Process name
- Executable path
- Process owner
- Parent PID where available
- Associated system service where detectable
- Network namespace
- Container association where detectable

Use Netlink as the preferred socket discovery mechanism.

Use procfs as a fallback.

Use ss only as an optional diagnostic and cross-check mechanism.

Handle:

- Missing process ownership information.
- Permission denied.
- Processes exiting during scanning.
- PID reuse.
- Incomplete socket metadata.
- IPv4 and IPv6 dual-stack behavior.
- Network namespace boundaries.

Do not incorrectly classify every UDP socket as a listening TCP-style service.

Do not equate a wildcard bind address with actual Internet reachability.

Implement periodic refresh with cancellation and proper goroutine lifecycle management.

Avoid excessive polling.

# 5. PROCESS INSPECTOR

Implement a process inspection subsystem.

Required functionality:

- List processes associated with sockets.
- Display PID and PPID.
- Display executable path.
- Display process owner.
- Display command line where permitted.
- Associate sockets with processes.
- Associate processes with systemd services when possible.
- Display CPU and memory usage where available.

Actions:

- Send SIGTERM.
- Send SIGKILL only after explicit confirmation.
- Stop associated systemd service.
- Restart associated systemd service when supported.

Never automatically kill a process just because its port is exposed.

Never assume that stopping a process and stopping its service are equivalent.

Protect against PID reuse by validating process identity before performing destructive operations.

# 6. FIREWALL ARCHITECTURE

Implement a firewall abstraction:

```
FirewallManager
```

With separate implementations:

```
UFWBackend
NFTablesBackend
```

Each backend must implement a common interface for:

- Detect()
- Status()
- ListRules()
- AddRule()
- DeleteRule()
- ValidateRule()
- Capabilities()

Use typed models.

Do not execute arbitrary shell commands.

Use argument-based process execution with fixed executable paths and strict input validation.

Prefer a supported nftables Netlink library over parsing and rewriting arbitrary nft command output.

## 6.1 UFW

Support:

- Detect installation.
- Detect active/inactive state.
- List rules.
- Add TCP rules.
- Add UDP rules.
- Add IPv4/IPv6 rules where supported.
- Allow.
- Deny.
- Reject.
- Delete rules.
- Source IP restrictions.
- Interface restrictions where supported.
- Rule comments.
- Rule ownership metadata where possible.

Validate every requested operation before execution.

Do not blindly delete rules using unstable rule numbers without verifying rule identity.

## 6.2 Native nftables

Support:

- Read existing tables.
- Read chains.
- Read rules.
- Inspect hooks and policies.
- Manage only PortPilot-owned resources.
- Create a dedicated inet table named portpilot.
- Create dedicated chains and rules.
- Attach comments identifying ownership.
- Delete only explicitly identified PortPilot-owned rules.
- Preserve all unrelated firewall configuration.

Never flush the entire nftables ruleset.

Never modify unrelated tables.

Never overwrite existing firewall policies.

## 6.3 Firewall Conflict Detection

This is a critical safety requirement.

Detect whether:

- UFW is installed.
- UFW is active.
- Native nftables rules exist.
- iptables legacy is active.
- Another firewall manager is controlling the system.
- The current process has sufficient permissions.

If UFW is active, UFW must be the authoritative mutation backend.

Do not independently modify nftables rules that are managed by UFW.

If ownership is ambiguous, use read-only mode and explain the conflict.

Do not silently disable another firewall manager.

Display the detected firewall configuration to the user.

# 7. TEMPORARY FIREWALL RULES

Implement temporary blocking with reliable expiration.

Presets:

- 5 minutes
- 15 minutes
- 30 minutes
- 1 hour
- 6 hours
- 24 hours
- Custom duration

Requirements:

1. Validate the requested rule.
2. Apply the firewall rule.
3. Store rule metadata.
4. Store expiration time.
5. Display remaining time.
6. Remove the rule after expiration.
7. Record the result in the audit log.

For native nftables, use appropriate native timeout support where technically suitable.

For UFW, implement persistent expiration using systemd scheduling where available.

Do not rely solely on an in-memory Go timer.

If persistent expiration cannot be guaranteed on the current platform, explicitly tell the user.

When PortPilot starts, reconcile expired rules and scheduled actions.

Never remove unrelated firewall rules.

Handle application crashes and unexpected shutdowns.

# 8. PRIVILEGE AND SECURITY MODEL

NEVER run the entire GUI as root.

Separate the application into:

1. Unprivileged GUI and application core.
2. Restricted privileged helper operations.

The helper must use the same executable.

Example internal command:

```
portpilot __privileged
```

Use Polkit/pkexec when available.

Provide sudo fallback when necessary.

Implement a strict request/response protocol.

Allow only predefined privileged operations.

Never accept arbitrary shell commands.

Never accept arbitrary executable paths from untrusted GUI input.

Validate:

- Port ranges.
- Protocols.
- IP addresses.
- Interfaces.
- Rule identifiers.
- Process identity.
- Service names.
- Requested actions.

Use structured JSON or another strictly validated protocol.

Do not expose a TCP or HTTP privileged control API.

Do not create a root-owned background daemon unless absolutely necessary and separately justified.

Never trust user-writable configuration files as authorization for privileged actions.

Every privileged action must be auditable.

Handle cancellation and failure safely.

# 9. SERVICE MANAGER

Implement automatic detection of the init system.

Support:

- systemd
- OpenRC
- SysV init where feasible

Capabilities:

- List services.
- Find service associated with a process.
- Get status.
- Start.
- Stop.
- Restart.
- Enable.
- Disable.

Do not expose unsupported operations.

Do not confuse a process name with a service name.

Always confirm potentially disruptive operations.

# 10. NETWORK AND CONTAINER INTEGRATIONS

Implement optional discovery integrations.

Docker:

- Read container list.
- Read container status.
- Read published ports.
- Map host ports to containers.
- Display container name and ID.

LXC:

- Detect installed LXC.
- Discover containers where permitted.
- Identify virtual interfaces and bridge associations.

KVM/libvirt:

- Detect libvirt availability.
- Discover virtual networks.
- Identify bridge interfaces.
- Associate known network interfaces with virtual networks.

Do not require Docker, LXC, or libvirt.

Do not require Docker socket access for normal application startup.

Use read-only access by default.

Treat Docker socket access as highly privileged.

Do not change container networking automatically.

# 11. DATABASE

Use embedded SQLite.

Suggested tables:

```
settings
firewall_rules
scheduled_rules
action_history
process_snapshots
network_interfaces
application_state
schema_migrations
```

Requirements:

- Versioned migrations.
- Transactional writes.
- Prepared statements.
- Foreign key constraints.
- Database integrity checks.
- Safe backup strategy.
- Graceful recovery from corrupted database.
- No storage of secrets unless absolutely necessary.

The database is a record of PortPilot state, not the authoritative source of kernel firewall state.

Always reconcile persisted state against actual system state.

Use XDG directories:

```
~/.config/portpilot/
~/.local/share/portpilot/
~/.cache/portpilot/
```

Set appropriate file permissions.

# 12. GUI DESIGN

Use Gio for the desktop interface.

Build a modern, professional, responsive system administration application.

Support:

- Dark mode.
- Light mode.
- Consistent typography.
- Consistent spacing.
- Reusable components.
- Accessible navigation.
- Keyboard shortcuts.
- Search.
- Sorting.
- Filtering.
- Pagination or virtualized lists where needed.
- Detail panels.
- Confirmation dialogs.
- Notifications.
- Loading states.
- Error states.
- Empty states.

Main navigation:

1. Overview
2. Ports
3. Processes
4. Firewall
5. Rules
6. Services
7. Containers
8. Activity
9. Settings

## Dashboard

Display:

- Total TCP listening sockets.
- Total UDP bound sockets.
- Localhost-only sockets.
- Wildcard-bound sockets.
- IPv6 sockets.
- Running services.
- Firewall status.
- Active rules.
- Security warnings.

## Ports Screen

Display:

- Port
- Protocol
- Local address
- State
- Process
- PID
- Service
- Firewall status

Filters:

- TCP
- UDP
- IPv4
- IPv6
- Localhost
- All interfaces
- Process
- Port range
- Firewall state

Actions:

- Inspect
- Block
- Unblock
- Temporary block
- View process
- View service

## Firewall Screen

Display:

- Active firewall backend.
- Firewall status.
- Default inbound policy.
- Default outbound policy where available.
- All existing rules.
- Rule ownership.
- Rule details.

Allow the user to create and edit rules through validated forms.

## Activity Screen

Display:

- Timestamp
- Operation
- Target
- Backend
- User
- Result
- Error details

# 13. SAFETY AND UX RULES

Never silently modify firewall configuration.

Before destructive operations:

- Explain the target.
- Explain the impact.
- Show the requested change.
- Require explicit confirmation.

Prevent accidental SSH lockout.

When modifying SSH-related firewall rules, show a dedicated warning.

Never block loopback or essential system networking automatically.

Do not classify a port as dangerous merely because it is open.

Clearly distinguish:

- Listening
- Firewall allowed
- Firewall denied
- Local-only
- Wildcard-bound
- Externally reachable (unknown unless verified)

Never claim external reachability based only on ss output.

Use safe defaults.

Provide an emergency rollback mechanism for firewall changes.

# 14. DISTRIBUTION AND BUILD SYSTEM

The main deliverable must be a single executable per architecture.

Targets:

```
linux/amd64
linux/arm64
```

Use:

```
go build
```

Provide:

```
make build
make test
make lint
make release
make clean
```

Embed:

- Icons.
- Fonts.
- Default configuration templates.
- UI resources.

Avoid runtime downloads.

Avoid mandatory installation scripts.

Provide an optional installation script, but the executable must work without installation when host prerequisites are available.

Investigate CGO and graphics backend requirements before claiming static or universal binary compatibility.

Document any unavoidable system libraries.

Do not promise that a binary built for a newer glibc distribution will work on every older Linux distribution.

Test actual binaries on representative distributions.

Provide SHA256 checksums for releases.

# 15. OBSERVABILITY AND ERROR HANDLING

Use structured logs.

Provide diagnostic information for:

- Permission denied.
- Missing UFW.
- Missing nftables support.
- Unsupported init system.
- Missing GUI display.
- Database errors.
- Netlink errors.
- Process disappearance.
- Docker socket unavailable.
- Firewall backend conflict.

Never hide system errors.

Never silently ignore failed firewall operations.

Display actionable errors.

# 16. TESTING STRATEGY

Implement:

Unit tests:

- Port parsing.
- Socket parsing.
- Process ownership.
- Firewall rule validation.
- IP validation.
- Rule serialization.
- Backend selection.
- Expiration calculations.

Integration tests:

- UFW detection.
- nftables rule lifecycle.
- Process discovery.
- Service discovery.
- SQLite migrations.
- Privilege helper validation.

Safety tests:

- Ensure no unrelated nftables tables are modified.
- Ensure UFW-managed nftables rules are not independently modified.
- Ensure no arbitrary command execution is possible.
- Ensure temporary rules expire correctly.
- Ensure SSH lockout protection works.
- Ensure application startup works without root.
- Ensure missing optional integrations do not break startup.

Use isolated Linux network namespaces and disposable test environments where possible.

Never run destructive integration tests against the developer's real host firewall.

# 17. DEVELOPMENT PHASES

Do not attempt to generate the entire project in one uncontrolled implementation step.

Follow these phases:

PHASE 1 — Foundation

- Initialize Go module.
- Establish package structure.
- Implement configuration.
- Implement logging.
- Implement SQLite.
- Implement application lifecycle.
- Implement build system.

PHASE 2 — Native GUI

- Create desktop window.
- Implement navigation.
- Implement reusable components.
- Implement theme.
- Build Dashboard layout.
- Build Ports and Processes screens.

PHASE 3 — Discovery Engine

- Implement Netlink discovery.
- Implement procfs fallback.
- Implement process mapping.
- Implement interface discovery.
- Implement refresh and cancellation.
- Add unit tests.

PHASE 4 — Firewall Read-Only

- Implement backend detection.
- Implement UFW inspection.
- Implement nftables inspection.
- Implement conflict detection.
- Display real firewall information.

PHASE 5 — Firewall Mutations

- Implement privilege helper.
- Implement Polkit integration.
- Implement UFW mutations.
- Implement nftables mutations.
- Implement rule ownership.
- Implement rollback.
- Implement integration tests.

PHASE 6 — Temporary Rules

- Implement expiration scheduler.
- Implement persistent expiration.
- Implement startup reconciliation.
- Implement UI countdown.
- Test crash recovery.

PHASE 7 — System Integration

- Implement systemd integration.
- Implement OpenRC/SysV capability detection.
- Implement Docker integration.
- Implement LXC and KVM discovery.

PHASE 8 — Activity and Security

- Implement audit history.
- Implement diagnostics.
- Implement safety checks.
- Implement permission handling.
- Perform security review.

PHASE 9 — Release

- Optimize binary.
- Build AMD64 and ARM64 targets.
- Verify embedded assets.
- Test on multiple distributions.
- Generate checksums.
- Complete documentation.

# 18. DOCUMENTATION REQUIREMENTS

Maintain:

```
README.md
AGENTS.md
docs/ARCHITECTURE.md
docs/SECURITY.md
docs/FIREWALL.md
docs/BUILD.md
docs/CHANGELOG.md
```

After each implementation phase:

- Update CHANGELOG.md.
- Document architectural decisions.
- Document changed modules.
- Document security implications.
- Document tests executed.
- Record known limitations.

Maintain a clear relationship between modules and their responsibilities.

Do not leave undocumented architectural changes.

# 19. ENGINEERING RULES

- Production-quality Go.
- Small, cohesive packages.
- Explicit interfaces.
- Context-aware operations.
- Proper goroutine cancellation.
- No data races.
- No global mutable state unless justified.
- No unnecessary dependencies.
- No hardcoded executable paths without detection.
- No arbitrary shell execution.
- No fake data in production UI.
- No placeholder functionality presented as complete.
- No swallowed errors.
- No destructive system modifications during startup.
- No implicit firewall changes.
- No unnecessary root permissions.

Do not claim a feature is complete until it has been implemented and tested.

If a feature cannot be implemented safely on a particular Linux distribution, expose the limitation explicitly.

# 20. INITIAL EXECUTION INSTRUCTIONS

Before writing implementation code:

1. Inspect the development environment.
2. Inspect the Go version.
3. Inspect Linux graphics support.
4. Verify Gio compatibility.
5. Verify SQLite driver compatibility.
6. Verify Netlink library APIs.
7. Verify nftables library APIs.
8. Review UFW behavior and supported command options.
9. Review Polkit integration requirements.
10. Document architectural decisions.

Then create:

- Initial project skeleton.
- AGENTS.md.
- ARCHITECTURE.md.
- SECURITY.md.
- BUILD.md.
- Initial Makefile.

Implement Phase 1 and Phase 2 before moving to system-changing functionality.

For firewall functionality, implement read-only inspection first.

Never modify the developer's actual firewall without explicit approval.

At the end of every phase, report:

- Implemented features.
- Changed files.
- Dependencies added.
- Tests executed.
- Test results.
- Known limitations.
- Remaining work.

Do not stop at generating plans or documentation. Continue implementing the project phase by phase.

# FINAL ACCEPTANCE CRITERIA

PortPilot is considered production-ready only when:

1. It launches as a native Linux desktop application.
2. It is distributed as a single executable per target architecture.
3. It works without Node.js or a separate backend.
4. It discovers real TCP and UDP sockets.
5. It correctly maps sockets to processes where permissions allow.
6. It detects UFW and nftables.
7. It does not conflict with existing firewall managers.
8. It can safely add and remove firewall rules.
9. Temporary rules have reliable expiration.
10. It does not require the GUI to run as root.
11. It maintains an audit trail.
12. It handles unavailable optional integrations gracefully.
13. It passes unit and integration tests.
14. It has documented Linux compatibility limitations.
15. It does not modify unrelated firewall configuration.

Your priority order is:

1. Security
2. Correctness
3. Data integrity
4. Cross-distribution compatibility
5. Reliability
6. Usability
7. Performance
8. Visual polish

Build PortPilot as a real Linux administration product, not a demonstration project.

## 9. Recommended Implementation Sequence

1. Architecture and documentation.
2. Native desktop GUI scaffolding.
3. Socket and port discovery engine.
4. Read-only firewall inspection.
5. Confirmed firewall mutation with rollback safety.
6. Container integrations and release packaging.

Recommendation for initial builds: commence development and verification on an Ubuntu 22.04 environment on system `$USER`. Establish discovery and GUI first; never mutate host firewall rules during initial verification runs.

### Key Tenets

- Zero runtime dependencies: no Node.js, external backend, or local web server.
- UFW and nftables support must respect ownership detection without concurrent conflict.
- The desktop interface runs strictly unprivileged; elevated privileges are requested only for administrative operations.

Result: This architecture establishes PortPilot as a robust, production-grade Linux administration utility suitable for public release on GitHub.
