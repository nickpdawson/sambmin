# Changelog

All notable changes to Sambmin will be documented in this file.

## [Unreleased]

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
