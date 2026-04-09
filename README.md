# Sambmin

![Beta](https://img.shields.io/badge/status-beta-orange)
![License: GPLv3](https://img.shields.io/badge/license-GPLv3-blue)
![Go 1.23+](https://img.shields.io/badge/Go-1.23%2B-00ADD8)
![React 19](https://img.shields.io/badge/React-19-61DAFB)
![FreeBSD](https://img.shields.io/badge/platform-FreeBSD-AB2B28)

A web-based management tool for Samba Active Directory Domain Controllers. Sambmin provides a modern browser interface for the tasks AD administrators typically handle through RSAT or command-line tools — user and group management, DNS, GPO browsing, replication monitoring, Kerberos diagnostics, and more.

Built for organizations running Samba AD as their directory service, Sambmin replaces the need for a Windows workstation with RSAT installed. It reads directory data via direct LDAP queries for speed, and delegates write operations through `samba-tool` for safety and compatibility.

## Features

- **Users** — Create, modify, enable/disable, unlock, reset passwords, rename, move between OUs
- **Groups** — Create, modify membership, rename, delete; supports all AD group types
- **Computers** — List, create, delete, move; shows OS info and last logon
- **Contacts** — Full CRUD with move and rename support
- **Organizational Units** — Tree browser with drag-and-drop-style navigation, create/delete
- **DNS Management** — Zone and record management, SRV validator, cross-DC consistency checks; supports both Samba internal DNS and BIND9 DLZ backends
- **GPO Browsing** — List, inspect, link/unlink Group Policy Objects to OUs
- **Replication** — Topology visualization, per-partition status, force sync (requires Domain Admin login)
- **Kerberos** — Policy viewer, service account browser, keytab export, SPN and delegation management
- **Password Policy** — Domain-wide policy editor, Fine-Grained Password Policies (PSOs)
- **Schema Browser** — Explore AD schema classes and attributes
- **Global Search** — Full-directory LDAP search with saved queries
- **Dashboard** — DC health, object counts, recent activity across all DCs
- **Self-Service** — Users can view their profile and change their own password
- **Audit Trail** — All write operations logged with who/what/when
- **RBAC** — Four roles mapped from AD group membership (Admin, Operator, DNS Admin, Authenticated)
- **Security** — CSRF protection, rate limiting, AES-256-GCM encrypted sessions, input validation

<!-- Screenshots coming soon
![Dashboard](docs/screenshots/dashboard.png)
![Users](docs/screenshots/users.png)
![DNS](docs/screenshots/dns.png)
-->

## Quick Start

```bash
# Build everything (backend + frontend)
make build      # builds for current platform
make frontend   # builds the React frontend

# Or cross-compile for all supported platforms
make build-all  # FreeBSD, Linux, macOS (amd64 + arm64)

# Configure
cp api/config.example.yaml /usr/local/etc/sambmin/config.yaml
# Edit config.yaml with your DC addresses, base DN, and bind DN

# Run
export SAMBMIN_BIND_PW="your-service-account-password"
export SAMBMIN_CONFIG="/usr/local/etc/sambmin/config.yaml"
./sambmin
```

Sambmin runs in **mock mode** when no domain controllers are configured, allowing you to explore the UI without a live AD environment.

For complete installation instructions, see:
- [FreeBSD Installation Guide](docs/installation/freebsd.md) (primary platform)
- [Linux Installation Guide](docs/installation/linux.md) (Ubuntu/Debian)
- [macOS Development Setup](docs/installation/macos.md)

## Architecture

Sambmin uses a split read/write architecture: the Go backend reads AD data directly via LDAP for speed, while write operations are delegated to Python scripts that wrap `samba-tool` for compatibility with Samba's internal consistency checks. A two-tier authentication model uses a service account for read operations and the logged-in user's credentials (encrypted in-session with AES-256-GCM) for writes, ensuring all mutations are attributed to the correct user.

For details, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Building & Testing

```bash
make test       # run Go tests (221 tests)
make build      # build backend with version injection
make frontend   # build React frontend for production
make dist       # package release tarballs for all platforms
```

See [BUILD.md](docs/BUILD.md) for prerequisites and detailed build instructions.

## Security

Sambmin implements CSRF protection (double-submit cookie), per-IP and per-username rate limiting on login, AES-256-GCM session encryption, RBAC enforcement on all write endpoints, LDAP injection prevention, and input validation. The service account is read-only by design.

See [SECURITY.md](docs/SECURITY.md) for the full security model.

## Contributing

Bug reports, feature requests, and pull requests are welcome. See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

## License

Sambmin is licensed under the [GNU General Public License v3.0](LICENSE), matching Samba's license.

## Acknowledgments

Sambmin depends on the [Samba Project](https://www.samba.org/) (GPLv3) for `samba-tool` and the AD domain controller implementation it manages.

For a complete list of dependencies and their licenses, see [ATTRIBUTION.md](docs/ATTRIBUTION.md).
