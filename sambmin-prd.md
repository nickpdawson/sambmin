# Sambmin — Product Requirements Document

## Web-Based Samba Active Directory Management Tool

**Version:** 1.0
**Date:** February 18, 2026
**Author:** Nick Dawson, CTO / Head of DevOps, DZsec
**License:** GPLv3 (matching Samba's license)
**Status:** Active Development — M13 (Write Operations Debugging)

---

## 1. Executive Summary

Sambmin is a world-class, web-based management tool for Samba Active Directory Domain Controllers. It provides full parity with Windows RSAT/ADUC administrative capabilities through a modern, secure, and delightful interface — while adding features that Windows tools lack entirely, such as DNS aging visualization, cross-DC consistency checking, certificate management integration, and a self-service user portal.

No production-ready web tool for Samba AD administration currently exists. Existing options are deprecated (SWAT), partial (identidude, samba4-manager), or stalled (Cockpit plugin). Sambmin fills this gap for the growing community of organizations running Samba AD on FreeBSD and Linux.

### Target Environment

- 3–6 Domain Controllers across 2–4 sites
- FreeBSD and Linux DC operating systems
- Both Samba internal DNS and BIND9 DLZ backends
- Both Heimdal and MIT Kerberos implementations
- Multi-VLAN enterprise networks with pfSense routing

### Design Philosophy: "Calm Power"

The interface feels like a quiet room full of sharp instruments. Everything is within reach, nothing is shouting for attention, and the interface trusts the user to be competent while protecting them from irreversible mistakes. We steal Windows RSAT's feature completeness and throw away its interaction model entirely.

---

## 2. Architecture

### 2.1 Tech Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Backend API | Go (net/http, gokrb5, go-ldap, pgx) | Static binaries, native Kerberos + LDAP, cross-compile from macOS to FreeBSD |
| Utility Scripts | Python 3.11+ | Wraps samba-tool, ldbtools, BIND9 utilities |
| Frontend | React 18 + TypeScript + Vite | Modern tooling, strong typing, fast HMR |
| UI Components | Ant Design 5 (ProComponents) | Enterprise admin components, ProTable, theming |
| Data Visualization | D3.js + React | Replication topology, group membership graphs, OU trees |
| Database | PostgreSQL 15+ | Audit logs, sessions, app config (never AD data) |
| Reverse Proxy | nginx | TLS termination, SPNEGO, static serving |
| Process Management | FreeBSD rc.d / systemd | Native, no extra dependencies |
| TLS | Let's Encrypt or local CA | User-configurable during setup |

### 2.2 System Architecture

```
Browser (Kerberos ticket or user/pass)
    │
    ▼
nginx (TLS termination, optional SPNEGO)
    │
    ▼
Go API Server (port 8443)
    ├── Direct LDAP reads to Samba DC(s) via connection pool
    ├── gokrb5 for Kerberos authentication
    ├── Calls Python scripts for samba-tool write operations
    ├── PostgreSQL for app data (audit, sessions, config)
    │
    ▼
Samba AD DC(s) — LDAP / RPC / DNS
```

### 2.3 Key Architectural Decisions

1. **LDAP for reads, samba-tool for writes** — Reading is fast via direct LDAP. Writing uses samba-tool (via Python wrappers) because it handles SID allocation, group nesting rules, password policy enforcement, and schema validation.

2. **Two-tier authentication** — Service account for all reads (pooled, fast). User's own credentials for writes (LDAP bind or samba-tool -U). Writes are audited under the acting user's identity.

3. **PostgreSQL for app data only** — AD data stays in Samba's LDB/LDAP. PostgreSQL stores audit logs, user preferences, session tokens, RBAC overrides, and application configuration.

4. **DNS backend abstraction** — Detects Samba internal DNS vs BIND9 DLZ per zone and routes operations accordingly.

5. **Kerberos abstraction** — Authentication and KDC management abstract over Heimdal and MIT implementations with detection at startup.

6. **Multi-DC awareness** — Connects to configurable primary DC with failover. Users can explicitly select a DC. Replication status gathered by querying all DCs.

---

## 3. Authentication & Authorization

### 3.1 Authentication Flow

1. Browser navigates to https://sambmin.dzsec.net
2. nginx checks for Kerberos ticket (SPNEGO Negotiate header)
   - Valid ticket → passes authenticated principal to Go API via header
   - No ticket → Go API serves login form
3. Login form accepts: username + password (LDAP simple bind to Samba DC)
   - Supports sAMAccountName, UPN, and DN formats
4. Go API creates session (secure cookie), stores encrypted credentials for writes
5. All subsequent requests authenticated via session token
6. Session timeout configurable (default 8 hours)
7. Logout clears session + encrypted credentials

### 3.2 RBAC Model

| Role | AD Group | Capabilities |
|------|----------|-------------|
| Full Admin | Domain Admins, Enterprise Admins | Everything: CRUD all objects, DNS, GPO, FSMO, schema, trusts |
| User Admin | Account Operators, custom group | Create/edit/delete users, reset passwords, manage group membership |
| DNS Admin | DnsAdmins, custom group | Create/edit/delete DNS zones and records |
| Help Desk | Custom group | Reset passwords, unlock accounts, view all objects |
| Read Only | Domain Users (default) | View everything, edit nothing |

Roles checked in Go API middleware. AD itself also enforces ACLs — if a user lacks AD permissions, the operation fails with an access denied error surfaced cleanly in the UI.

### 3.3 Self-Service Portal

Authenticated non-admin users can:
- Change their own password (with policy validation feedback)
- Edit user-writable profile fields: phone, mobile, department, title, office, address
- View their own group memberships
- View their own certificate status (published certs in userCertificate attribute)
- View their own account status (lockout, expiration, last logon)

Self-service operations use the user's own credentials (already available from session). No elevated privileges required.

### 3.4 Security Measures

- TLS everywhere — nginx terminates TLS (Let's Encrypt or local CA)
- CSRF protection — SameSite cookies + CSRF token for mutations
- Rate limiting — Login attempts rate-limited per IP
- Network binding — Go API binds to localhost; nginx handles external access
- Secrets management — Service account credentials encrypted at rest (AES-256-GCM)
- Content Security Policy — Strict CSP headers to prevent XSS
- Audit trail — Every mutation logged: who, what, when, from where, which DC, success/failure

---

## 4. User Interface

### 4.1 Design Principles

1. **Progressive Disclosure, Not Progressive Hiding** — Common actions prominent, advanced always visible but secondary. No "Advanced Settings" buttons hiding critical features.

2. **Context Over Navigation** — Viewing a user? See their groups inline in the drawer. Every object is a hub connecting to related objects.

3. **Destructive Actions Require Intention** — Three tiers: type-to-confirm (FSMO transfer, schema changes, delete OU with children), summary confirmation (delete user, disable accounts), inline undo (edit description, change TTL).

4. **The Interface Is a Search Engine** — Command palette (Cmd+K) is the primary power-user navigation. Prefixes: > actions, @ users, # groups, : DNS records.

5. **Show System Health, Not Just Objects** — Dashboard foregrounds replication health, DNS consistency, DC status. The admin should never be surprised by a problem they could have seen.

6. **Respect the Terminal** — Show equivalent samba-tool commands. Support CSV/JSON import/export. One-click copy for DNs, SIDs, UPNs, IP addresses.

### 4.2 Visual Design

- **Inspiration:** Linear's density, Vercel's typography, Grafana's monitoring
- **Typography:** Inter (body), JetBrains Mono (DNs, SIDs, IPs, LDAP filters, CLI commands)
- **Color:** Restrained — blue for interactive elements only, status colors (green/amber/red) for actual status only
- **Dark mode:** First-class, respects system preference, toggle in sidebar footer
- **Spacing:** 8px base grid
- **Tables:** Ant Design ProTable with inline editing, row selection, floating batch action bar, column toggles, virtual scrolling for 10K+ rows

### 4.3 Navigation Structure

```
[Sambmin logo]
[Domain selector dropdown]

--- Search (Cmd+K) ---

OVERVIEW
  Dashboard

DIRECTORY
  Users
  Groups
  Computers
  Contacts
  Organizational Units

INFRASTRUCTURE
  DNS
  Sites & Services
  Replication

POLICY & SECURITY
  Group Policy
  Kerberos & SPNs
  Authentication Policies
  FSMO Roles
  Schema
  Trusts

OPERATIONS
  Service Accounts
  Certificates
  Delegation

SYSTEM
  Audit Log
  Backup & Restore
  Settings
  Self-Service Portal (if non-admin)
```

Left sidebar, collapsible to icon-only. List views use right-side drawers (640px) for detail/edit. DNS and Replication use full-page views due to data density.

### 4.4 Key Interaction Patterns

- **Object detail:** Right-side drawer keeps list context visible
- **Bulk operations:** Checkbox selection + floating bottom action bar (Linear-style)
- **DNS editing:** Inline table editing, no modals for simple changes
- **Replication topology:** Interactive D3.js force-directed graph with site boundaries
- **Group membership:** Directed graph showing nesting chains
- **Confirmations:** Tiered by severity (type-to-confirm / summary / inline undo)
- **Loading states:** Skeleton loaders matching table row height, never blank screens
- **Empty states:** Illustration + clear call-to-action, never blank tables
- **Timestamps:** Relative by default ("5m ago"), absolute on hover
- **CLI equivalent:** Every write operation shows the samba-tool command that would achieve the same result

---

## 5. Feature Specifications

### 5.1 Dashboard

**DC Health Strip**
- One card per DC, color-coded: green (healthy), amber (degraded), red (unreachable)
- Shows: hostname, site, FSMO roles held, last replication time, LDAP response time
- Click to drill into DC detail

**Alert Banners (conditional)**
- Replication lag exceeding threshold
- Locked accounts count
- DNS diagnostic failures
- Degraded zpool on Matterhorn (custom integration)
- Certificate expiration warnings

**Quick Action Cards**
- Create User, Reset Password, Create DNS Record, Unlock Account
- Each links to the relevant creation form or action modal

**Domain Metrics with Deltas**
- Total users, computers, groups, DNS zones, locked accounts, disabled accounts
- Show change from previous day/week

**Recent Activity Timeline**
- Objects modified in last 24 hours from LDAP whenChanged
- Success/failure indicators
- Actor identity where available

**Replication Topology Mini-Map**
- Simplified D3.js force-directed graph
- Click to expand to full Replication page

---

### 5.2 Users

**List View**
- ProTable with sortable, filterable columns: display name, sAMAccountName, email, department, title, status, last logon, group count
- Tab filters: All | Active | Disabled | Locked Out | Recently Created | Expiring Soon
- Badge counts on each tab
- Row selection with floating bulk action bar

**User Detail Drawer (640px)**
Tabbed layout with the following sections:

*Identity Tab*
- Display name, sAMAccountName, UPN, DN (monospace, copy-to-clipboard)
- objectGUID, objectSid (monospace, copy-to-clipboard)
- whenCreated, whenChanged timestamps
- Profile photo placeholder

*Organization Tab*
- Title, department, company, manager (linked to manager's user object)
- Office, phone, mobile, fax, IP phone (with "Others" multi-value support)
- Address: street, city, state, zip, country

*Account Tab*
- Account status: enabled/disabled toggle
- Account expiration date picker
- Lockout status with time and count
- Password last set, password expiration countdown
- Logon hours matrix (168-bit bitmap — visual weekly grid editor)
- "Log On To" computer restrictions
- Primary group display and setter
- userAccountControl flags display (human-readable)
- Authentication policy and silo assignment

*Groups Tab*
- Direct group memberships list
- "Add to Group" with typeahead search
- Remove from group with inline confirmation
- Show primary group

*Profile Tab*
- Profile path, logon script
- Home folder: local path or mapped drive (letter + UNC path)

*Certificates Tab*
- Published certificates from userCertificate attribute
- For each cert: subject, issuer, serial, validity dates, key usage
- Visual status: valid (green), expiring soon (amber), expired (red)
- Integration note: certificates pushed via OpenXPKI SCEP / MDM enrollment
- Download certificate as PEM/DER

*Dial-in / RADIUS Tab*
- msNPAllowDialin, msRADIUSServiceType attributes
- Relevant for FreeRADIUS integration

*Object Tab (Advanced)*
- Raw attribute viewer: all attributes with types and values
- Edit any writable attribute with schema-based validation
- Binary attribute hex display, decoded SID display for SID-type attributes
- DN-type attributes are clickable links to that object

**User Actions**
- Create user (drawer form with progressive disclosure)
- Edit user attributes (inline in drawer)
- Rename user (CN, sAMAccountName, UPN, display name)
- Delete user (summary confirmation)
- Reset password (modal with policy validation + generate button + must-change-at-next-logon option)
- Enable / disable account
- Unlock account
- Move to OU (OU picker tree)
- Set expiration date
- Set/remove authentication policy and silo
- Show equivalent samba-tool command for all operations

**Bulk Operations (floating action bar)**
- Enable, disable, delete, move to OU, add to group, set expiration
- CSV/JSON import with dry-run validation
- CSV/JSON/LDIF export

---

### 5.3 Groups

**List View**
- ProTable: name, type (security/distribution), scope (domain local/global/universal), member count, description
- Tabs: All | Security | Distribution
- Search with type-ahead

**Group Detail Drawer**
- Identity: name, DN, SID, type, scope, description, whenCreated/Changed
- Members list with add/remove (typeahead search)
- Member Of (groups this group belongs to)
- Nested group membership visualization (D3.js directed graph)
- Effective membership resolution: show all transitive members with nesting path

**Group Actions**
- Create group (type, scope, name, description, initial members)
- Edit group (description, mail-address)
- Rename group (sAMAccountName, CN)
- Delete group (summary confirmation)
- Add/remove members
- Move to OU
- View group statistics (samba-tool group stats)

---

### 5.4 Computers

**List View**
- ProTable: name, DNS hostname, operating system, OS version, status, last logon, site, managed by
- Tabs: All | Active | Disabled | Stale (no logon in N days)

**Computer Detail Drawer**
- Identity: name, DN, SID, DNS hostname
- OS: operatingSystem, operatingSystemVersion, operatingSystemServicePack
- Network: IP address(es) from DNS, site membership
- Account: status, last logon, password last set, userAccountControl flags
- Managed By field
- Location field
- SPN list
- Delegation settings
- Authentication policy/silo assignment

**Computer Actions**
- Add computer (name, OU, description, IP address, SPNs)
- Edit computer attributes
- Delete computer (LDAP delete)
- Disable computer (UAC toggle)
- Reset machine password
- Move to OU
- DNS cleanup: remove all DNS records for this computer (samba-tool dns cleanup)

---

### 5.5 Contacts

**List View**
- ProTable: display name, email, phone, department, company
- Search with type-ahead

**Contact Detail Drawer**
- Name: given name, initials, surname, display name
- Contact info: email, phone, mobile, office
- Organization: department, company, job title
- Location

**Contact Actions**
- Add contact (name, email, phone, org fields)
- Edit contact
- Rename contact (given-name, surname, initials, display-name)
- Delete contact
- Move to OU

---

### 5.6 Organizational Units

**Dual View: List / Tree Toggle**
- List: flat ProTable of all OUs with path, object count, description
- Tree: interactive tree on left, contents on right (tree-table hybrid)

**OU Detail Drawer**
- Identity: name, DN, description
- Contents: count of users, groups, computers, contacts, child OUs
- Delegation: display current delegations (via dsacl get)
- Protection: accidental deletion protection status

**OU Actions**
- Create OU (name, description, parent OU)
- Rename OU
- Delete OU (warn if children exist; type-to-confirm for force-subtree-delete)
- Move OU
- List all objects in OU (with recursive option)
- Move objects into OU (drag-drop in tree or bulk move)

---

### 5.7 DNS Management

DNS is one of the most opaque parts of Samba AD administration. Sambmin aims to make it fully transparent.

#### 5.7.1 DNS Server Info

- Display DNS server configuration from samba-tool dns serverinfo: version, boot method, zone count, forest/domain names
- Current forwarder configuration (from smb.conf dns forwarder parameter)
- Forwarder reachability test (DNS query to each forwarder)
- Root hints display (samba-tool dns roothints)
- Clear banner for Samba DNS limitations: conditional forwarders not supported, zone transfers not supported for internal DNS, forwarder changes require smb.conf edit

#### 5.7.2 Zone Management

**Zone List View**
- ProTable: zone name, type (forward/reverse), backend (Samba internal / BIND9 DLZ), record count, dynamic update policy, aging status, SOA serial
- Tabs: All | Forward Lookup | Reverse Lookup | ForestDNSZones | DomainDNSZones

**Zone Properties Panel**
- Full zone info from samba-tool dns zoneinfo: type, flags, update policy, serial, DNSNODE count
- Aging/scavenging configuration:
  - Aging enabled/disabled toggle
  - No-refresh interval (hours)
  - Refresh interval (hours)
  - Aging-enabled timestamp
  - Scavenging servers
  - Visual timeline showing when records become eligible for scavenging
- Dynamic update policy: None | Nonsecure | Secure (editable)
- SOA record editor with dedicated form (7-field: nameserver, email, serial, refresh, retry, expire, minimum-ttl)

**Zone Actions**
- Create zone (forward or reverse)
  - Reverse zone helper: enter subnet (e.g., 10.15.1.0/24) → auto-generates in-addr.arpa name
- Delete zone (type-to-confirm)
- Configure aging/scavenging (samba-tool dns zoneoptions)
- Bulk operations on aging:
  - Mark old records static by date (--mark-old-records-static)
  - Mark records static/dynamic by regex pattern
  - Dry-run preview before applying

#### 5.7.3 Record Management

**Record List View (Full Page)**
- Tabbed record type filters: All | A/AAAA | CNAME | MX | SRV | TXT | NS | PTR | SOA
- Columns: name, type, data, TTL, static/dynamic indicator, timestamp (for dynamic records), age
- Inline table editing: click to edit value, TTL, or static/dynamic flag
- Type-adaptive creation form: fields change based on record type (e.g., MX shows priority + hostname, SRV shows priority + weight + port + target)
- Visual distinction between static records (no timestamp) and dynamic records (with timestamp and age display)
- Warning banner on dynamic records: "This record was registered dynamically and may be overwritten by the client"

**Record Actions**
- Create record (all types: A, AAAA, CNAME, MX, SRV, TXT, NS, PTR)
- Update record (samba-tool dns update with old/new value syntax)
- Delete record (confirmation with type/value display)
- Toggle static/dynamic
- Bulk operations:
  - Delete multiple records
  - Change TTL for multiple records
  - Bulk create PTR from selected A records
  - Find-and-replace in record data (e.g., IP migration)
- Export zone as BIND-format zone file
- Import records from CSV

**CLI Equivalent Display**
Every DNS operation shows the exact samba-tool dns command.

#### 5.7.4 DNS Query Tool

- "Query DNS from DC" panel: select a DC, enter name + type
- Queries via samba-tool dns query (RPC-based, not UDP DNS)
- Shows what the DC's database contains vs what resolves via dig
- Side-by-side comparison across multiple DCs for the same query

#### 5.7.5 DNS Diagnostics

**AD SRV Record Validator (comprehensive)**
Checks all required AD service records per site, per DC:
- _ldap._tcp.dc._msdcs.\<domain\>
- _kerberos._tcp.dc._msdcs.\<domain\>
- _ldap._tcp.\<site\>._sites.dc._msdcs.\<domain\> (per site)
- _kerberos._tcp.\<site\>._sites.dc._msdcs.\<domain\> (per site)
- _ldap._tcp.gc._msdcs.\<forest\>
- _gc._tcp.\<forest\>
- _ldap._tcp.pdc._msdcs.\<domain\>
- _kpasswd._tcp.\<domain\>
- _kpasswd._udp.\<domain\>

Display: matrix of DCs × SRV record types with pass/fail status.

**Cross-DC DNS Consistency Checker**
- Query same records across all DCs and diff results
- Flag records existing on some DCs but not others
- Show DnsNode LDAP metadata: which DC last modified, USN
- Compare zone record counts across DCs

**Stale Record Analysis**
- List all dynamic records sorted by age
- Highlight records exceeding scavenging threshold
- One-click cleanup for records associated with decommissioned hosts (samba-tool dns cleanup)
- Orphaned PTR detection: PTR records pointing to nonexistent A records

**Additional Checks**
- Missing reverse PTR records for A records
- TTL inconsistency detection within zones
- SOA serial comparison across DCs
- DNS resolution test from each DC (forward + reverse for key hosts)
- Forwarder reachability test

**DNS Change Metadata**
- Surface whenChanged, uSNChanged for DNS record LDAP objects
- "Who changed this record?" audit capability

---

### 5.8 Sites & Services

**Site Management**
- List all sites (samba-tool sites list)
- Create / remove sites
- Site detail: DCs in site, subnets assigned, site links

**Subnet Management**
- List subnets per site (samba-tool sites subnet list)
- Create subnet (subnet + site assignment)
- Remove subnet
- Change subnet site assignment (samba-tool sites subnet set-site)
- View subnet details

**Site Links**
- List site links with cost, schedule, transport
- Edit site link properties
- Visual site topology (D3.js graph with sites as clusters, links as weighted edges)

---

### 5.9 Replication

**Topology Visualization**
- D3.js force-directed graph with site grouping
- Nodes = DCs (color by site, icon by FSMO role), edges = replication links
- Data from samba-tool visualize (ntdsconn, reps, uptodateness modes)
- Interactive: drag nodes, zoom, click for detail

**Replication Status Table**
- Per-partnership: source DC, destination DC, naming context, last sync, pending changes, status, last error
- Data from samba-tool drs showrepl

**LDAP Compare Tool**
- Compare two LDAP databases across partitions (samba-tool ldapcmp)
- Partition selector: domain, configuration, schema, dnsdomain, dnsforest
- Diff display with side-by-side comparison

**Replication Actions**
- Force sync per partnership (samba-tool drs replicate)
- Trigger KCC run (samba-tool drs kcc)
- Query/change NTDS Settings options (samba-tool drs options)

**Uptodateness Matrix**
- DC × DC matrix showing replication lag (from samba-tool visualize uptodateness --distance)
- Color-coded: green (current), amber (lagging), red (significantly behind)

---

### 5.10 Group Policy (GPOs)

**GPO List View**
- All GPOs with name, GUID, status, linked containers, created/modified dates

**GPO Detail**
- GPO information (samba-tool gpo show)
- Linked containers (samba-tool gpo listcontainers)
- Policy settings browser

**GPO Actions**
- Create GPO (samba-tool gpo create)
- Delete GPO (samba-tool gpo del) — type-to-confirm
- Link/unlink GPO to container (samba-tool gpo setlink / dellink)
- Get/set inheritance on containers (samba-tool gpo getinheritance / setinheritance)
- Fetch/download GPO (samba-tool gpo fetch)
- List GPOs for a specific user/computer (samba-tool gpo list)

**Samba VGP Extensions**
These are Samba-specific GPO extensions for managing Unix/Linux clients — critical for FreeBSD DC environments:
- Sudoers policies: add/list/remove (samba-tool gpo manage sudoers)
- OpenSSH settings: list/set (samba-tool gpo manage openssh)
- Startup scripts: add/list/remove (samba-tool gpo manage scripts startup)
- Symbolic links: add/list/remove (samba-tool gpo manage symlink)
- Files deployment: add/list/remove (samba-tool gpo manage files)
- MOTD (message of the day): list/set (samba-tool gpo manage motd)
- Login issue banner: list/set (samba-tool gpo manage issue)
- Host access control: add/list/remove (samba-tool gpo manage access)

**GPO Limitations Banner**
Clear messaging about Samba GPO limitations vs Windows: Samba serves GPOs via SYSVOL but does not enforce GPO restrictions on DCs themselves. Password policies use samba-tool domain passwordsettings, not GPO.

---

### 5.11 Kerberos & SPNs

**Kerberos Backend Detection**
- Auto-detect Heimdal vs MIT at startup
- Display current backend and version

**SPN Management**
- List SPNs for any user/computer (samba-tool spn list)
- Add SPN (samba-tool spn add) with duplicate detection across forest
- Delete SPN (samba-tool spn delete)
- SPN search: find all objects with a specific SPN pattern

**Keytab Management**
- Generate keytab for a service principal
- Download keytab file
- Domain keytab export (samba-tool domain exportkeytab)

**Kerberos Delegation Management**
- View delegation settings for any account (samba-tool delegation show)
- Add/remove allowed-to-delegate-to services (samba-tool delegation add-service / del-service)
- Toggle unconstrained delegation (samba-tool delegation for-any-service on/off)
- Toggle protocol transition / S4U2Proxy (samba-tool delegation for-any-protocol on/off)

**KDC Diagnostics**
- Ticket issuance test: attempt to get a TGT for a test principal
- Encryption type configuration display
- Ticket policy display

---

### 5.12 Authentication Policies & Silos

This is a Samba 4.15+ feature providing advanced Kerberos authentication controls.

**Authentication Policies**
- List all policies (samba-tool domain auth policy list)
- View policy details (samba-tool domain auth policy view)
- Create policy with options:
  - Strong NTLM policy (Disabled/Optional/Required)
  - User TGT lifetime
  - Service TGT lifetime
  - Computer TGT lifetime
  - Allowed-to-authenticate-from conditions (per account type)
  - Allowed-to-authenticate-to conditions (per account type)
  - NTLM auth exceptions
- Modify policy
- Delete policy (with protection awareness)
- Assign policy to users (samba-tool user auth policy assign)
- View/remove policy assignment from users

**Authentication Silos**
- List all silos (samba-tool domain auth silo list)
- View silo details
- Create silo with policies for user/service/computer accounts
- Grant/revoke silo membership (samba-tool domain auth silo member grant/revoke)
- List silo members
- Enforce/audit mode toggle

---

### 5.13 FSMO Roles

**Role Display**
- Visual display of all 5 FSMO roles and which DC holds each:
  - Schema Master
  - Domain Naming Master
  - PDC Emulator
  - RID Master
  - Infrastructure Master
- Data from samba-tool fsmo show

**Role Transfer**
- Transfer workflow with type-to-confirm (samba-tool fsmo transfer)
- Target DC selector
- Pre-transfer health check: verify target DC is reachable and replication is current

**Role Seizure**
- Emergency seize workflow with strong warnings (samba-tool fsmo seize)
- Type-to-confirm with explicit warning text
- Only available to Full Admin role

---

### 5.14 Schema

**Schema Browser**
- Attribute viewer: display definition, syntax, single/multi-value, indexed (samba-tool schema attribute show)
- Show which objectClasses contain an attribute (samba-tool schema attribute show_oc)
- ObjectClass viewer: display definition, must/may attributes, hierarchy (samba-tool schema objectclass show)
- Search/filter attributes and classes

**Schema Modification**
- Modify attribute behavior (samba-tool schema attribute modify)
- Type-to-confirm for all schema changes — schema modifications are irreversible

---

### 5.15 Trust Relationships

**Trust List**
- All trusts with type, direction, status (samba-tool domain trust list)

**Trust Detail**
- Show trust details (samba-tool domain trust show)
- Trust namespaces (samba-tool domain trust namespaces)

**Trust Actions**
- Create trust (samba-tool domain trust create) — type-to-confirm
- Modify trust (samba-tool domain trust modify)
- Delete trust (samba-tool domain trust delete) — type-to-confirm
- Validate trust (samba-tool domain trust validate)

**Trust Limitations Banner**
Samba trust support is experimental. Clear display of known limitations: cannot add users/groups from trusted domain into domain groups.

---

### 5.16 Service Account Management

Dedicated view for managing service accounts with specialized tooling.

**Service Account Discovery**
- Filter by OU (e.g., Service Accounts OU)
- Filter by naming convention pattern
- Filter by SPN presence
- Filter by delegation flags

**Service Account Detail**
- All standard user properties
- SPN list with add/remove
- Delegation configuration
- Keytab generation and download
- Password age and rotation tracking
- Authentication policy/silo assignment

**Service Account Actions**
- Create service account (pre-filled with service account defaults: password never expires, etc.)
- Manage SPNs
- Configure delegation (constrained, unconstrained, protocol transition)
- Generate keytab
- Rotate password

---

### 5.17 Certificate Viewer

Surface certificate data stored in AD, with awareness of the DZsec PKI infrastructure (OpenXPKI, SCEP, MDM).

**Per-User Certificate View**
- Read userCertificate attribute (binary DER-encoded X.509)
- Decode and display: subject, issuer, serial number, validity (not before / not after), key usage, extended key usage, SAN
- Visual status: valid (green), expiring within 30 days (amber), expired (red)
- Certificate chain display where available
- Download as PEM or DER

**Certificate Overview Dashboard**
- Count of users with published certificates
- Certificates expiring within 30/60/90 days
- Certificates by issuer (identify OpenXPKI-issued vs third-party)
- Expired certificate report

---

### 5.18 Delegation of Control

**View Current Delegations**
- Per-OU: display ACEs from dsacl get showing who has what permissions
- Human-readable translation of SDDL ACE entries

**Effective Permissions Viewer**
- For a given user + target object/OU: compute effective permissions considering group memberships and ACE inheritance

**Delegation Actions**
- Set ACLs on directory objects (samba-tool dsacl set)
- Delete ACLs (samba-tool dsacl delete)
- Common delegation templates: "Allow password reset on this OU", "Allow user creation on this OU", "Allow group membership management"

---

### 5.19 Password Policy Management

**Default Domain Password Policy**
- Display and edit (samba-tool domain passwordsettings show/set):
  - Minimum password length
  - Password complexity (on/off)
  - Password history length
  - Minimum password age
  - Maximum password age
  - Account lockout threshold
  - Account lockout duration
  - Account lockout observation window
  - Store passwords with reversible encryption (on/off)

**Fine-Grained Password Policies (PSOs)**
- List all PSOs (samba-tool domain passwordsettings pso list)
- Create PSO with name, precedence, and all policy settings
- Modify PSO
- Delete PSO
- Apply PSO to user or group (samba-tool domain passwordsettings pso apply)
- Unapply PSO from user or group
- Show which PSO applies to a specific user (samba-tool domain passwordsettings pso show-user)
- Show PSO details (samba-tool domain passwordsettings pso show)

**Password Policy Tester**
- "Would this password pass?" inline validator
- Select which policy (default or specific PSO) to test against
- Shows which rules pass/fail

---

### 5.20 Domain Management

**Domain Information**
- Domain and forest functional levels (samba-tool domain level show)
- Domain info by IP (samba-tool domain info)

**Functional Level Management**
- Raise domain/forest functional level (samba-tool domain level raise) — type-to-confirm
- Display prerequisites and warnings before raising

**Domain Backup & Restore**
- Online backup (samba-tool domain backup online) — creates tar of running DC's DB
- Offline backup (samba-tool domain backup offline) — with proper locking
- Backup with domain rename (samba-tool domain backup rename)
- Restore from backup (samba-tool domain backup restore) — type-to-confirm
- Scheduled backup capability (configurable cron integration)

**Database Health Check**
- Run AD database consistency check (samba-tool dbcheck)
- Display errors and warnings
- Option to auto-fix known issues

**Forest Configuration**
- dSHeuristics display and modification (samba-tool forest directory_service)
- Explanation of each heuristic flag

---

### 5.21 Advanced Search & LDAP Query Builder

**Visual Filter Builder**
- Drag-and-drop AND/OR/NOT groups
- Attribute picker (populated from schema)
- Operator selector (=, >=, <=, ~=, present, bitwise AND/OR)
- Value field with type-appropriate input
- Live preview of generated LDAP filter string

**Raw LDAP Filter Mode**
- Text input for power users
- Syntax highlighting and validation
- Common filter templates:
  - All users in department X
  - All disabled accounts
  - All accounts with password expiring in N days
  - All computers with OS matching pattern
  - All objects modified in last N hours
  - All accounts with specific UAC flags

**Attribute Chooser**
- Pick which attributes to return in results
- Default sets per object type

**Query Results**
- ProTable display with chosen attributes as columns
- Export to CSV, JSON, LDIF
- Save query as template with parameterization (fill in variables at runtime)
- Saved queries accessible from sidebar and command palette

---

### 5.22 Audit Log

**Log Viewer**
- Full-featured table with filtering:
  - By actor (who performed the action)
  - By action type (create, modify, delete, auth, password reset, etc.)
  - By object type (user, group, computer, DNS, GPO, etc.)
  - By object name/DN
  - By time range
  - By result (success/failure)
  - By DC (which domain controller)
  - By source IP
- Stored in PostgreSQL

**Log Actions**
- Export to CSV/JSON
- Retention policy configuration (auto-delete after N days)
- Search within log entries

---

### 5.23 Settings

**Connection Configuration**
- DC list with failover order
- Primary DC selector
- LDAP connection settings (TLS, port, base DN)
- Service account credentials (encrypted)
- Connection health test

**TLS Configuration**
- Current certificate status (issuer, expiry, SANs)
- Let's Encrypt renewal status
- CA certificate upload for local CA

**Authentication Settings**
- Kerberos/SPNEGO configuration
- LDAP bind configuration
- Session timeout

**RBAC Configuration**
- Role-to-AD-group mapping editor
- Self-service field whitelist (which attributes users can edit)

**Notification/Webhook Configuration**
- Webhook endpoints (HTTPS only, admin-configurable)
- Shared secret (HMAC signature on payload)
- Event type subscriptions (account lockout, replication failure, password reset, etc.)
- Rate limiting per webhook
- Payload format: event type, actor, object DN, timestamp (never passwords or sensitive data)
- Test webhook button

**Application Info**
- Sambmin version, Go version, Samba version
- Connected DCs and health
- Database connection status
- Uptime

---

### 5.24 Webhooks & Notifications

Webhooks are admin-only configurable with security constraints:

**Security Model**
- Only pre-configured HTTPS endpoints (no user-supplied URLs)
- HMAC-SHA256 signature on every payload using shared secret
- Payload never includes passwords, credentials, or sensitive attribute values
- Rate-limited: max N events per minute per endpoint
- Retry with exponential backoff on failure
- Dead letter queue for failed deliveries

**Supported Events**
- Account lockout (threshold exceeded)
- Replication failure
- DC unreachable
- Password reset performed
- User created/deleted
- Privileged group membership changed (Domain Admins, Enterprise Admins)
- DNS zone created/deleted
- Certificate expiring (< 30 days)
- FSMO role transfer/seizure

---

## 6. User Flows

### 6.1 New Employee Onboarding

1. Admin clicks "Create User" from dashboard or Users page
2. Progressive disclosure form: required fields first (name, username, password), then optional (department, title, manager, groups, OU)
3. Auto-generated username suggestion from first.last
4. Password generator with policy validation inline
5. OU picker (tree selector)
6. Group selector (typeahead, multiple)
7. Review panel showing all fields + equivalent samba-tool command
8. Submit → API creates user via samba-tool → success notification
9. Optionally: generate initial keytab, assign auth policy

### 6.2 Password Reset (Help Desk)

1. Help desk user searches for locked/requesting user via Cmd+K or Users page
2. Opens user detail drawer → sees account status
3. Clicks "Reset Password" in action menu
4. Modal: enter new password or click "Generate" for random compliant password
5. Toggle: "User must change password at next logon"
6. Confirm → API calls samba-tool user setpassword → success
7. Audit log captures: who reset, for whom, when, from which IP

### 6.3 Account Lockout Investigation

1. Dashboard shows "3 Locked Accounts" alert
2. Click alert → Users page filtered to Locked Out tab
3. Click user → drawer shows lockout time, lockout count
4. Check "Recent Activity" for failed auth attempts
5. If legitimate lockout: click "Unlock" → inline confirmation → done
6. If suspicious pattern: navigate to Audit Log filtered by that user

### 6.4 DNS Record Troubleshooting

1. Dashboard shows DNS diagnostic alert (e.g., "Missing SRV records")
2. Click → DNS Diagnostics page with detailed check results
3. SRV validator matrix shows which DCs are missing which records
4. Click failed check → pre-filled create record form
5. Use "Query DNS from DC" tool to verify inconsistency across DCs
6. Create/fix records → verify with cross-DC consistency checker

### 6.5 Replication Failure Response

1. Dashboard shows replication alert with red edge on topology mini-map
2. Click → Replication page with full topology
3. Identify failed link in status table (source DC, dest DC, error)
4. Click "Force Sync" on the affected partnership
5. Monitor uptodateness matrix for convergence
6. If persistent: use LDAP Compare to identify drift between DCs

### 6.6 Employee Offboarding

1. Search for departing user
2. Open user drawer → Actions menu → "Disable Account"
3. Summary confirmation with impact display
4. Remove from all groups (bulk remove in Groups tab)
5. Move to "Disabled Users" OU
6. Optionally: set account expiration date for deferred deletion
7. Audit trail shows complete offboarding sequence

### 6.7 Self-Service Password Change

1. Non-admin user logs in → sees Self-Service Portal in sidebar
2. Clicks "Change Password"
3. Form: current password, new password (with inline policy validator), confirm
4. Submit → API uses user's own credentials to change password
5. Success notification → session updated with new credentials

### 6.8 Bulk User Import

1. Admin navigates to Users → clicks "Import"
2. Upload CSV or JSON file
3. Dry-run validation: shows each row with pass/fail status
4. Fix errors in source file or skip failed rows
5. Execute import → progress indicator
6. Results summary: N created, M failed with reasons per failure
7. All created users appear in audit log

### 6.9 Service Account Provisioning

1. Navigate to Service Accounts
2. Click "Create Service Account"
3. Form pre-filled with service account defaults (password never expires, etc.)
4. Add SPNs (e.g., HTTP/webapp.dzsec.net)
5. Configure delegation if needed (constrained to specific services)
6. Set authentication policy/silo
7. Generate keytab → download for deployment to service host
8. Audit log captures all steps

### 6.10 DNS Aging Setup

1. Navigate to DNS → select zone → Zone Properties
2. Click "Configure Aging"
3. Form: enable aging, set no-refresh interval, set refresh interval
4. Preview: "Records older than X days will become eligible for scavenging"
5. Optionally: bulk mark critical records as static (by regex pattern, with dry-run)
6. Confirm → samba-tool dns zoneoptions applied
7. Dashboard shows aging status per zone going forward

---

## 7. Confirmation Tiers for Write Operations

### Type-to-Confirm (irreversible, high-impact)
- Delete OU with children (force-subtree-delete)
- FSMO role transfer or seizure
- Schema modifications
- Delete DNS zone
- Raise domain/forest functional level
- Delete trust relationship
- Domain restore from backup
- Authentication policy/silo deletion (if enforced)

### Summary Confirmation (modal with details)
- Delete user/group/computer/contact
- Disable accounts
- Bulk operations (enable/disable/delete multiple objects)
- Delete GPO
- Delete DNS records
- Remove group members

### Inline with Undo (low-risk, easily reversed)
- Edit description, department, title, phone, office
- Change DNS record TTL
- Add/remove group member
- Toggle static/dynamic on DNS record
- Edit contact fields

---

## 8. API Design

### 8.1 API Principles

- RESTful endpoints under /api/
- JSON request/response bodies
- Authentication via session cookie (set during login)
- RBAC enforced per endpoint in middleware
- All mutations return the modified object
- Error responses include: error code, human-readable message, samba-tool error output where applicable

### 8.2 Endpoint Summary

**Authentication**
- POST /api/auth/login — LDAP bind, create session
- POST /api/auth/logout — clear session
- GET /api/auth/me — current user info and roles

**Users**
- GET /api/users — list (with filters, pagination)
- GET /api/users/:id — detail
- POST /api/users — create
- PUT /api/users/:id — update attributes
- DELETE /api/users/:id — delete
- POST /api/users/:id/password — reset password
- POST /api/users/:id/enable — enable
- POST /api/users/:id/disable — disable
- POST /api/users/:id/unlock — unlock
- POST /api/users/:id/move — move to OU
- POST /api/users/:id/rename — rename
- GET /api/users/:id/certificates — list published certs
- POST /api/users/import — bulk import (CSV/JSON)
- GET /api/users/export — export (CSV/JSON/LDIF)

**Groups**
- GET /api/groups — list
- GET /api/groups/:id — detail with members
- POST /api/groups — create
- PUT /api/groups/:id — update
- DELETE /api/groups/:id — delete
- POST /api/groups/:id/members — add members
- DELETE /api/groups/:id/members/:member — remove member
- GET /api/groups/:id/effective-members — transitive membership
- GET /api/groups/stats — group statistics

**Computers**
- GET /api/computers — list
- GET /api/computers/:id — detail
- POST /api/computers — add
- DELETE /api/computers/:id — delete
- POST /api/computers/:id/disable — disable
- POST /api/computers/:id/move — move to OU
- POST /api/computers/:id/dns-cleanup — remove DNS records

**Contacts**
- GET /api/contacts — list
- GET /api/contacts/:id — detail
- POST /api/contacts — create
- PUT /api/contacts/:id — update
- DELETE /api/contacts/:id — delete
- POST /api/contacts/:id/move — move to OU

**OUs**
- GET /api/ous — list
- GET /api/ous/tree — tree structure
- GET /api/ous/:dn/objects — list objects in OU
- POST /api/ous — create
- DELETE /api/ous/:dn — delete
- POST /api/ous/:dn/move — move
- POST /api/ous/:dn/rename — rename

**DNS**
- GET /api/dns/serverinfo — server configuration
- GET /api/dns/zones — list zones
- GET /api/dns/zones/:zone — zone info
- POST /api/dns/zones — create zone
- DELETE /api/dns/zones/:zone — delete zone
- PUT /api/dns/zones/:zone/options — set aging/scavenging options
- GET /api/dns/zones/:zone/records — list records (filterable by type)
- POST /api/dns/zones/:zone/records — create record
- PUT /api/dns/zones/:zone/records — update record
- DELETE /api/dns/zones/:zone/records — delete record
- POST /api/dns/query — query a name from a specific DC
- GET /api/dns/diagnostics — run diagnostic checks
- GET /api/dns/consistency — cross-DC consistency check
- POST /api/dns/cleanup/:hostname — cleanup records for host
- GET /api/dns/roothints — root hints

**Replication**
- GET /api/replication/status — showrepl data
- GET /api/replication/topology — ntdsconn/reps visualization data
- GET /api/replication/uptodateness — uptodateness matrix
- POST /api/replication/sync — force sync
- POST /api/replication/kcc — trigger KCC
- POST /api/replication/compare — ldapcmp between two DCs

**Sites**
- GET /api/sites — list sites
- GET /api/sites/:name — site detail
- POST /api/sites — create site
- DELETE /api/sites/:name — remove site
- GET /api/sites/:name/subnets — list subnets
- POST /api/sites/subnets — create subnet
- DELETE /api/sites/subnets/:subnet — remove subnet
- PUT /api/sites/subnets/:subnet/site — change site assignment

**GPO**
- GET /api/gpo — list all GPOs
- GET /api/gpo/:id — GPO detail
- POST /api/gpo — create GPO
- DELETE /api/gpo/:id — delete GPO
- POST /api/gpo/:id/link — link to container
- DELETE /api/gpo/:id/link — unlink from container
- GET /api/gpo/manage/:type — list VGP extension entries
- POST /api/gpo/manage/:type — add VGP extension entry

**Kerberos & SPNs**
- GET /api/spn/:account — list SPNs
- POST /api/spn — add SPN
- DELETE /api/spn — delete SPN
- GET /api/delegation/:account — show delegation
- POST /api/delegation/:account/service — add allowed service
- DELETE /api/delegation/:account/service — remove allowed service
- POST /api/keytab/:account — generate keytab

**Auth Policies & Silos**
- GET /api/auth-policies — list
- POST /api/auth-policies — create
- PUT /api/auth-policies/:name — modify
- DELETE /api/auth-policies/:name — delete
- GET /api/auth-silos — list
- POST /api/auth-silos — create
- PUT /api/auth-silos/:name — modify
- DELETE /api/auth-silos/:name — delete
- POST /api/auth-silos/:name/members — grant membership
- DELETE /api/auth-silos/:name/members/:member — revoke membership

**FSMO**
- GET /api/fsmo — show roles
- POST /api/fsmo/transfer — transfer role
- POST /api/fsmo/seize — seize role

**Schema**
- GET /api/schema/attributes/:name — show attribute
- GET /api/schema/attributes/:name/objectclasses — show containing objectclasses
- GET /api/schema/objectclasses/:name — show objectclass
- PUT /api/schema/attributes/:name — modify attribute

**Trusts**
- GET /api/trusts — list
- GET /api/trusts/:domain — show trust
- POST /api/trusts — create trust
- PUT /api/trusts/:domain — modify trust
- DELETE /api/trusts/:domain — delete trust
- POST /api/trusts/:domain/validate — validate trust

**Password Policies**
- GET /api/password-policy — show default policy
- PUT /api/password-policy — set default policy
- GET /api/password-policy/pso — list PSOs
- POST /api/password-policy/pso — create PSO
- PUT /api/password-policy/pso/:name — modify PSO
- DELETE /api/password-policy/pso/:name — delete PSO
- POST /api/password-policy/pso/:name/apply — apply to user/group
- POST /api/password-policy/pso/:name/unapply — remove from user/group
- GET /api/password-policy/user/:username — show effective policy for user
- POST /api/password-policy/test — test password against policy

**Domain**
- GET /api/domain/info — domain and forest info
- GET /api/domain/level — functional levels
- POST /api/domain/level — raise functional level
- POST /api/domain/backup — create backup
- POST /api/domain/restore — restore from backup
- POST /api/domain/dbcheck — run database check

**DS ACLs**
- GET /api/dsacl/:dn — get ACL on object
- PUT /api/dsacl/:dn — set ACL
- DELETE /api/dsacl/:dn — delete ACE

**Search**
- POST /api/search — execute LDAP query with filter, attributes, base DN
- GET /api/search/saved — list saved queries
- POST /api/search/saved — save query template

**Audit**
- GET /api/audit — query audit log (filterable)
- GET /api/audit/export — export to CSV/JSON
- PUT /api/audit/retention — set retention policy

**Self-Service**
- GET /api/self — current user profile
- PUT /api/self — update writable fields
- POST /api/self/password — change own password

**Settings**
- GET /api/settings — all settings
- PUT /api/settings/:section — update settings section

---

## 9. Samba Feature Limitations

Sambmin should clearly surface these Samba limitations rather than silently omitting features:

| Feature | Samba Status | Sambmin Approach |
|---------|-------------|-----------------|
| DNS conditional forwarders | Not implemented | Info banner, explain limitation |
| DNS zone transfers | Not implemented (uses AD replication) | Info banner, note BIND9 can do it |
| DNS scavenging | Aging supported; auto-scavenging limited | Implement app-level scavenging job |
| DNS forwarder config | smb.conf only, not via RPC/LDAP | Display current config, explain change requires smb.conf edit |
| DNSSEC | Not supported | Info banner |
| Trust relationships | Experimental, limited | Warning banner with specific limitations |
| GPO enforcement on DCs | Not supported (serves via SYSVOL only) | Explain; use samba-tool passwordsettings instead |
| gMSA (Group Managed Service Accounts) | Partial support | Display if supported, note limitations |
| AD Recycle Bin | Supported at sufficient forest level | Implement if functional level allows |
| BitLocker recovery keys | Not applicable (Windows-specific) | Omit |

---

## 10. Development Phases

### Phase 1: Foundation (COMPLETED — M1–M12)
- Project scaffold, Go + React + Python structure
- Application shell: sidebar, dark mode, command palette, keyboard shortcuts
- Dashboard with DC health, metrics, activity timeline
- All directory pages: Users, Groups, Computers, OUs (list + detail drawers)
- Live LDAP integration with connection pooling and multi-DC failover
- Live DNS integration (samba-tool dns)
- Authentication system (LDAP bind, session management, encrypted credential storage)
- Write operations backend (user/group/computer/OU/DNS CRUD)
- Write operations frontend wiring
- nginx + TLS + production deployment

### Phase 2: Completeness (CURRENT — M13+)
- Debug and verify all write operations end-to-end
- User self-service portal (password change, profile editing)
- Full user properties (all tabs: organization, account, profile, certificates, dial-in, object)
- Contacts module (full CRUD)
- Computer add/move/rename
- Group/user rename workflows
- Advanced search / LDAP query builder with saved queries
- CSV/JSON/LDIF import and export
- Password policy management (default + PSOs)

### Phase 3: DNS Deep Dive
- DNS server info display with forwarder status
- Zone properties panel with full zoneinfo display
- Aging/scavenging configuration UI (zoneoptions)
- Static vs dynamic record distinction in record table
- DNS query tool (query from specific DC via RPC)
- SOA record dedicated editor
- Reverse zone creation helper
- Cross-DC DNS consistency checker
- Enhanced AD SRV record validator (per-site, per-DC matrix)
- Stale record analysis and cleanup
- DNS change metadata (whenChanged, uSNChanged on DnsNode objects)
- Bulk operations: PTR from A records, find-and-replace, zone export

### Phase 4: Infrastructure & Replication
- Replication topology D3.js visualization (ntdsconn, reps, uptodateness)
- Replication status table with per-partnership detail
- Force sync, KCC trigger, NTDS options
- Uptodateness matrix (DC × DC heatmap)
- LDAP compare tool (cross-DC diff)
- Sites & Services: site CRUD, subnet CRUD, site link management
- Visual site topology

### Phase 5: Policy & Security
- GPO management: full CRUD, link/unlink, inheritance, fetch
- Samba VGP extensions: sudoers, openssh, scripts, symlinks, files, motd, issue, access
- Kerberos & SPN management: SPN CRUD, keytab generation, delegation
- Authentication policies and silos: full CRUD, assign to users, membership management
- FSMO roles: display, transfer, seize
- Schema browser with attribute/objectclass exploration
- Trust management: CRUD, validate, namespaces
- DS ACLs / Delegation of control: view, set, delete, common templates

### Phase 6: Operations & Polish
- Service account management view
- Certificate viewer (userCertificate decode, expiry tracking, PKI integration awareness)
- Domain backup/restore
- Domain functional level management
- Database health check (dbcheck)
- Forest dSHeuristics configuration
- Audit log viewer with full filtering and export
- Webhook/notification system (admin-only, HTTPS-only, HMAC-signed)
- Saved searches and bookmarks
- Customizable dashboard layout
- Settings management UI
- API documentation (OpenAPI/Swagger)

---

## 11. Non-Functional Requirements

### Performance
- Dashboard load: < 2 seconds
- User list (1000 users): < 1 second with virtual scrolling
- DNS record list (10,000 records): < 2 seconds with virtual scrolling
- LDAP query response: < 500ms for paged results
- Write operations: < 3 seconds including samba-tool execution

### Security
- All traffic encrypted (TLS 1.2+)
- Session cookies: HttpOnly, Secure, SameSite=Strict
- CSRF tokens on all mutation endpoints
- Rate limiting on login (5 attempts per minute per IP)
- Encrypted credential storage in session (AES-256-GCM)
- Audit trail immutable by non-admin users
- No credentials in URL parameters or logs

### Reliability
- DC failover: automatic switch to next DC if primary unreachable
- Graceful degradation: read-only mode if all write-capable DCs are down
- Session persistence: survive API restart via PostgreSQL session store

### Compatibility
- Samba 4.15+ (required for aging/scavenging, auth policies)
- FreeBSD 14+ and Linux (Debian/Ubuntu, RHEL)
- Modern browsers: Chrome, Firefox, Safari, Edge (latest 2 versions)
- Desktop-first layout, functional on tablet

---

## 12. Success Metrics

- All samba-tool subcommands accessible via web UI
- Feature parity with Windows ADUC for core directory operations
- DNS management surpasses Windows DNS Manager for Samba-specific diagnostics
- Zero data loss from accidental operations (tiered confirmation system)
- Complete audit trail for all mutations
- Self-service portal reduces help desk password reset tickets
- Open-source release ready under GPLv3
