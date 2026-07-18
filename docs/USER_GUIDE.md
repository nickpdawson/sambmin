# Sambmin User Guide

A tour of what Sambmin does and where to find it. Detailed guides are linked
from each section.

Sambmin reads directory data directly over LDAP and performs writes through
`samba-tool` as the **logged-in user**, so every change is attributed to the
real actor. Access to each area is governed by [RBAC](#roles--access) derived
from AD group membership.

---

## Directory

- **[Users](features/user-management.md)** — full account lifecycle: create
  (in any OU, with initial groups and POSIX attributes), edit profile,
  reset/expire passwords, **password-never-expires** and **account-expiry**
  controls, enable/disable/unlock, rename, move, group membership, delete.
- **Groups** — create, edit, rename, move, delete; manage membership; all AD
  group types and scopes. Member and move targets resolve by sAMAccountName
  automatically.
- **Computers** — list, create, delete, and move computer accounts; OS and
  last-logon details.
- **Contacts** — full CRUD for contact objects, with move and rename.
- **Organizational Units** — tree browser; create, move, and delete OUs. OUs
  can't be created inside `CN=` containers (AD forbids it); Sambmin rejects that
  up front with a clear message.

## Policy & Security

- **Group Policy** — browse GPOs, inspect them, and link/unlink to OUs.
- **[Password Policies](features/password-policy.md)** — domain default policy,
  Fine-Grained Password Policies (PSOs), and a password tester.
- **[Delegation of Control](features/delegation.md)** — grant scoped AD rights
  (reset passwords, manage users/groups/computers, read-only bind, full control,
  directory replication) to users and groups on an OU or the whole domain.
  Multi-select trustees × capabilities.
- **Kerberos** — policy viewer, service-account browser, keytab export (shows
  CLI fallback), SPN management, and constrained-delegation configuration.
- **FSMO Roles** — view the five FSMO role holders.
- **Schema** — browse AD schema classes and attributes.

## Infrastructure

- **DNS** — manage zones and records for both the Samba internal DNS and BIND9
  DLZ backends; SRV validator and cross-DC consistency checks.
- **Sites & Services** — view AD sites and subnets.
- **Replication** — topology visualization and per-partition status. Requires a
  **Domain Admin** login (the read-only service account lacks DRS rights).

## System

- **Dashboard** — DC health, object counts, and recent activity.
- **Advanced Search** — full-directory LDAP search with saved queries.
- **Audit Log** — every write operation, with who / what / when.
- **Settings** — connection, auth, and application configuration.
- **Self-Service** — any signed-in user can view their profile and change their
  own password.

---

## Roles & access

Four roles are derived from AD group membership and enforced at the API before
any handler runs:

| Role | Granted to | Can do |
|---|---|---|
| **Authenticated** | any signed-in user | read everything; self-service profile + password change |
| **Operator** | Account Operators, Domain Admins, Enterprise Admins | user / group / computer / contact / OU CRUD |
| **DNS Admin** | DnsAdmins, Domain Admins, Enterprise Admins | DNS zone and record changes |
| **Admin** | Domain Admins, Enterprise Admins | password policy, **delegation of control**, GPO, replication, FSMO, SPN/delegation, keytab |

See [ARCHITECTURE.md](ARCHITECTURE.md) for the internals and
[SECURITY.md](SECURITY.md) for the security model.

---

## Related documentation

- [Installation](installation/) — FreeBSD, Linux, macOS
- [Configuration](CONFIGURATION.md) — `config.yaml` reference
- [Architecture](ARCHITECTURE.md) — read/write split, auth model, `samba-tool`
  integration notes, delegation internals
- [Security](SECURITY.md) — auth, CSRF, rate limiting, RBAC
- [FAQ](FAQ.md) — common questions and troubleshooting
