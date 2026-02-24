# FAQ

## Why Sambmin?

**Why not just use RSAT?**
RSAT requires a Windows workstation and doesn't work from a browser. If your Samba AD domain is managed by a team using macOS, Linux, or remote access, RSAT means maintaining a Windows VM just for directory management. Sambmin gives you the same capabilities from any browser.

**Why not phpLDAPadmin / Apache Directory Studio / etc.?**
Generic LDAP tools work but don't understand AD-specific concepts like GPOs, DRS replication, Fine-Grained Password Policies, or Kerberos service accounts. Sambmin is purpose-built for Samba AD and exposes these features with appropriate UI.

**Why not use samba-tool directly?**
Sambmin uses `samba-tool` under the hood for writes. The value is in the read side: browsing objects with context, searching across the directory, visualizing replication topology, checking DNS consistency across DCs — tasks that are tedious from the command line.

## Platform Support

**What platforms can Sambmin run on?**
- **FreeBSD** — Primary platform, most tested
- **Linux** — Ubuntu/Debian with systemd (see [installation guide](installation/linux.md))
- **macOS** — Development only (see [dev setup](installation/macos.md))

**Is there a Docker image?**
Not yet. Docker support is planned but not available in the beta.

**Does it work with Microsoft AD (not Samba)?**
Sambmin is designed for Samba AD. The LDAP read path would work with Microsoft AD, but write operations use `samba-tool` which is Samba-specific. Microsoft AD support is not planned.

**What browsers are supported?**
Any modern browser with JavaScript enabled: Chrome, Firefox, Safari, Edge. No IE11 support.

## Architecture

**Why does Sambmin use Python scripts for writes instead of direct LDAP?**
`samba-tool` handles Samba-internal consistency checks — SID allocation, RID pool management, schema validation, DNS record formatting. Reimplementing these in Go would be fragile and risk data corruption. The Python scripts are thin wrappers that accept JSON and return JSON, keeping the integration clean.

**Why are sessions in memory?**
The in-memory implementation works well for single-server deployments. The trade-off is that sessions are lost on server restart — users have to log in again.

**Can I run multiple Sambmin instances for HA?**
Not currently. In-memory sessions mean you'd need sticky sessions or a shared session store. This is on the roadmap.

## Known Limitations

**Why does replication monitoring require Domain Admin login?**
The `drs showrepl` command uses DCE/RPC and requires Domain Admin privileges. The read-only service account gets `WERR_DS_DRA_ACCESS_DENIED`. Log in as a Domain Admin to view replication details.

**Why is the Settings page empty?**
The Settings UI exists but has no backend persistence yet. It shows mock data. Application settings are configured via `config.yaml`.

**Why can't I export keytabs from the web UI?**
Keytab export requires root-level access to the SAM database, which a web application shouldn't have. Sambmin shows the equivalent `samba-tool` CLI commands to run on the DC directly.

**One of my DCs shows as "unreachable" — is that a bug?**
Probably not. Sambmin reports the actual connectivity status. If a DC is powered off, on a different network segment, or behind a firewall, it will show as unreachable. Check network connectivity to port 636 on that DC.

## Troubleshooting

**I get "authentication not configured" when trying to log in**
This means the LDAP connection to your DC failed at startup. Sambmin fell back to mock mode. Check:
- DC hostname/IP and port in config.yaml
- Service account credentials in SAMBMIN_BIND_PW
- Network connectivity: `openssl s_client -connect dc1.yourdomain.com:636`
- Server logs: `tail /var/log/sambmin.log` or `journalctl -u sambmin`

**Login fails with valid credentials**
- Ensure the user account isn't locked or disabled in AD
- Check if rate limiting kicked in (429 responses in the browser console)
- Verify the base_dn in config.yaml matches your domain

**Changes I make aren't showing up**
- Write operations go through `samba-tool` → the Python scripts. Check that:
  - `scripts_path` in config.yaml points to the correct directory
  - Python 3.11+ is installed and accessible
  - The scripts have execute permission
- AD replication between DCs may cause a delay if you're reading from a different DC than the one that received the write

**The dashboard shows stale data**
There's no real-time push (WebSocket) in the beta. Refresh the page to get current data.
