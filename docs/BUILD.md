# Building Sambmin

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| [Go](https://go.dev/) | 1.23+ | Backend compilation |
| [Node.js](https://nodejs.org/) | 20+ | Frontend build toolchain |
| [Python](https://www.python.org/) | 3.11+ | samba-tool wrapper scripts (runtime only) |
| [npm](https://www.npmjs.com/) | 10+ | Frontend package management (ships with Node) |

## Backend

### Build for current platform (development)

```bash
cd api
go build -o sambmin ./cmd/sambmin/
```

### Cross-compile for FreeBSD (production)

```bash
cd api
GOOS=freebsd GOARCH=amd64 go build -o sambmin ./cmd/sambmin/
```

### Cross-compile for Linux

```bash
cd api
GOOS=linux GOARCH=amd64 go build -o sambmin ./cmd/sambmin/
```

Go's cross-compilation requires no additional tooling — set `GOOS` and `GOARCH` and build.

## Frontend

### Install dependencies

```bash
cd web
npm install
```

### Development server

```bash
cd web
npm run dev
```

This starts a Vite dev server on `http://localhost:5173` with hot module replacement. API requests are proxied to the Go backend (configure in `vite.config.ts`).

### Production build

```bash
cd web
npm run build
```

Output goes to `web/dist/`. Deploy this directory to your web server's document root.

### Lint

```bash
cd web
npm run lint
```

## Running Tests

### Backend tests

```bash
cd api
go test ./...
```

Currently 221 tests covering handlers, RBAC, session management, auth, middleware, LDAP operations, and write operations. Tests use table-driven patterns and do not require a live AD connection.

### Frontend type checking

```bash
cd web
npx tsc -b
```

TypeScript compilation catches type errors. The `npm run build` command runs this automatically.

## Mock Mode

For frontend development without a Samba AD environment, start the backend with no DCs configured:

```bash
cd api
go run ./cmd/sambmin/
```

Without `domain_controllers` in the config (or without a config file at all), Sambmin starts in mock mode with synthetic data and no authentication requirement.

## Production Build Notes

### Frontend build on slow filesystems

If your source tree is on a network filesystem (Google Drive, SMB share, etc.), the frontend build may be slow. Build in a local temp directory instead:

```bash
rsync -a --delete web/ /tmp/sambmin-web/
cd /tmp/sambmin-web && npm run build
cp -r /tmp/sambmin-web/dist/ web/dist/
```

### Binary size

The Go binary is a single statically-linked executable (~15-20 MB). No runtime dependencies beyond the OS.

### Deployment checklist

1. Cross-compile backend binary for target OS
2. Build frontend (`npm run build`)
3. Copy binary, `web/dist/`, `scripts/`, and config to target
4. Set environment variables (`SAMBMIN_BIND_PW`, `SAMBMIN_CONFIG`)
5. Configure reverse proxy (nginx or Apache)
6. Start the service

See the [installation guides](installation/) for platform-specific instructions.
