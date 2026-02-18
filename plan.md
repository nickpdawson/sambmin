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

---

## Milestones & Accomplishments

### Completed

**M1: Project Scaffold** (2026-02-17)
- [x] Git repo initialized
- [x] CLAUDE.md with project conventions
- [x] Go backend structure: `api/cmd/sambmin/main.go`, handlers, middleware, config, models, scripts executor, DNS backend interface
- [x] React + Vite + TypeScript + Ant Design 5 frontend scaffolded
- [x] Python samba-tool wrapper scripts: `user_ops.py`, `group_ops.py`, `dns_ops.py`
- [x] FreeBSD deployment configs: rc.d service script, nginx reverse proxy, TLS setup scripts
- [x] Theme system with light/dark mode (Ant Design token-based, system preference detection)
- [x] 13 placeholder pages for all navigation sections
- Commit: `1301174`

**M2: Application Shell & Dashboard** (2026-02-17)
- [x] Collapsible sidebar navigation with grouped menu items
- [x] Breadcrumb navigation
- [x] Dark/light mode toggle with localStorage persistence
- [x] Dashboard with DC health strip (3 DCs, color-coded status, FSMO role tags)
- [x] Alert banners (conditional: replication lag, locked accounts)
- [x] Quick action cards (Create User, Reset Password, DNS Record, Unlock)
- [x] Domain metrics (users, computers, groups, DNS zones, locked, disabled)
- [x] Recent activity timeline with success/failure indicators
- [x] Mock API handlers for dashboard data

**M3: Users Page** (2026-02-17)
- [x] ProTable with sortable, filterable columns (name, email, dept, title, status, last logon, groups)
- [x] Tab filters: All | Active | Disabled | Locked Out (with badge counts)
- [x] Row selection with bulk action bar (floating bottom bar, Linear-style)
- [x] User detail drawer (560px, identity with copy-to-clipboard, organization, account status, group memberships)
- [x] Create user drawer (progressive disclosure, auto-username, password generator, collapsible sections)
- [x] CLI equivalent display (show samba-tool command)
- [x] Per-row action menu (View, Reset Password, Unlock, Enable/Disable, Delete)
- [x] Mock API handler with 8 realistic AD users
- Commit: `9c46b31`

**M4: Command Palette** (2026-02-17)
- [x] Cmd+K keyboard shortcut (global)
- [x] cmdk-based fuzzy search
- [x] Actions group: Create User, Reset Password, Unlock Account, Create DNS Record, Force Replication
- [x] Navigation group: all 14 sections with keyboard shortcut hints
- [x] Custom CSS with blurred overlay, dark mode support
- Commit: `c7090a3`

**M5: DNS & Settings Pages** (2026-02-17)
- [x] DNS mock API: zones (6 zones, mixed samba/bind9), records (23 records), diagnostics (7 health checks)
- [x] Settings mock API: connection (3 DCs), TLS, auth (Kerberos/LDAP), RBAC (5 roles), application info
- [x] DNS frontend: zone list with stats, record table with type filter tabs, diagnostics with health summary
- [x] DNS record type badges, copy-to-clipboard, CLI equivalent commands
- [x] Settings frontend: connection card with DC table, TLS cert status, auth config, RBAC role mapping table, app info
- [x] All routes wired to mock handlers

