# Sambmin: Web-Based Samba AD Management Tool

## Context

There is no modern, production-ready web-based tool for administering Samba Active Directory Domain Controllers. Existing tools are either deprecated (SWAT), partial implementations (identidude, samba4-manager), or stalled community projects (Cockpit plugin). Windows RSAT/ADUC is powerful but ugly, cumbersome, and Windows-only. Samba admins are forced to choose between CLI-only management (`samba-tool`, `ldbtools`) or inadequate web UIs.

Sambmin fills this gap: a world-class web-based Samba AD management tool that steals the **feature depth** of Windows admin tools while delivering a modern, delightful interface inspired by Linear, Vercel, and Grafana. The tool runs on FreeBSD, supports multi-DC/multi-site environments (3-6 DCs, 2-3 sites), handles both Samba internal DNS and BIND9 backends, and works with both Heimdal and MIT Kerberos.

Target: internal production use first, then open-source release under GPLv3 (matching Samba's own license). Project name is working title - may rename before public release.

---

## Architecture

### Tech Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Backend API** | Go (net/http or Gin) | Static binaries, native Kerberos (gokrb5) + LDAP, cross-compile from macOS to FreeBSD (`GOOS=freebsd GOARCH=amd64`), zero runtime deps on target |
| **Utility Scripts** | Python 3 | Wraps `samba-tool`, `ldbtools`, BIND9 utilities; Samba's ecosystem is Python-native |
| **Frontend** | React + TypeScript + Vite | Modern build tooling, strong typing, fast HMR |
| **UI Components** | Ant Design 5 (ProComponents) | Enterprise admin components, ProTable, built-in theming/dark mode |
| **Data Visualization** | D3.js + React | Replication topology, group membership graphs, OU trees |
| **Database** | PostgreSQL | Audit logs, session data, app config, saved searches (NOT AD data) |
| **Reverse Proxy** | nginx (already on Bridger) | TLS termination, optional SPNEGO, static file serving |
| **Process Mgmt** | FreeBSD rc.d | Native, no extra dependencies |
| **TLS** | Let's Encrypt (ACME) or local CA | User-configurable during setup |

### System Architecture Diagram

```
Browser (Kerberos ticket or user/pass)
    |
    v
nginx (TLS termination, optional SPNEGO)
    |
    v
Go API Server (port 8443)
    |-- Direct LDAP to Samba DC(s)
    |-- gokrb5 for Kerberos auth
    |-- Calls Python scripts for samba-tool operations
    |-- PostgreSQL for app data (audit, sessions, config)
    |
    v
Samba AD DC(s) -- LDAP / RPC / DNS
```

### Key Architectural Decisions

1. **Go API calls Python scripts via exec** - Not embedded Python. Go shells out to Python scripts that wrap `samba-tool` and `ldbtools`. Clean separation: Go owns HTTP/auth/sessions, Python owns Samba CLI integration. Scripts live in `/usr/local/share/sambmin/scripts/`.

2. **LDAP as primary read path, samba-tool as primary write path** - Reading objects is fast via direct LDAP queries from Go. Writing/modifying uses `samba-tool` (via Python wrappers) because it handles all the AD-specific logic (SID allocation, group nesting rules, password policy enforcement, schema validation) that would be painful to reimplement.

3. **PostgreSQL for app-layer data only** - AD directory data stays in Samba's LDB/LDAP. PostgreSQL stores: audit logs, user preferences (saved searches, dashboard layout), session tokens, RBAC overrides, and application configuration.

4. **DNS backend abstraction** - The DNS management module detects whether each zone uses Samba internal DNS or BIND9 DLZ and routes operations accordingly. Internal DNS: `samba-tool dns` commands. BIND9: `rndc` + zone file manipulation or dynamic updates via `nsupdate`.

5. **Kerberos abstraction** - Authentication and KDC management abstract over Heimdal and MIT Kerberos. Detection at startup via binary/library inspection. Different code paths for keytab management, principal operations, and encryption type configuration.

6. **Multi-DC awareness** - The API connects to a configurable primary DC but can failover. Operations show which DC is being queried. Users can explicitly select a DC. Replication status is gathered by querying all DCs.

---

## Authentication & Security

### Authentication Flow

```
1. Browser navigates to https://sambmin.dzsec.net
2. nginx checks for Kerberos ticket (SPNEGO Negotiate header)
   - If valid ticket: passes authenticated principal to Go API via header
   - If no ticket: Go API serves login form
3. Login form accepts: username + password (LDAP simple bind to Samba DC)
4. Go API creates session (JWT or secure cookie), stores in PostgreSQL
5. All subsequent requests authenticated via session token
6. Session timeout configurable (default 8 hours)
```

### Security Measures

- **TLS everywhere** - nginx terminates TLS. Let's Encrypt via certbot/ACME or local CA cert (user choice during setup wizard)
- **RBAC** - Map AD group memberships to Sambmin roles: Full Admin, User Admin, DNS Admin, Read-Only. Configurable in Sambmin settings.
- **Audit trail** - Every mutation logged: who, what, when, from where, which DC, success/failure
- **CSRF protection** - SameSite cookies + CSRF token for form submissions
- **Rate limiting** - Login attempts rate-limited per IP
- **Network binding** - Go API binds to localhost by default; nginx handles external access
- **Secrets management** - Service account credentials stored encrypted, not in plaintext config
- **Content Security Policy** - Strict CSP headers to prevent XSS

---

## Feature Set (Phased)

### Phase 1: Foundation + Auth + Dashboard

**Project Setup**
- [ ] Initialize git repo in Sambmin directory
- [ ] Go module setup with directory structure
- [ ] React + Vite + TypeScript + Ant Design scaffolding
- [ ] PostgreSQL schema for audit/sessions/config
- [ ] nginx config template for reverse proxy + TLS
- [ ] FreeBSD rc.d service script
- [ ] CLAUDE.md with project conventions and tool permissions
- [ ] Setup wizard (first-run configuration: DC connection, TLS, admin account)

**Authentication**
- [ ] Kerberos/SPNEGO authentication (support Heimdal + MIT)
- [ ] LDAP username/password authentication
- [ ] Session management (secure cookies, configurable timeout)
- [ ] RBAC: role mapping from AD groups

**Application Shell**
- [ ] Sidebar navigation (collapsible)
- [ ] Theme system: light + dark mode (Ant Design token-based)
- [ ] Command palette (Cmd+K) - navigation + actions
- [ ] Keyboard shortcuts (G+U for Users, G+D for DNS, etc.)
- [ ] Breadcrumb navigation
- [ ] DC selector (choose which DC to query)
- [ ] Responsive layout (desktop-first, functional on tablet)

**Dashboard (First screen - see the environment before managing it)**
- [ ] DC health strip (one card per DC, color-coded status, last replication time)
- [ ] Alert banner (conditional: replication lag, locked accounts, DNS issues)
- [ ] Quick action cards (Create User, Reset Password, DNS Record, Unlock Account)
- [ ] Key metrics with deltas (total users, computers, groups, DNS zones)
- [ ] Recent activity timeline
- [ ] Replication topology mini-map (D3.js force-directed, simplified)

### Phase 2: Core Directory

**Users**
- [ ] List view with ProTable: sortable, filterable, column toggles
- [ ] Tabs: All | Active | Disabled | Locked Out | Recently Created
- [ ] Create user (drawer form with progressive disclosure)
- [ ] Edit user (inline drawer, all attributes)
- [ ] Delete user (summary confirmation)
- [ ] Password reset (modal with policy validation + generate button)
- [ ] Enable/disable accounts
- [ ] Unlock accounts
- [ ] Bulk operations (floating action bar: disable, enable, delete, move to OU, add to group)
- [ ] CSV/JSON import with dry-run validation
- [ ] Show equivalent `samba-tool` command for operations

**Groups**
- [ ] List view with ProTable
- [ ] Create group (security + distribution types)
- [ ] Edit group, manage membership
- [ ] Nested group membership visualization (D3.js directed graph)
- [ ] Effective membership resolution (show transitive members with path)
- [ ] Add/remove members with typeahead search

**Computers**
- [ ] List view: name, OS, site, last logon, status
- [ ] Detail view with machine account info
- [ ] Delete/disable machine accounts

**Organizational Units**
- [ ] Tree-table hybrid: OU tree on left, contents on right
- [ ] Create/rename/delete OUs
- [ ] Move objects between OUs (drag-drop in tree + bulk move)
- [ ] OU delegation display

### Phase 3: DNS (The Big One)

**DNS Backend Abstraction**
- [ ] Auto-detect Samba internal DNS vs BIND9 DLZ per zone
- [ ] Unified API that routes to correct backend
- [ ] For internal DNS: wrap `samba-tool dns` commands
- [ ] For BIND9: `rndc` + `nsupdate` for dynamic updates

**DNS Zone Management**
- [ ] Zone list view: name, type, record count, backend, status
- [ ] Create/delete zones (forward + reverse)
- [ ] Zone transfer configuration
- [ ] Zone delegation
- [ ] SOA record management

**DNS Record Management (Full Page View)**
- [ ] Tabbed record type filters: All | A/AAAA | CNAME | MX | SRV | TXT | NS | PTR
- [ ] Inline table editing (click to edit in place, no modals)
- [ ] Type-adaptive creation form (fields change based on record type)
- [ ] Bulk record operations (delete, change TTL)
- [ ] Record import/export

**DNS Diagnostics**
- [ ] Missing SRV records for AD services (_ldap._tcp, _kerberos._tcp, etc.)
- [ ] Missing reverse PTR records for A records
- [ ] Stale dynamic records (orphaned computer records)
- [ ] TTL inconsistency detection
- [ ] SOA serial comparison across DCs
- [ ] DNS resolution testing from each DC

### Phase 4: Infrastructure

**Replication**
- [ ] Topology visualization: D3.js force-directed graph with site grouping
- [ ] Nodes = DCs (color by site, shape by FSMO role), edges = replication links
- [ ] Per-partnership status table: source, dest, last sync, pending, status
- [ ] Force-sync button per partnership
- [ ] Conflict detection with side-by-side diff resolution
- [ ] Convergence monitoring
- [ ] Replication health alerts

**Sites & Services**
- [ ] Site list and management
- [ ] Subnet-to-site mapping
- [ ] Site link configuration (cost, schedule, transport)
- [ ] Visual site topology

**FSMO Roles**
- [ ] Visual role assignment display across DCs
- [ ] Transfer workflow with type-to-confirm
- [ ] Seize workflow (emergency, with strong warnings)
- [ ] Role health monitoring

### Phase 5: Policy & Security

**Group Policy (GPOs)**
- [ ] GPO list view
- [ ] Create/delete GPOs
- [ ] Link/unlink GPOs to OUs
- [ ] GPO inheritance visualization
- [ ] Policy settings browser (read/edit)
- [ ] GPO backup/restore
- [ ] Note: clearly surface Samba GPO limitations vs Windows

**Kerberos Management**
- [ ] Abstract over Heimdal and MIT implementations
- [ ] SPN management (add, remove, list per object)
- [ ] Keytab generation and download
- [ ] Encryption type configuration
- [ ] Ticket policy management
- [ ] KDC diagnostics (ticket issuance test, encryption negotiation test)

**Schema**
- [ ] Schema browser (read-only by default)
- [ ] Attribute and class hierarchy visualization
- [ ] Custom attribute creation (with type-to-confirm, schema changes are serious)

### Phase 6: Polish & Hardening

**Dashboard Enhancements**
- [ ] Saved searches / bookmarks
- [ ] Customizable dashboard layout

**Audit Log**
- [ ] Full log viewer with filtering by actor, action, object type, time range, result
- [ ] Export to CSV/JSON
- [ ] Retention policy configuration

**Settings**
- [ ] Connection configuration (DCs, failover order)
- [ ] TLS certificate management (Let's Encrypt renewal or CA cert upload)
- [ ] RBAC role-to-group mapping
- [ ] Session timeout configuration
- [ ] Notification/alerting configuration
- [ ] Application backup/restore

---

## UX/UI Design

### Design Philosophy: "Calm Power"

The interface feels like a quiet room full of sharp instruments. Everything is within reach, nothing is shouting for attention, and the interface trusts the user to be competent while protecting them from irreversible mistakes.

**NOT Windows ADUC:** No tree-view-only navigation, no 12-tab property sheets, no modal-per-operation, no "click 5 times to reset a password." We steal Windows' feature completeness and throw away its interaction model entirely.

### Six Governing Principles

1. **Progressive Disclosure, Not Progressive Hiding** - Common path prominent, advanced always visible but secondary. No "Advanced Settings" buttons hiding critical features behind extra clicks.

2. **Context Over Navigation** - Viewing a user? See their groups inline in the drawer. Don't navigate away. Every object is a hub connecting to related objects.

3. **Destructive Actions Require Intention** - Three tiers: type-to-confirm (delete OU with children, FSMO transfer, schema changes), summary confirmation (delete user, disable accounts), inline undo (edit description, change TTL).

4. **The Interface Is a Search Engine** - Command palette (Cmd+K) is the primary power-user navigation. Every object, action, and page reachable by typing. Prefixes: `>` actions, `@` users, `#` groups, `:` DNS records.

5. **Show System Health, Not Just Objects** - Dashboard foregrounds replication health, DNS consistency, DC status. Favicon changes based on system health. The admin should never be surprised by a problem they could have seen.

6. **Respect the Terminal** - Show equivalent `samba-tool` commands. Support CSV/JSON import/export. One-click copy for DNs, SIDs, UPNs, IP addresses.

### Navigation Structure

```
[sambmin logo]
[Domain selector dropdown]

--- Search (Cmd+K) ---

OVERVIEW
  Dashboard

DIRECTORY
  Users
  Groups
  Computers
  Organizational Units

INFRASTRUCTURE
  DNS
  Sites & Services
  Replication

POLICY & SECURITY
  Group Policy
  Kerberos
  FSMO Roles
  Schema

SYSTEM
  Audit Log
  Settings
```

Left sidebar, collapsible to icon-only. List views use right-side drawers (640px) for detail/edit - keeps the list visible. DNS and Replication use full-page views due to data density.

### Visual Design: "Professional Calm"

- **Inspiration:** Linear's density, Vercel's typography, Grafana's monitoring (but prettier)
- **Typography:** Inter (body), JetBrains Mono (DNs, SIDs, IPs, LDAP filters, CLI commands)
- **Color:** Restrained - blue for interactive elements only, status colors (green/amber/red) for actual status only, never decoration
- **Dark mode:** First-class, respects system preference, toggle in sidebar footer
- **Spacing:** 8px base grid, generous but not wasteful
- **Tables:** Ant Design ProTable with inline editing, row selection, floating batch action bar, column toggles, virtual scrolling for 10K+ rows

### Key Interaction Patterns

- **Object detail:** Right-side drawer keeps list context visible
- **Bulk operations:** Checkbox selection + floating bottom action bar (Linear-style)
- **DNS editing:** Inline table editing, no modals for simple changes
- **Replication topology:** Interactive D3.js force-directed graph with site boundaries
- **Group membership:** Directed graph showing nesting chains
- **Confirmations:** Tiered by severity (type-to-confirm / summary / inline undo)
- **Loading states:** Skeleton loaders matching table row height, never blank screens
- **Empty states:** Illustration + clear call-to-action, never blank tables
- **Timestamps:** Relative by default ("5m ago"), absolute on hover

---

## Project Structure

```
sambmin/
├── CLAUDE.md                    # Project instructions for Claude
├── LICENSE                      # GPLv3 (matches Samba's license)
├── README.md                    # Project documentation
├── docker-compose.yml           # Dev environment (PostgreSQL)
│
├── api/                         # Go backend
│   ├── cmd/
│   │   └── sambmin/
│   │       └── main.go          # Entry point
│   ├── internal/
│   │   ├── auth/                # Kerberos + LDAP authentication
│   │   │   ├── kerberos.go      # SPNEGO/gokrb5 handler
│   │   │   ├── ldap_bind.go     # LDAP password auth
│   │   │   └── session.go       # Session management
│   │   ├── config/              # App configuration
│   │   ├── dns/                 # DNS management (abstraction layer)
│   │   │   ├── backend.go       # Interface for DNS backends
│   │   │   ├── samba_dns.go     # Samba internal DNS implementation
│   │   │   └── bind9.go         # BIND9 DLZ implementation
│   │   ├── directory/           # LDAP operations (users, groups, OUs, computers)
│   │   ├── handlers/            # HTTP route handlers
│   │   ├── middleware/          # Auth, CORS, logging, rate limiting
│   │   ├── models/              # Data structures
│   │   ├── replication/         # Replication monitoring
│   │   ├── scripts/             # Python script executor
│   │   └── store/               # PostgreSQL data access
│   ├── go.mod
│   └── go.sum
│
├── scripts/                     # Python utility scripts
│   ├── requirements.txt
│   ├── user_ops.py              # User CRUD via samba-tool
│   ├── group_ops.py             # Group operations
│   ├── dns_ops.py               # DNS operations via samba-tool dns
│   ├── gpo_ops.py               # GPO management
│   ├── kerberos_ops.py          # Kerberos operations (Heimdal + MIT)
│   ├── replication_ops.py       # Replication status queries
│   └── schema_ops.py            # Schema operations
│
├── web/                         # React frontend
│   ├── src/
│   │   ├── App.tsx
│   │   ├── theme/
│   │   │   └── tokens.ts        # Light + dark theme tokens
│   │   ├── layouts/
│   │   │   └── AppLayout.tsx    # Shell: sidebar, toolbar, content
│   │   ├── components/
│   │   │   ├── CommandPalette/
│   │   │   ├── DCHealthStrip/
│   │   │   ├── ReplicationGraph/
│   │   │   ├── OUTree/
│   │   │   ├── GroupGraph/
│   │   │   └── ConfirmDialog/   # Tiered confirmation components
│   │   ├── pages/
│   │   │   ├── Dashboard/
│   │   │   ├── Users/
│   │   │   ├── Groups/
│   │   │   ├── Computers/
│   │   │   ├── OUs/
│   │   │   ├── DNS/
│   │   │   ├── Sites/
│   │   │   ├── Replication/
│   │   │   ├── GPO/
│   │   │   ├── Kerberos/
│   │   │   ├── FSMO/
│   │   │   ├── Schema/
│   │   │   ├── AuditLog/
│   │   │   └── Settings/
│   │   ├── hooks/               # Custom React hooks
│   │   ├── api/                 # API client layer
│   │   └── utils/
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
│
├── deploy/                      # Deployment configs
│   ├── freebsd/
│   │   ├── rc.d/sambmin         # FreeBSD service script
│   │   ├── nginx.conf           # nginx reverse proxy config
│   │   └── pkg-install.sh       # FreeBSD package dependencies
│   ├── tls/
│   │   ├── letsencrypt.sh       # ACME/certbot setup
│   │   └── local-ca.sh          # Local CA cert setup
│   └── setup-wizard.sh          # First-run configuration
│
└── docs/                        # Documentation
    ├── architecture.md
    ├── api.md
    ├── development.md
    └── deployment.md
```

---

## Build & Deployment Strategy

**Development:** macOS (or any machine with Go + Node.js)
**Cross-compilation:** `GOOS=freebsd GOARCH=amd64 go build` produces a FreeBSD binary from macOS. No Go installation needed on FreeBSD servers.
**FreeBSD server dependencies (via `pkg install`):** python311, postgresql15-server, nginx, samba418 (already present). No Go package needed on servers.
**Deployment:** SCP/rsync the Go binary + Python scripts + built React static files to Bridger. nginx serves the React build and proxies API requests to the Go binary.

---

## Development Environment Setup (Step 1)

1. **Initialize git repo** in the Sambmin directory
2. **Create CLAUDE.md** with project conventions and tool permissions
3. **Scaffold Go module** (`api/`)
4. **Scaffold React app** (`web/`) with Vite + TypeScript + Ant Design
5. **Create Python scripts directory** (`scripts/`) with requirements.txt
6. **Create deployment configs** (`deploy/freebsd/`)
7. **Set up PostgreSQL schema** for audit/sessions/config
8. **Configure nginx template** for reverse proxy

---

## Verification Plan

### Local Development Testing
- Go API starts and serves on configured port
- React dev server builds and hot-reloads
- Authentication works: Kerberos ticket from `kinit` passes SPNEGO negotiation
- Authentication works: LDAP bind with username/password
- LDAP queries return users/groups/computers/OUs from Samba DC
- Python scripts execute `samba-tool` commands successfully via Go exec
- DNS operations work against both Samba internal DNS and BIND9

### Integration Testing on Bridger
- SSH to bridger.dzsec.net as administrator
- Deploy Go binary + Python scripts + built React frontend
- nginx reverse proxy routes to Go API
- TLS works (Let's Encrypt or local CA)
- Full CRUD cycle: create user, modify, reset password, add to group, disable, delete
- DNS record creation/modification/deletion on both backends
- Replication status visible across all DCs
- Audit log captures all operations
- Dark mode toggle works
- Command palette finds objects across all types

### Security Testing
- Unauthenticated requests are rejected
- RBAC restricts operations based on AD group membership
- CSRF protection prevents cross-site attacks
- Rate limiting blocks brute-force login attempts
- Audit log cannot be tampered with by non-admin users
- TLS configuration scores A+ on SSL Labs (when using Let's Encrypt)
