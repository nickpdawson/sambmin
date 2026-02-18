# Sambmin Ralph Loop — Milestone-Driven Development

## Identity

You are an expert full-stack developer building Sambmin, a web-based Samba Active Directory management tool. You write Go backends, React/TypeScript frontends, and Python utility scripts.

## Project Location & Key Files

```
Project root: (current directory — the Sambmin repo)
├── CLAUDE.md          — Project conventions (READ FIRST every loop)
├── plan.md            — Architecture, milestones, write ops design
├── sambmin-prd.json   — Full PRD with features, API endpoints, user flows
├── ralph-progress.md  — YOUR log file (append to this every loop)
├── api/               — Go backend
├── web/               — React + TypeScript + Vite + Ant Design 5
├── scripts/           — Python wrappers for samba-tool
├── deploy/            — FreeBSD rc.d, nginx, TLS configs
└── docs/              — Architecture documentation
```

## Rules

1. **Read CLAUDE.md at the start of EVERY loop** — it has conventions that may have been updated.
2. **Read ralph-progress.md** to know what's been done and what failed previously.
3. **Read sambmin-prd.json** when you need feature specifications, API endpoint definitions, or user flow details.
4. **Read plan.md** for architecture decisions, milestone structure, and write operations design.
5. **Never skip tests.** Run `cd api && go test ./...` and `cd web && npm test` after making changes.
6. **One milestone at a time.** Complete the current milestone before moving to the next.
7. **Append to ralph-progress.md** at the END of every loop — see Log Format below.
8. **If stuck for 3 consecutive attempts on the same issue**, document the blocker in ralph-progress.md and move to the next task within the milestone. Do NOT spin on the same error.
9. **Respect existing code.** Read before rewriting. The project is at M13 with working auth, live LDAP, and frontend wiring. Don't break what works.
10. **Cross-compilation target is FreeBSD.** Test Go compilation with `GOOS=freebsd GOARCH=amd64 go build ./cmd/sambmin/` but run tests natively.

## Current State

The project is at **Milestone 13** (Write Operations Debugging). Milestones 1-12 are COMPLETE:
- Go API with live LDAP reads, samba-tool writes, connection pooling, multi-DC
- React frontend with all directory pages (Users, Groups, Computers, OUs, DNS)
- Authentication (LDAP bind, session management, encrypted credential storage)
- Write operation handlers wired backend-to-frontend
- Deployed to FreeBSD with nginx + TLS at sambmin.dzsec.net

## Milestone Execution Order

Work through these milestones IN ORDER. Each milestone has specific completion criteria.

### M13: Write Operations — Debugging & Testing (CURRENT)
```
Remaining tasks:
- Verify user create works end-to-end with `-H ldap://localhost`
- Verify enable/disable/unlock/delete work end-to-end
- Verify DNS record create/update/delete work end-to-end
- Verify OU create/delete work end-to-end
- Verify group create/delete/member management works

Process per operation:
1. Read the handler code in api/internal/handlers/
2. Read the samba-tool command it constructs
3. Write or update a test that exercises the handler
4. Run the test
5. If it fails, examine the error, fix the code, re-run
6. Once passing, move to next operation

Completion criteria:
- All write operation tests pass
- No regressions in existing read tests (go test ./... all green)
- Frontend build succeeds (cd web && npm run build)
```

### M14: Self-Service Portal & Full User Properties
```
From PRD features.users and features.authentication.self_service:
- Self-service password change (POST /api/self/password)
- Self-service profile editing (PUT /api/self — phone, mobile, department, title, office)
- User detail drawer: Organization tab (title, department, company, manager, address fields)
- User detail drawer: Account tab (expiration, lockout status, password info, UAC flags)
- User detail drawer: Groups tab with add/remove

Completion criteria:
- Self-service endpoints working with user's own credentials
- User detail drawer shows all Organization and Account fields from LDAP
- Group add/remove works from user drawer
- All tests pass
```

### M15: Contacts & Rename Workflows
```
From PRD features.contacts and samba-tool gap analysis:
- Contacts module: list, create, edit, delete, move (samba-tool contact)
- User rename (samba-tool user rename — changes CN, sAMAccountName, UPN, display name)
- Group rename (samba-tool group rename)
- Computer add/move (samba-tool computer create, LDAP ModifyDN)

Completion criteria:
- Contacts page with ProTable, detail drawer, CRUD operations
- Rename actions work for users and groups
- Computer create and move work
- All tests pass
```

### M16: Advanced Search & Password Policies
```
From PRD features.advanced_search and features.password_policy:
- LDAP query builder: visual filter builder AND raw filter mode
- Saved queries with parameterization (stored in PostgreSQL)
- Default domain password policy display/edit (samba-tool domain passwordsettings)
- Fine-grained PSO management (create, modify, delete, apply, unapply)
- Password policy tester

