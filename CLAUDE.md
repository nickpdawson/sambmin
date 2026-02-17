# Sambmin - Project Conventions

## What This Is
Web-based Samba Active Directory management tool. Go API backend + React frontend + Python utility scripts.

## Project Structure
- `api/` — Go backend (net/http, LDAP, Kerberos auth, PostgreSQL)
- `web/` — React + TypeScript + Vite + Ant Design 5 frontend
- `scripts/` — Python wrappers around samba-tool, ldbtools, BIND9 utilities
- `deploy/` — FreeBSD rc.d, nginx, TLS, setup configs
- `docs/` — Architecture and API documentation

## Tech Stack
- **Backend:** Go (net/http router, gokrb5, go-ldap, pgx)
- **Frontend:** React 18, TypeScript, Vite, Ant Design 5 + ProComponents, D3.js
- **Scripts:** Python 3.11+, wrapping samba-tool CLI
- **Database:** PostgreSQL 15+ (app data only — audit, sessions, config)
- **Target OS:** FreeBSD (cross-compiled from macOS)

## Build Commands
```bash
# Go backend
cd api && go build -o sambmin ./cmd/sambmin/

# Cross-compile for FreeBSD
cd api && GOOS=freebsd GOARCH=amd64 go build -o sambmin ./cmd/sambmin/

# React frontend
cd web && npm install && npm run dev    # dev server
cd web && npm run build                  # production build

# Run tests
cd api && go test ./...
cd web && npm test
```

## Code Conventions

### Go
- Standard library `net/http` for routing (no framework unless complexity demands it)
- Packages under `api/internal/` — not importable externally
- Error handling: wrap with context using `fmt.Errorf("operation: %w", err)`
- Logging: `slog` (structured logging, stdlib)
- Config: environment variables + YAML config file
- Tests: table-driven, `_test.go` files alongside source

### TypeScript/React
- Functional components only, hooks for state
- Ant Design 5 components — don't reinvent what AntD provides
- API client in `web/src/api/` — typed request/response
- Pages in `web/src/pages/`, shared components in `web/src/components/`
- Use `Inter` for body text, `JetBrains Mono` for technical values (DNs, SIDs, IPs)

### Python Scripts
- Each script is a standalone CLI tool callable via `python3 script.py <action> [args]`
- JSON output to stdout for Go to parse
- Errors to stderr with non-zero exit code
- No interactive prompts — all input via CLI args or stdin JSON

## Architecture Decisions
- Go reads AD via direct LDAP; writes via samba-tool (through Python scripts)
- DNS management abstracts over Samba internal DNS and BIND9 backends
- Kerberos auth abstracts over Heimdal and MIT implementations
- PostgreSQL stores ONLY app data (audit, sessions, config) — never AD data
- All mutations logged to audit trail

## Security
- Never commit .env files, credentials, or private keys
- Service account creds encrypted at rest
- All API endpoints require authentication
- RBAC mapped from AD group membership
- CSRF tokens on all mutation endpoints

## License
GPLv3 (matching Samba's license)
