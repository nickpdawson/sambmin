# Sambmin Fix Pass — Post-Deployment Integration Testing

## Context

Sambmin was built through milestones M13-M20 using automated Ralph loops. The code compiles, 221 tests pass, and the frontend builds. However, **live deployment testing on the FreeBSD DC (Bridger)** revealed significant issues. Many features that Ralph marked "complete" are broken, incomplete, or regression-damaged.

This is a focused fix pass. Do NOT add new features. Fix what's broken.

## Critical Rules

1. **Read CLAUDE.md first** — project conventions.
2. **Read plan.md** — especially the Write Operations Architecture section for the `-H ldap://localhost` pattern.
3. **Every `samba-tool` command MUST include `-H ldap://localhost`** — this is the #1 bug. Without it, samba-tool tries to open the local sam.ldb file directly, which fails with "Permission denied" because the web app process doesn't have root access to `/var/db/samba4/private/sam.ldb`. The correct pattern is already in the M11-M13 handlers — follow it exactly.
4. **Every `samba-tool` command that performs a write MUST include `-U user%pass`** using the session credentials. Read-only commands use the service account bind password from config.
5. **Test every handler by reading its code, verifying the samba-tool command it constructs, and confirming `-H ldap://localhost` is present.**
6. **Do not break working features.** The Users page, Groups page, Computers page, OUs list view, DNS records, and auth were all working before M16-M20. If they're broken now, find the regression and fix it.

## The samba-tool Pattern

Every samba-tool command on this system MUST look like one of these:

**Read operations (service account):**
```
samba-tool <command> -H ldap://localhost -U services%<bind_password>
```

**Write operations (user session credentials):**
```
samba-tool <command> -H ldap://localhost -U <username>%<password>
```

**NEVER** call samba-tool without `-H ldap://localhost`. It will try to open local TDB files and fail.

Examples of correct commands:
```bash
# Sites
samba-tool sites list -H ldap://localhost -U services%password
samba-tool sites create MySite -H ldap://localhost -U admin%password

# FSMO
samba-tool fsmo show -H ldap://localhost -U services%password
samba-tool fsmo transfer --role=rid -H ldap://localhost -U admin%password

# Replication
samba-tool drs showrepl -H ldap://localhost -U services%password
samba-tool drs replicate <dest> <source> <nc> -H ldap://localhost -U admin%password

# Subnets
samba-tool sites subnet list -H ldap://localhost -U services%password
samba-tool sites subnet create 10.15.1.0/24 MySite -H ldap://localhost -U admin%password

# DNS (already uses --server= pattern but also needs auth)
samba-tool dns serverinfo localhost -U services%password
samba-tool dns zoneinfo localhost dzsec.net -U services%password
samba-tool dns zoneoptions localhost dzsec.net --aging=1 -U admin%password

# GPO
samba-tool gpo listall -H ldap://localhost -U services%password
samba-tool gpo create "My GPO" -H ldap://localhost -U admin%password

# Schema
samba-tool schema attribute show cn -H ldap://localhost -U services%password
samba-tool schema objectclass show user -H ldap://localhost -U services%password

# Domain
samba-tool domain level show -H ldap://localhost -U services%password
samba-tool domain passwordsettings show -H ldap://localhost -U services%password
samba-tool domain passwordsettings pso list -H ldap://localhost -U services%password

# SPNs
samba-tool spn list administrator -H ldap://localhost -U services%password
samba-tool spn add HTTP/web.dzsec.net administrator -H ldap://localhost -U admin%password

# Auth policies
samba-tool domain auth policy list -H ldap://localhost -U services%password

# Trusts
samba-tool domain trust list -H ldap://localhost -U services%password
```

## Specific Issues Found (fix ALL of these)

### CRITICAL — Regression

**C1: Users page shows nothing**
- Users page was working perfectly before M16-M20.
- Now shows empty or fails to load.
- Check: did a route change break `/api/users`? Did frontend routing change? Did the handler registration get overwritten?
- Fix: restore Users page functionality. Compare against the git history if needed (`git log --oneline`, `git diff HEAD~5 -- api/internal/handlers/users_live.go`, `git diff HEAD~5 -- web/src/pages/Users/`).

### CRITICAL — Missing `-H ldap://localhost`

**C2: Sites & Services — "Permission denied" on sam.ldb**
- `samba-tool sites list` missing `-H ldap://localhost`
- `samba-tool sites subnet list` missing `-H ldap://localhost`
- Fix: add `-H ldap://localhost -U services%<password>` to all sites/subnet read commands.

**C3: FSMO Roles — "Permission denied" on sam.ldb**
- `samba-tool fsmo show` missing `-H ldap://localhost`
- Fix: add `-H ldap://localhost -U services%<password>`.

