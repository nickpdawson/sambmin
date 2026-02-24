# macOS Development Setup

> **macOS is for development only.** Sambmin targets FreeBSD and Linux for production deployment. Use macOS to develop, test, and cross-compile.

## Prerequisites

- macOS 13+ (Apple Silicon or Intel)
- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) 20+ (via Homebrew or direct download)
- A remote Samba AD DC accessible over the network (or use mock mode)

## Setup

### 1. Clone and build

```bash
git clone https://github.com/yourorg/sambmin.git
cd sambmin

# Backend
cd api && go build -o sambmin ./cmd/sambmin/

# Frontend
cd ../web && npm install
```

### 2. Run in mock mode (no AD required)

```bash
cd api && go run ./cmd/sambmin/
```

Without a config file, Sambmin starts in mock mode with synthetic data and no authentication. The API runs on `http://localhost:8443`.

In a second terminal:

```bash
cd web && npm run dev
```

The Vite dev server starts on `http://localhost:5173` and proxies API requests to the backend.

### 3. Connect to a remote AD (optional)

Create a local config file:

```bash
cp api/config.example.yaml api/config.yaml
```

Edit `api/config.yaml` with your remote DC's address, then:

```bash
export SAMBMIN_BIND_PW="your-service-account-password"
export SAMBMIN_CONFIG="$(pwd)/api/config.yaml"
cd api && go run ./cmd/sambmin/
```

You'll need network access to the DC on port 636 (LDAPS). VPN may be required.

## Cross-Compilation

Build for deployment targets without leaving macOS:

```bash
# FreeBSD (primary target)
cd api && GOOS=freebsd GOARCH=amd64 go build -o sambmin ./cmd/sambmin/

# Linux
cd api && GOOS=linux GOARCH=amd64 go build -o sambmin ./cmd/sambmin/
```

## Running Tests

```bash
cd api && go test ./...
```

All tests run locally without a live AD connection.

## Frontend Development

The Vite dev server provides hot module replacement for rapid frontend iteration:

```bash
cd web && npm run dev
```

When running against the Go backend in mock mode, you get full UI functionality with synthetic data — no Samba environment needed.
