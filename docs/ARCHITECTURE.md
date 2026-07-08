# Architecture

## System Overview

Sambmin is a three-tier web application:

1. **Frontend** — React 19 SPA with Ant Design 5, served by nginx
2. **Backend** — Go API server using stdlib `net/http`, communicating with AD via LDAP and `samba-tool`
3. **Directory** — One or more Samba AD Domain Controllers running LDAP (port 636/LDAPS)

```
┌─────────────┐     HTTPS     ┌─────────────┐     LDAPS      ┌──────────────┐
│   Browser   │ ────────────► │    nginx     │ ──────────────► │   Samba DC   │
│  (React SPA)│               │  (reverse    │                 │  (LDAP read) │
└─────────────┘               │   proxy)     │                 └──────────────┘
                              │              │
                              │   ┌──────┐   │     stdin/out   ┌──────────────┐
                              │   │  Go  │◄──┘ ──────────────► │   Python +   │
                              │   │  API │                     │  samba-tool   │
                              │   └──────┘                     │  (writes)     │
                              └──────────────┘                 └──────────────┘
```

## Read/Write Split

**Reads** go directly from Go to LDAP. The Go backend maintains a connection pool to all configured DCs, with automatic failover. Direct LDAP queries are fast and avoid spawning processes.

**Writes** are delegated to Python scripts that wrap `samba-tool`. This is intentional: `samba-tool` handles Samba-specific consistency checks (SID allocation, RID pool management, schema validation) that would be error-prone to reimplement. The Python scripts accept JSON on stdin and emit JSON on stdout, with errors on stderr.