**C4: Replication — all 5 DCs showing "Unreachable"**
- `samba-tool drs showrepl` missing `-H ldap://localhost`
- The handler likely iterates over DCs and runs `drs showrepl` against each. Every invocation needs `-H ldap://<dc_hostname> -U services%<password>`.
- Note: for replication you query EACH DC, not just localhost. The command should be:
  `samba-tool drs showrepl -H ldap://bridger.dzsec.net -U services%password`
  `samba-tool drs showrepl -H ldap://showdown.dzsec.net -U services%password`
  etc. (or use the DC's IP/hostname from config)

### HIGH — Incomplete/Stub Pages

**H1: Kerberos page — does nothing**
- The page exists but has no real functionality.
- Backend: check if SPN handlers exist (`handleListSPNs`, etc.). If they exist, wire the frontend. If they're stubs, implement them properly with `-H ldap://localhost`.
- Minimum viable: list SPNs for a given account, add SPN, delete SPN.

**H2: Settings page — not real data**
- If the settings page exists but shows hardcoded/mock data, either wire it to real config or clearly mark it as "coming soon" rather than showing fake data that confuses the user.

### MEDIUM — Functionality Issues

**M1: OU tree view broken**
- List view works, tree view doesn't.
- Check the `/api/ous/tree` endpoint and the frontend TreeView component.
- The tree should render the full OU hierarchy from the domain root.

**M2: OUs don't show CN= containers**
- AD has both `OU=` organizational units and `CN=` containers (like `CN=Users`, `CN=Computers`, `CN=Builtin`).
- The OU page should list BOTH. The LDAP query probably filters on `(objectClass=organizationalUnit)` which misses containers.
- Fix: query for `(|(objectClass=organizationalUnit)(objectClass=container))` or add containers as a separate section.

### LOW — Error Message Cleanup

**L1: Strip samba-tool warnings from error display**
- The "WARNING: Using passwords on command line is insecure. Installing the setproctitle python module..." text appears in error messages shown to users.
- This was already handled in M11 (`Cleaned error messages from samba-tool`). Check if the new handlers use the same error cleaning function.
- The `cleanSambaToolError()` or equivalent function should strip WARNING lines and extract the actual error.

## Process

```
1. Read CLAUDE.md
2. Read plan.md (Write Operations Architecture section)
3. Fix C1 first (Users regression) — this is the most important
4. Fix C2-C4 (samba-tool -H flags) — audit EVERY handler file for missing -H
5. Fix H1-H2 (incomplete pages)
6. Fix M1-M2 (OU issues)
7. Fix L1 (error cleanup)
8. Run: cd api && go test ./...
9. Run: cd web && npx tsc --noEmit
10. Run: cd web && npm run build
11. Run: GOOS=freebsd GOARCH=amd64 go build -o sambmin ./cmd/sambmin/
12. Verify no regressions by checking that handler registrations in routes.go are intact
```

## Audit Checklist

Go through EVERY file in `api/internal/handlers/` and verify:

- [ ] `infrastructure.go` — sites, subnets, FSMO, replication, audit: ALL need `-H ldap://...`
- [ ] `dns_deep.go` — serverinfo, zoneinfo, zoneoptions, query, SRV validator, consistency: verify auth flags
- [ ] `gpo.go` or equivalent — GPO handlers: need `-H ldap://localhost`
- [ ] `kerberos.go` or equivalent — SPN handlers: need `-H ldap://localhost`
- [ ] `policy.go` or equivalent — auth policies/silos: need `-H ldap://localhost`
- [ ] `schema.go` or equivalent — schema browser: need `-H ldap://localhost`
- [ ] `domain.go` or equivalent — domain level, backup, dbcheck: need `-H ldap://localhost`
- [ ] `trust.go` or equivalent — trust management: need `-H ldap://localhost`
- [ ] `password_policy.go` or equivalent — domain passwordsettings: need `-H ldap://localhost`
- [ ] `search.go` or equivalent — verify LDAP search still works
- [ ] `users_live.go` — verify not broken by later changes
- [ ] `contacts_live.go` — verify not broken
- [ ] `directory_live.go` — verify not broken

Then go through EVERY file in `web/src/pages/` and verify:

- [ ] Each page's API calls match existing backend routes
- [ ] No hardcoded/mock data in pages that should be live
- [ ] Error handling displays clean messages (not raw samba-tool stderr)
- [ ] All pages listed in navigation actually render

## Completion

When ALL issues above are fixed, ALL tests pass, and frontend builds:

<promise>FIXES_COMPLETE</promise>
