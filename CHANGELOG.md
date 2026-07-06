# Changelog

All notable changes to Sambmin will be documented in this file.

## [0.1.0-beta.8] - 2026-07-06

### Fixed
- **Create User pre-filled `username@example.com` instead of the real domain** — the auto-name handler hardcoded `example.com` into the email field (which *was sent* to the server on create), and the Username field displayed a literal `@example.com` suffix. The login and `/auth/me` responses now include a `domain` field derived from the configured base DN (`DC=alpinenet,DC=us` → `alpinenet.us`); the create-user drawer uses it for the `@domain` username suffix, the auto-filled email address, and the email placeholder. The dead `userPrincipalName` form value (never rendered or sent — samba-tool derives the UPN from the domain) was dropped.

## [0.1.0-beta.7] - 2026-07-06

### Fixed
- **Password Policies page was blank (domain default policy and PSOs)** — `runSambaTool` skipped `-H ldap://localhost` for *all* `samba-tool domain` subcommands on the assumption they were DCE/RPC. But `domain passwordsettings` is LDAP-capable, and without `-H` samba-tool opens `/var/db/samba4/private/sam.ldb` directly — which requires root, so every policy read/write failed with `Permission denied` under the service user. The skip list is now precise (extracted to `sambaToolWantsLDAPURL`): only `drs`, `dns`, and non-`passwordsettings` `domain` subcommands (e.g. `exportkeytab`, which genuinely reads the local SAM) omit `-H`. Affects the domain default policy viewer/editor, all PSO operations, effective-policy lookup, and the password tester.
- **Password policy load failures now render a persistent error with a Retry button** in the Domain Default Policy card, instead of a transient toast followed by a permanently blank card.

## [0.1.0-beta.6] - 2026-07-05

### Fixed
- **Adding users to groups failed with "Unable to find <display name>"** — `handleAddGroupMember`/`handleRemoveGroupMember` passed the DN's leading CN (usually the display name, e.g. "Test User") to `samba-tool group addmembers`, which resolves members by sAMAccountName. Both handlers now resolve the member DN to its sAMAccountName via LDAP (`samAccountNameFromDN`), falling back to the CN only when LDAP is unavailable.
- **Creating an OU under a CN= container silently failed** — picking "Users"/"Computers" as the parent produced `LDAP_NAMING_VIOLATION` (organizationalUnit is not a valid child class of a container). The backend now rejects CN= parents with a clear 400, the Parent OU dropdown only offers real OUs (full DNs shown), and create-modal errors are surfaced in a Modal instead of being swallowed.
- **Creating a user or group in a specific OU never worked** — the frontend sends full DNs, but `samba-tool`'s `--userou`/`--groupou` flags expect an RDN sequence relative to the base DN (samba appends the domain DN itself, doubling it). New `relativeToBase` helper strips the base-DN suffix server-side before invoking samba-tool.
- **"Additional Groups" in the create-user drawer was collected but never sent** — the create request now carries `groups[]`; the backend adds memberships after a successful create and reports any failures back (`groupsFailed`), which the UI surfaces as a warning.

### Added
- **Move objects between OUs** — new endpoints `POST /api/users/{dn}/move`, `POST /api/groups/{dn}/move`, and `POST /api/ous/{dn}/move` (wrapping `samba-tool user|group|ou move`). Users and Groups pages gain a "Move to OU" action; the OUs page "Move OU" action (previously "not yet implemented") now works, with self/descendant destinations excluded and a domain-root option.

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
