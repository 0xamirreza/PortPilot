# Firewall

The window can read firewall state and, after confirmation, change only the backend selected for this host.

## Read

Without root, PortPilot records whether `ufw` and `nft` exist and whether `/etc/ufw/ufw.conf` says `ENABLED=yes`. It does not run `ufw status` or `nft list` without authorization.

The global `🔒 AUTHORIZE` button in the top window header (or a confirmed action) starts `pkexec` once. The helper runs `ufw status verbose`, `nft -j list ruleset`, and `iptables-legacy -S` when those binaries exist. The mode is then one of:

- `ufw` when UFW is active. Mutations use UFW only.
- `nftables` when UFW is not active, nft is readable, iptables-legacy has no rules, and the only table is `inet portpilot` or there are no tables. New rules go in `inet portpilot` with chain policy `accept`.
- `read-only` when another nftables table or iptables-legacy rules make ownership ambiguous.
- `unavailable` when the live ruleset could not be read.

Clicking the unlocked `ROOT (ACTIVE)` button in the header instantly revokes authorization, returning to unprivileged mode with zero lingering elevated data.

Both the Firewall overview and Rules management screens are fully structured:
- **Firewall Screen**: Shows backend indicator (`UFW (ACTIVE)`, `NFTABLES`), active reason banner, metrics bar (`ACTIVE RULES`, `SCHEDULES`, `UFW ACTIVE`, `UFW ON BOOT`, `NFT TABLES`), and action controls (`RECONCILE`, `ROLLBACK`).
- **Rules Screen**: Organized 4-row rule creation form (Port/Protocol, Source/Interface, Duration presets, and color-coded `ALLOW` / `DENY` / `REJECT` actions), followed by a structured table of active rules and temporary schedule countdown badges.

## Write

Commands are argument lists. There is no shell. UFW rules carry a `portpilot:<id>` comment and are deleted by that same specification, not by a displayed number. nftables rules in `inet portpilot` are deleted by handle, and only when the comment has the prefix.

Loopback destinations and the `lo` interface are rejected. Denying or rejecting a port that looks like SSH, including port 22, requires the confirmation flag.

ROLLBACK deletes the last rule id this helper stored. It does not restore foreign rules.

## Temporary rules

A duration is stored in `/var/lib/portpilot/scheduled.json` and armed with `systemd-run --on-active`. The timer runs `portpilot __privileged expire <id>`. On nftables, a timeout set in `inet portpilot` also expires the match. If systemd cannot arm a timer, an nftables rule stays until that kernel timeout and the next reconcile; a UFW rule is removed immediately because UFW cannot expire it. RECONCILE deletes scheduled rules whose time has passed. An in-memory timer is not the expiry mechanism.

Tests parse fixture output and a fake command runner. They do not change the host firewall.
