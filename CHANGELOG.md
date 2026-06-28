# Changelog

All notable changes to Sambmin will be documented in this file.

## [0.1.0-beta.5] - 2026-06-28

### Fixed
- **DNS zone listing returned 502 from nginx, surfacing as "Zones (0)" in the UI** — `handleListDNSZonesLive` enriched every zone sequentially with a `samba-tool dns query` (~2s each over RPC), so on a 34-zone domain the request ran ~80s. The Go `http.Server` had `WriteTimeout: 30s`, which slammed the TCP connection shut mid-handler; nginx logged `upstream prematurely closed connection while reading response header from upstream` and returned 502. The handler kept running and eventually logged `status:200`, hiding the failure server-side. Two changes: (1) bumped `WriteTimeout` to 180s in `cmd/sambmin/main.go`; (2) parallelized per-zone enrichment with a worker pool of 8 in `handleListDNSZonesLive` (mirrors `expandContainers`). A 34-zone DC now responds in ~20s.

## [0.1.0-beta.4] - 2026-06-27

### Added
- **Auto-assign RFC2307 POSIX attributes on new users and groups** — when the domain is already using RFC2307 (detected by sampling for any existing `uidNumber`), Sambmin now sets `uidNumber`, `gidNumber`, `unixHomeDirectory`, and `loginShell` on newly created users, and `gidNumber` on newly created groups. Allocation mirrors LDAP Account Manager: max(existing) + 1, floored at a configurable minimum (default 10000). If the primary group (typically Domain Users) lacks a `gidNumber`, one is allocated and written back. Configurable via a new `rfc2307` block in `config.yaml` (`min_uid`, `min_gid`, `default_shell`, `home_template`). Without this, member hosts using `idmap config <domain> : backend = ad / schema_mode = RFC2307` (e.g. TrueNAS) silently drop newly created principals into winbind's negative cache, making them invisible to NSS-driven UI dropdowns.

## [0.1.0-beta.3] - 2026-05-06

### Fixed
- **DNS zone listing now surfaces records nested under container nodes** — `samba-tool dns query @ ALL` only enumerates immediate children of the zone root, so dynamically-registered IoT devices (e.g. an A record at `kp115-0e309f` plus a `_dyndns` TXT child) were invisible. `QueryAllRecords` now follows the initial query with parallel subqueries (cap 8) for any name reporting `Children>0`, remapping subquery names back into the flat zone view. `iot.dzsec.net` went from 5 visible records to ~110.

### Removed
- **Per-record Dynamic (`dyn`/`static`) column in the records table** — `samba-tool dns query` does not expose the `dwTimeStamp` field, so the parser was hardcoding every non-SOA record as dynamic. Will return once we read the `dnsRecord` blob via LDAP.

## [0.1.0-beta.2] - 2026-04-23

### Added
- **User Profile tab** — New tab in user detail drawer with Windows profile fields (profile path, logon script, home drive, home directory) and Unix/POSIX attributes (login shell, home directory, UID, GID number)
- All profile fields are editable inline with LDAP modify writes

### Fixed
- **DNS record create/update/delete now works** — `samba-tool dns` uses DCE/RPC, not LDAP; was incorrectly getting `-H ldap://localhost` appended which caused all DNS write operations to fail
- **DNS commands use primary DC from config** — replaced all hardcoded `localhost` / `dc1.example.com` with the configured primary DC hostname in both backend samba-tool calls and frontend CLI preview strings

## [0.1.0-beta.1] - 2026-04-09

First public beta release.

### Features
- **Users** — Create, modify, enable/disable, unlock, reset passwords, rename, move between OUs
- **Groups** — Create, modify membership, rename, delete; all AD group types supported
- **Computers** — List, create, delete, move; OS info and last logon display
- **Contacts** — Full CRUD with move and rename
- **Organizational Units** — Tree browser with create/delete
- **DNS Management** — Zone and record CRUD, SRV validator, cross-DC consistency checks; supports Samba internal DNS and BIND9 DLZ backends
- **GPO Browsing** — List, inspect, link/unlink GPOs to OUs
- **Replication Monitoring** — Topology visualization, per-partition status, force sync
- **Kerberos** — Policy viewer, service account browser, keytab export, SPN and delegation management
- **Password Policy** — Domain-wide editor and Fine-Grained Password Policies (PSOs)
- **Schema Browser** — Explore AD schema classes and attributes
- **Global Search** — Full-directory LDAP search with saved queries
- **Dashboard** — DC health, object counts, recent audit activity
- **Self-Service** — Profile viewer and password change for authenticated users
- **Settings** — Persistent GUI configuration (connection, auth, RBAC, application settings)
- **Authentication** — LDAP bind + optional Kerberos, AES-256-GCM encrypted sessions
- **RBAC** — Four roles mapped from AD group membership
- **Security** — CSRF protection, rate limiting, input validation, audit trail
- **Mock Mode** — Full UI exploration without a live AD environment
- **Multi-platform** — Pre-built binaries for FreeBSD, Linux, and macOS (amd64 + arm64)
- **Build automation** — Makefile with cross-compilation and release packaging

### Known Limitations
- Replication monitoring requires Domain Admin login (service account lacks DRS permissions)
- Keytab export requires root SAM access; UI provides CLI fallback commands
- TLS management in Settings is display-only (handled by reverse proxy)
- Write operations require samba-tool and Python 3.11+ on the server
- SPN search accepts account names only, not SPN service names