**M6: Live LDAP Integration & Deployment** (2026-02-17)
- [x] LDAP connection pool (`api/internal/ldap/pool.go`) with multi-DC failover, TLS, health checks, paged search
- [x] AD attribute constants and UAC/groupType flag parsing (`api/internal/ldap/attributes.go`)
- [x] Directory client with typed queries (`api/internal/directory/`) — users, groups, computers, OUs, filters
- [x] Live handlers: `users_live.go`, `directory_live.go` (groups, computers, OUs)
- [x] Conditional routing: live LDAP handlers when DCs configured, mock handlers otherwise
- [x] Service account bind (CN=services) with SAMBMIN_BIND_PW env var
- [x] Cross-compiled and deployed to Bridger (FreeBSD 14.2)
- [x] LDAPS connection to Samba DC on localhost:636 (TLS 1.3, Let's Encrypt cert)
- [x] Verified: 31 real users, 48 computers, 50 groups from live AD
- [x] Real DC topology discovered: bridger, showdown, yellowstone, wintergreen, moran (5 DCs)

**M7: nginx + TLS + DNS** (2026-02-17)
- [x] nginx server block for `sambmin.dzsec.net` (separate from bridger.dzsec.net to avoid `/api/` conflict with MDM)
- [x] DNS CNAME record: `sambmin.dzsec.net → bridger.dzsec.net` (via `samba-tool dns add`)
- [x] Let's Encrypt cert expanded to cover `sambmin.dzsec.net` SAN (via Cloudflare DNS challenge)
- [x] SPA routing: `try_files $uri $uri/ /index.html` for client-side React Router
- [x] Static asset caching (30d immutable for hashed `/assets/` files)
- [x] Full site live at `https://sambmin.dzsec.net` with green lock

**M8: Groups, Computers, OUs Frontend** (2026-02-17)
- [x] Groups page: ProTable with type/scope tags, member count, search, tabs (All/Security/Distribution), detail drawer with member list
- [x] Computers page: ProTable with DNS hostname, OS, status, last logon, detail drawer with network/OS/account sections
- [x] OUs page: dual view (List/Tree toggle), tree built from API `/api/ous/tree`, detail drawer with location info
- [x] All three pages fetch live LDAP data from API
- [x] English locale fix (antd `en_US` ConfigProvider)

**M9: Live Dashboard + DNS Data** (2026-02-17)
- [x] Dashboard: live DC health checks (LDAP probe per DC with 3s timeout)
- [x] Dashboard: live recent activity (LDAP search for objects changed in last 24h)
- [x] DNS: `samba-tool dns` integration via `exec.Command` (`api/internal/dns/samba.go`)
- [x] DNS: live zone listing, record listing, diagnostics (AD SRV record checks)
- [x] FreeBSD rc.d service script with secrets.env for bind password
- [x] DC health fix: store resolved SAMBMIN_BIND_PW back into config for handler access
- [x] 4/5 DCs healthy (yellowstone genuinely unreachable)

**M10: Authentication System** (2026-02-17)
- [x] Session store with AES-256-GCM encrypted password storage (`api/internal/auth/session.go`)
- [x] LDAP bind authenticator supporting sAMAccountName, UPN, DN formats (`api/internal/auth/ldap_bind.go`)
- [x] RequireAuth middleware (`api/internal/auth/middleware.go`)
- [x] Login/logout/me HTTP handlers with secure cookies (`api/internal/handlers/auth.go`)
- [x] Auth initialization in main.go (session store + LDAP authenticator targeting primary DC)
- [x] React AuthProvider context with session persistence check on mount (`web/src/hooks/useAuth.tsx`)
- [x] Login page calls real `/api/auth/login`, redirects on success
- [x] Protected routes: unauthenticated users redirected to `/login`
- [x] Header shows username + logout button
- [x] API client auto-redirects to `/login` on 401
- [x] Auth tests: 10/10 pass (session CRUD, password encrypt/decrypt, expiry, DN conversion)
- [x] Deployed to Bridger, verified login with administrator and ndawson accounts

**M11: Write Operations — Backend** (2026-02-17)
- [x] `runSambaTool()` helper: executes samba-tool with user session credentials via `-U user%pass -H ldap://localhost`
- [x] `requireSession()` helper: extracts session from request, returns 401 if missing
- [x] User CRUD handlers: create, update (LDAP modify), delete, reset password, enable, disable, unlock
- [x] Group CRUD handlers: create, update, delete, add/remove members
- [x] Computer handler: delete (LDAP delete as user)
- [x] OU handlers: create, delete
- [x] DNS handlers: zone create/delete, record create/update/delete
- [x] `dirClient.ModifyAttributes()` and `dirClient.DeleteObject()` for LDAP write operations
- [x] Handler tests: 11/11 pass
- [x] FreeBSD rc.d script fix: `/usr/bin/env` to pass env vars through `daemon -u`
- [x] Cleaned error messages from samba-tool (strip warnings, extract meaningful error line)

**M12: Write Operations — Frontend Wiring** (2026-02-17)
- [x] Users page: enable/disable/unlock/delete actions call real API (with confirmation modals)
- [x] Users page: bulk operations (enable/disable/delete) wired for multi-select
- [x] Users page: reset password modal with real API call
- [x] CreateUserDrawer: POST /api/users with real form data
- [x] CreateUserDrawer: fetch live OUs from /api/ous for dropdown
- [x] CreateUserDrawer: fetch live groups from /api/groups for dropdown
- [x] DNS CreateRecordDrawer: create/edit records via real API
- [x] DNS delete confirmation: calls real DELETE endpoint with type/value query params
- [x] OUs page: create OU modal with name, description, parent OU
- [x] OUs page: delete OU with confirmation modal (warns about children)
- [x] Error dialogs use Modal.error for visibility (not corner notifications)

**M13: Write Operations — Debugging & Testing** (2026-02-18)
- [x] Fixed `--server=localhost` → `-H ldap://localhost` for samba-tool LDAP connection
- [x] Fixed Permission denied on sam.ldb (was trying local file access, now uses remote LDAP)
- [x] Fixed CN vs sAMAccountName: added LDAP lookup via `GetSamAccountName()` for user actions
- [x] Fixed delete confirmation modal not rendering (moved Modal.confirm outside try/catch, fixed useCallback deps)
- [x] Fixed `timeAgo()` showing "739664d ago" for epoch zero — now shows "Never" for users who haven't logged in
- [x] Fixed startup in mock mode: SAMBMIN_CONFIG env var must be set for process to find config.yaml
- [x] Verified: user create, enable, disable work end-to-end
- [x] Verified: login/logout works via LDAP bind against Samba DC
- [x] Error dialogs switched to Modal.error (unmissable vs corner notifications)
- [ ] Verify user delete works end-to-end
- [ ] Verify DNS record create/update/delete work end-to-end
- [ ] Verify OU create/delete work end-to-end
- [ ] Verify group create/delete/member management works

### Next Up

**M14: Replication & Infrastructure**
- [ ] D3.js replication topology visualization
- [ ] `samba-tool drs showrepl` integration for real replication status
- [ ] Sites & Services management
- [ ] FSMO role display and transfer workflow

**M15: Polish & Hardening**
- [ ] PostgreSQL integration for audit log, session storage, app config
- [ ] Saved searches / bookmarks
- [ ] Customizable dashboard layout
- [ ] CSV/JSON export from all list views
- [ ] User attribute editing in detail drawer (inline edit)
- [ ] Group membership visualization

---

## Write Operations Architecture

### The Problem: Read vs Write Credentials

The current `services` bind account is appropriate for **reading** the directory. Write operations (creating users, modifying groups, resetting passwords, deleting objects) require **Domain Admin** or equivalent privileges. We do NOT want to run the entire application with Domain Admin credentials — that violates least privilege.

### Solution: Two-Tier Authentication

**Tier 1: Service Account (always active)**
- Account: `CN=services,CN=Users,DC=dzsec,DC=net`
- Purpose: All read operations (list, search, detail views, dashboard metrics, DC health)
- Credentials: Bound at server startup via `SAMBMIN_BIND_PW`
- No user interaction required

**Tier 2: User Session (on-demand, for writes)**
- User authenticates via login form (LDAP bind with their own credentials) or Kerberos/SPNEGO
- Session stored in secure cookie (JWT or opaque token backed by PostgreSQL)
- Write operations execute under the **user's identity**, not the service account
- Two approaches for executing writes:

  **Approach A: LDAP Bind per Write (simpler, recommended for Phase 1)**
  - When user performs a write, API creates a new LDAP connection bound as the user
  - Pros: Simple, uses LDAP directly, user's permissions enforced by AD itself
  - Cons: Can't do everything via LDAP (password resets, some GPO ops need samba-tool)

  **Approach B: samba-tool with User Credentials (for operations requiring it)**
  - Pass user credentials to `samba-tool` via `-U username%password` flag
  - Works for: user creation, password reset, group management, DNS, GPO, FSMO transfer
  - User's session stores their encrypted credentials (encrypted with server-side key, never plaintext at rest)
  - Credentials cleared on logout or session timeout

  **Recommended: Hybrid A+B**
  - Simple attribute edits (description, department, title, phone): LDAP modify as user
  - Complex operations (create user, reset password, GPO, FSMO): samba-tool as user
  - All writes audited with the acting user's identity

### Authentication Flow

```
1. User navigates to https://sambmin.dzsec.net
2. If Kerberos ticket available (SPNEGO):
   - nginx passes Negotiate header to Go API
   - Go API validates via gokrb5, creates session
3. Otherwise, login form:
   - User enters AD username + password
   - Go API performs LDAP bind to verify credentials
   - On success: creates session, stores encrypted creds for write operations
4. Session cookie set (HttpOnly, Secure, SameSite=Strict)
5. Reads: always via service account (fast, pooled)
6. Writes: via user's credentials (LDAP bind or samba-tool -U)
7. Logout: clear session + encrypted creds
```

### RBAC Model

Map AD group membership to Sambmin roles:

| Role | AD Group | Can Do |
|------|----------|--------|
| **Full Admin** | Domain Admins, Enterprise Admins | Everything: create/delete users, groups, OUs, DNS, GPO, FSMO, schema |
| **User Admin** | Account Operators, custom group | Create/edit/delete users, reset passwords, manage group membership |
| **DNS Admin** | DnsAdmins, custom group | Create/edit/delete DNS zones and records |
| **Help Desk** | custom group | Reset passwords, unlock accounts, view all objects |
| **Read Only** | Domain Users (default) | View everything, edit nothing |

Roles are checked in the Go API middleware. The AD itself also enforces ACLs — if a user tries to modify an object they don't have AD permissions on, the LDAP modify or samba-tool command will fail with an access denied error, which we surface cleanly.

### Write Operations by Object Type

**Users**
| Operation | Method | Command/Action |
|-----------|--------|----------------|
| Create user | samba-tool | `samba-tool user create <username> <password> --given-name=... --surname=... -U user%pass` |
| Edit attributes | LDAP modify | Direct LDAP modify as user (displayName, department, title, etc.) |
| Delete user | samba-tool | `samba-tool user delete <username> -U user%pass` |
| Reset password | samba-tool | `samba-tool user setpassword <username> --newpassword=... -U user%pass` |
| Enable/disable | LDAP modify | Toggle userAccountControl bit 0x0002 |
| Unlock account | LDAP modify | Set lockoutTime to 0 |
| Move to OU | LDAP moddn | LDAP ModifyDN operation |
| Add to group | LDAP modify | Add member DN to group's member attribute |

**Groups**
| Operation | Method | Command/Action |
|-----------|--------|----------------|
| Create group | samba-tool | `samba-tool group add <name> --group-type=... -U user%pass` |
| Delete group | samba-tool | `samba-tool group delete <name> -U user%pass` |
| Edit description | LDAP modify | Direct LDAP modify |
| Add member | samba-tool | `samba-tool group addmembers <group> <user> -U user%pass` |
| Remove member | samba-tool | `samba-tool group removemembers <group> <user> -U user%pass` |

**Computers**
| Operation | Method | Command/Action |
|-----------|--------|----------------|
| Delete computer | LDAP delete | LDAP delete as user |
| Disable computer | LDAP modify | Toggle userAccountControl |
| Reset machine password | samba-tool | `samba-tool computer reset <name> -U user%pass` |

**OUs / Containers**
| Operation | Method | Command/Action |
|-----------|--------|----------------|
| Create OU | samba-tool | `samba-tool ou create <dn> -U user%pass` |
| Delete OU | samba-tool | `samba-tool ou delete <dn> -U user%pass` (fails if children exist) |
| Rename OU | LDAP moddn | LDAP ModifyDN |
| Move objects into OU | LDAP moddn | LDAP ModifyDN for each object |

**DNS**
| Operation | Method | Command/Action |
|-----------|--------|----------------|
| Create zone | samba-tool | `samba-tool dns zonecreate <server> <zone> -U user%pass` |
| Delete zone | samba-tool | `samba-tool dns zonedelete <server> <zone> -U user%pass` |
| Create record | samba-tool | `samba-tool dns add <server> <zone> <name> <type> <value> -U user%pass` |
| Update record | samba-tool | `samba-tool dns update <server> <zone> <name> <type> <old> <new> -U user%pass` |
| Delete record | samba-tool | `samba-tool dns delete <server> <zone> <name> <type> <value> -U user%pass` |

### Confirmation Tiers for Write Operations

**Type-to-confirm** (irreversible, high-impact):
- Delete OU with children
- FSMO role transfer/seize
- Schema modifications
- Delete DNS zone

**Summary confirmation** (modal with details):
- Delete user/group/computer
- Disable accounts
- Bulk operations (enable/disable/delete multiple)

**Inline with undo** (low-risk, easily reversed):
- Edit description, department, title
- Change DNS record TTL
- Add/remove group member

### Implementation Priority

1. **Authentication** — Login form with LDAP bind, session management, logout
2. **Password reset** — Most requested admin action, high value
3. **Enable/disable/unlock** — Quick account management
4. **User attribute editing** — Department, title, description in the drawer
5. **Group membership** — Add/remove members
6. **User creation** — Full create workflow
7. **DNS record CRUD** — Create/update/delete records
8. **OU management** — Create/delete/move
9. **Bulk operations** — Multi-select actions
10. **Computer management** — Delete/disable machine accounts
