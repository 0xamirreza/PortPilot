# Security

The window and `doctor` run as the invoking user. If the effective uid is root, the window refuses to start.

## Helper

`portpilot __privileged` reads one JSON request from stdin and writes one JSON response. It exits immediately when it is not root. pkexec starts it; the GUI does not keep a root process.

Allowed operations are `authorize`, `inspect`, `reconcile`, `apply`, `delete`, `rollback`, `service`, and `signal`. The expire subcommand accepts one rule id that is already in the root schedule file and is due.

The `authorize` operation inspects live root process socket file descriptors and live firewall status in one atomic invocation. The unprivileged GUI does not maintain any persistent root daemon or socket; clicking the lock icon immediately clears privileged caches and revokes elevated state.

The helper executes only `ufw`, `nft`, `iptables-legacy`, `systemctl`, and `systemd-run`, looked up under `/usr/sbin`, `/sbin`, `/usr/bin`, and `/bin`. Arguments are separate. Service names are a short character class with no path and no shell metacharacters. Signals are SIGTERM and SIGKILL. SIGKILL requires `confirm_kill`. Before a signal, start ticks in `/proc/<pid>/stat` must match the value the window captured.

User-writable config and SQLite are not authorization. The schedule file under `/var/lib/portpilot` is the expiry record. The helper appends `/var/lib/portpilot/audit.log`. The user database also records the result of each privileged call.

`assets/polkit/dev.portpilot.policy` is a sample policy. It is not copied into `/usr/share/polkit-1`.

## Discovery

Socket diagnostics and `/proc` are read. Process file descriptors that the user cannot read stay unread. Missing ownership is not labeled an attack.

Bind exposure is classified as localhost, wildcard, or a single address. The interface states that a wildcard bind does not establish reachability from another host.

Docker, LXC, and libvirt commands run only after Containers is opened, and only as list operations.

## What was not exercised on the desktop hosts

The interactive window run was Manjaro with XFCE and X11. The Ubuntu 22.04 machine also opened the window. `sudo -n` needs a password on those desktops, so the helper was not used to change UFW or nftables there. Debian 12, Fedora 41, Arch Linux, and openSUSE Leap 15.6 ran `doctor` only, with no display. Unit tests use fixtures. Do not point integration tests at the live firewall.
