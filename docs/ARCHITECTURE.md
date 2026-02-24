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