This split means reads are fast (single LDAP round-trip) while writes are safe (Samba's own tooling enforces correctness).

## Authentication Model

Sambmin uses a two-tier authentication model:

### Service Account (Reads)

A dedicated service account (configured via `bind_dn` and `SAMBMIN_BIND_PW`) is used for all LDAP read operations. This account needs only read access to the directory — no write permissions, no Domain Admin membership.

### User Credentials (Writes)

When a user logs in, their password is encrypted with AES-256-GCM using a randomly generated server key and stored in the session. When a write operation is requested, the user's password is decrypted and passed to `samba-tool` via the Python scripts. This ensures:

- All mutations are attributed to the actual user, not a shared service account
- The service account cannot be used to make unauthorized changes
- AD audit logs correctly reflect who made each change

### Session Management

Sessions are stored in memory with automatic expiry (configurable, default 8 hours). Each session contains:
- User identity (DN, sAMAccountName, group memberships)
- AES-256-GCM encrypted password
- CSRF token
- Expiry timestamp

A background goroutine cleans up expired sessions every 5 minutes.

## RBAC Model

Four roles are derived from AD group membership:

| Role | AD Groups | Permissions |
|------|-----------|-------------|
| **Admin** | Domain Admins, Enterprise Admins | All operations including password policy, GPO, replication, FSMO, SPN, delegation, keytab |
| **DNS Admin** | DnsAdmins, Domain Admins, Enterprise Admins | DNS zone and record management |
| **Operator** | Account Operators, Domain Admins, Enterprise Admins | User, group, computer, contact, and OU CRUD |
| **Authenticated** | Any logged-in user | Read access to all objects, self-service profile and password change |

Role checks happen at the route level — write endpoints are wrapped with `RequireRole` middleware that returns 403 before the handler executes.

## samba-tool Integration Notes

Hard-won details of driving `samba-tool` programmatically. Each of these caused a real bug before it was encoded here:

- **Member and move targets are sAMAccountNames, not CNs.** `samba-tool group addmembers`, `user move`, and `group move` resolve objects by account name. A user DN's leading CN is usually the *display name* ("Jane Smith"), which fails with `Unable to find`. Handlers resolve DNs to sAMAccountName via LDAP first (`samAccountNameFromDN`), falling back to the CN only when LDAP is unavailable.
- **`--userou`/`--groupou` take base-DN-relative RDNs.** samba-tool appends the domain DN itself, so passing a full DN produces a doubled suffix (`OU=X,DC=a,DC=b,DC=a,DC=b`) and fails. `relativeToBase` strips the base-DN suffix before invoking samba-tool. The `move` subcommands, by contrast, normalize and accept either form.
- **OUs cannot be created inside CN= containers.** AD's schema forbids `organizationalUnit` as a child of containers like `CN=Users` or `CN=Computers` (`LDAP_NAMING_VIOLATION`). The API rejects CN= parents with a 400 up front; users, groups, and computers may live in containers, only OUs are restricted.
- **The `-H ldap://` decision is per-subcommand, not per-command-group.** `drs` and `dns` use DCE/RPC and reject `-H ldap://...`. Within `domain`, it varies: `exportkeytab` reads the local SAM (no `-H`), while `passwordsettings` is LDAP-capable and *requires* `-H` — without it, samba-tool opens `sam.ldb` directly, which needs root and fails with `Permission denied` under the service user. `sambaToolWantsLDAPURL` encodes the rule.

## Delegation of Control (dsacl)

Delegation is driven by `samba-tool dsacl get/set/delete`, and the interface has sharp edges worth recording:

- **`--car` only covers replication rights.** Its accepted values are the directory-replication / FSMO control-access rights (`get-changes`, `get-changes-all`, `repl-sync`, …) — *not* the delegation rights an admin usually wants. Everything else (reset password, create/delete objects, manage membership, read/full control) must be expressed as raw **SDDL** via `--sddl`. The `--car` path takes the trustee as a DN (`--trusteedn`); the SDDL path embeds the trustee as a **SID**, so Sambmin resolves trustee DN → `objectSid` (decoding the binary SID) before building the ACE.
- **Generic rights are canonicalized on store.** A container-inheritable Generic All (`(A;CI;GA;;;<sid>)`) is written back as **two** ACEs: an inherit-only `(A;CIIO;GA;;;<sid>)` for descendants plus the object's own expanded specific mask `(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;<sid>)`. Removal must delete the *stored* forms, not the submitted one — so the UI removes by the exact ACE strings returned from `get`, and both stored forms are attributed back to the single "Full control" template. Specific-rights ACEs (reset password, read-only, membership) round-trip unchanged.
- **"Delegations" = explicit ACEs granted to domain principals.** A DACL is full of inherited ACEs (flag `ID`) and non-inherited class defaults for well-known principals (SYSTEM, Domain Admins). Sambmin's delegation view shows only ACEs that are *not* inherited **and** whose trustee is a real domain principal (`S-1-5-21-…`), which cleanly excludes both noise sources.
- **The only colons in an SDDL string are the `O:`/`G:`/`D:`/`S:` component markers.** SIDs use hyphens and ACE fields use semicolons, so the DACL can be isolated by splitting on those markers; ACEs are then split on balanced parentheses (honoring nesting so conditional ACEs don't split wrong).

## DNS Backend Abstraction

Samba AD supports two DNS backends:

1. **Samba Internal DNS** — DNS data stored in `sam.ldb`, managed by `samba_dnsupdate`
2. **BIND9 DLZ** — DNS data in Samba's LDB but served by BIND9 via the DLZ module

Sambmin's DNS management reads zone and record data from LDAP (both backends store records in `CN=MicrosoftDNS` partitions) and delegates writes to `samba-tool dns`, which handles both backends transparently.

## Kerberos Abstraction

Samba ships with Heimdal on most platforms but can be built with MIT Kerberos. Sambmin's configuration accepts an `implementation` field (`heimdal` or `mit`) that adjusts:

- Keytab export commands
- Kerberos policy attribute interpretation
- Ticket lifetime display formatting

## Security Features

- **CSRF Protection** — Double-submit cookie pattern; `X-CSRF-Token` header must match `sambmin_csrf` cookie on all mutations
- **Rate Limiting** — Sliding window: 10 failed attempts per IP per minute, 5 per username per 15 minutes
- **Session Encryption** — User passwords encrypted with AES-256-GCM, random 256-bit key generated at startup
- **Cookie Security** — `HttpOnly`, `Secure`, `SameSite=Strict` on session cookies
- **Input Validation** — LDAP filter escaping on all user-supplied search terms, DN validation
- **CORS** — Configurable allowed origins, credentials mode
- **CSP** — `default-src 'none'` on API responses
- **Request Tracing** — Unique `X-Request-ID` on every request
- **Panic Recovery** — Middleware catches panics, returns 500, logs stack
- **Structured Logging** — JSON-formatted request logs with timing via `slog`

## Directory Structure

```
sambmin/
├── api/                        # Go backend
│   ├── cmd/sambmin/            # Entry point (main.go)
│   └── internal/
│       ├── auth/               # Session store, LDAP auth, RBAC
│       ├── config/             # YAML config loading
│       ├── directory/          # LDAP client, object mapping
│       ├── handlers/           # HTTP handlers (routes, CRUD, search, etc.)
│       ├── ldap/               # Connection pool, DC failover
│       └── middleware/         # CORS, CSRF, rate limiting, logging
├── web/                        # React frontend
│   ├── src/
│   │   ├── api/                # Typed API client
│   │   ├── components/         # Shared UI components
│   │   ├── pages/              # Page-level components
│   │   └── App.tsx             # Router, layout, auth context
│   └── package.json
├── scripts/                    # Python wrappers for samba-tool
├── deploy/
│   ├── freebsd/                # rc.d service, nginx config, pkg install
│   ├── linux/                  # systemd unit file
│   ├── apache/                 # Apache reverse proxy config
│   └── tls/                    # Let's Encrypt and self-signed CA scripts
├── docs/                       # Documentation
└── config.example.yaml         # (in api/)
```

## Known Limitations

- **In-memory sessions** — Sessions are lost on server restart.
- **Single-server** — No clustering or HA for the Sambmin server itself (nginx upstream handles DC failover for LDAP).
- **Audit to stdout** — Audit logs go to structured JSON stdout/file.
- **No WebSocket** — Dashboard and replication status require manual refresh; no real-time push.
- **DRS permissions** — `drs showrepl` requires Domain Admin; the read-only service account gets `WERR_DS_DRA_ACCESS_DENIED`. Users must log in as Domain Admin to view replication details.