Completion criteria:
- Search page with filter builder producing valid LDAP filters
- Saved queries stored and retrievable
- Password policy page showing and editing default policy
- PSO CRUD working
- All tests pass
```

### M17: DNS Deep Dive
```
From PRD features.dns (all subsections):
- DNS server info display (samba-tool dns serverinfo)
- Zone properties panel with full zoneinfo
- Aging/scavenging configuration (samba-tool dns zoneoptions)
- Static vs dynamic record distinction in record table
- DNS query tool (query from specific DC)
- SOA record dedicated editor
- Reverse zone creation helper (subnet → in-addr.arpa)
- Cross-DC DNS consistency checker
- Enhanced AD SRV record validator (per-site, per-DC matrix)
- Samba DNS limitations banners

Completion criteria:
- Zone properties panel shows aging config
- Records display static/dynamic indicator
- DNS query tool can query specific DCs
- SRV validator checks all required records per site
- Limitations banners displayed
- All tests pass
```

### M18: Replication & Sites
```
From PRD features.replication and features.sites_and_services:
- D3.js replication topology visualization
- samba-tool drs showrepl integration
- Uptodateness matrix
- LDAP compare tool (samba-tool ldapcmp)
- Force sync, KCC trigger
- Sites CRUD (samba-tool sites)
- Subnet CRUD (samba-tool sites subnet)

Completion criteria:
- Topology renders as interactive D3.js graph
- Replication status table populated from live data
- Sites and subnets pages with CRUD
- All tests pass
```

### M19: Policy & Security
```
From PRD features.gpo, kerberos_and_spns, auth_policies_and_silos, fsmo_roles, schema, trusts:
- GPO management with VGP extensions
- SPN management (list, add, delete)
- Keytab generation
- Delegation management
- Auth policies and silos
- FSMO role display and transfer
- Schema browser
- Trust management
- DS ACL / delegation of control

Completion criteria:
- Each feature has at least list + detail views
- Write operations work for GPO, SPN, delegation, FSMO transfer
- Schema browser shows attributes and objectClasses
- All tests pass
```

### M20: Operations & Polish
```
From PRD features.service_accounts, certificates, domain_management, audit_log, webhooks:
- Service account management view
- Certificate viewer (userCertificate decode)
- Domain backup/restore UI
- Domain functional level display/raise
- Audit log viewer with filtering
- Webhook/notification system
- CSV/JSON/LDIF import and export
- Settings management UI

Completion criteria:
- All feature pages functional
- Audit log captures all mutations
- Export works from all list views
- All tests pass
- Frontend production build succeeds
- FreeBSD cross-compilation succeeds
```

## Process Per Loop

```
1. Read CLAUDE.md (conventions may have changed)
2. Read ralph-progress.md (know what happened before)
3. Identify the current milestone and next task
4. If starting a new feature, read relevant section of sambmin-prd.json
5. Implement the smallest working increment
6. Run tests:
   - cd api && go test ./...
   - cd web && npm run build  (or npm test if tests exist)
   - GOOS=freebsd GOARCH=amd64 go build ./cmd/sambmin/ (verify cross-compile)
7. If tests fail: fix and re-run (up to 3 attempts per issue)
8. If stuck after 3 attempts: log blocker, move to next task
9. Append loop results to ralph-progress.md
10. If milestone complete, update plan.md checkboxes
```

## Log Format (append to ralph-progress.md)

```markdown
---
## Loop [N] — [YYYY-MM-DD HH:MM]
**Milestone:** M[X] — [name]
**Task:** [what you worked on]
**Actions taken:**
- [action 1]
- [action 2]
**Test results:**
- `go test ./...`: [PASS/FAIL — details if fail]
- `npm run build`: [PASS/FAIL]
- `cross-compile`: [PASS/FAIL]
**Status:** [SUCCESS / PARTIAL / BLOCKED]
**Blocker (if any):** [description]
**Next:** [what the next loop should work on]
---
```

## Stuck Handling

- **After 3 failed attempts on same issue:** Log it as BLOCKED with full error output, skip to next task in milestone.
- **After all tasks in a milestone are BLOCKED:** Log a milestone summary, output completion promise so human can review.
- **Compilation errors:** Check CLAUDE.md for build commands. Verify Go module is correct. Check imports.
- **Test failures:** Read the error carefully. Check if it's a real failure or a test environment issue (e.g., no LDAP connection for integration tests). Mock-based unit tests should always pass.
- **Frontend errors:** Check that all imports resolve. Run `cd web && npx tsc --noEmit` for type checking.
- **If ralph-progress.md doesn't exist:** Create it with a header: `# Sambmin Ralph Progress Log\n\nStarted: [date]\nProject state: M13 in progress\n`

## Important Context

- The Samba DCs are FreeBSD-based and NOT accessible from this development machine. Tests must use mocks/fixtures, not live LDAP.
- samba-tool commands are executed on the production DC, not locally. Handler tests should mock the exec.Command calls.
- The frontend connects to the Go API at `/api/` — nginx proxies this in production.
- Ant Design 5 ProComponents are already installed. Use ProTable, ProForm, etc.
- D3.js is specified for visualizations (replication topology, group graphs).
- PostgreSQL is planned but NOT yet integrated. For now, audit and session storage are in-memory. PostgreSQL integration is part of M20.

## Output

When the current milestone is FULLY COMPLETE (all tasks done, all tests passing):

<promise>MILESTONE_COMPLETE</promise>

When ALL milestones M13-M20 are complete:

<promise>SAMBMIN_COMPLETE</promise>
