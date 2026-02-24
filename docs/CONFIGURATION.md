# Configuration

Sambmin is configured through a YAML config file and environment variables. Environment variables take precedence for sensitive values.

## Config File

The default config file path is `/usr/local/etc/sambmin/config.yaml`. Override with the `SAMBMIN_CONFIG` environment variable.

A complete example is provided in `api/config.example.yaml`.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `SAMBMIN_BIND_PW` | Yes | LDAP service account password. Never put this in config.yaml. |
| `SAMBMIN_CONFIG` | No | Path to config.yaml. Default: `/usr/local/etc/sambmin/config.yaml` |
## Full Config Reference

```yaml
# ─── Server ───────────────────────────────────────────────

# Address to bind the HTTP server. Use 127.0.0.1 when behind a reverse proxy.
bind_addr: "127.0.0.1"

# Port for the HTTP server. nginx proxies to this port.
port: 8443

# Origins allowed for CORS. Include your frontend URL and dev server if applicable.
allowed_origins:
  - "https://sambmin.example.com"
  - "http://localhost:5173"       # Vite dev server (remove in production)

# ─── Domain Controllers ──────────────────────────────────

# List all DCs in your domain. Mark one as primary for authentication binds.
# The LDAP pool will connect to all reachable DCs for read operations.
domain_controllers:
  - hostname: "dc1.example.com"   # FQDN (used for TLS ServerName verification)
    address: "10.0.0.1"           # IP address or resolvable hostname
    site: "Default-First-Site-Name"
    port: 636                     # LDAPS port (default: 636)
    primary: true                 # Used for auth binds and write operations
  - hostname: "dc2.example.com"
    address: "10.0.0.2"
    site: "Seattle"
    port: 636

# ─── LDAP ─────────────────────────────────────────────────

# Base DN for your domain (e.g., DC=example,DC=com)
base_dn: "DC=example,DC=com"

# Service account DN for read-only LDAP queries
bind_dn: "CN=sambmin-svc,CN=Users,DC=example,DC=com"

# Service account password — use SAMBMIN_BIND_PW env var instead
# bind_pw: ""  # DO NOT SET HERE — use environment variable

# ─── Kerberos ─────────────────────────────────────────────

kerberos:
  realm: "EXAMPLE.COM"            # Kerberos realm (usually uppercase domain)
  kdc: "dc1.example.com"          # KDC hostname
  keytab_path: "/usr/local/etc/sambmin/sambmin.keytab"  # Optional keytab file
  implementation: "heimdal"       # "heimdal" or "mit"

# ─── Scripts ──────────────────────────────────────────────

# Path to Python utility scripts (samba-tool wrappers)
scripts_path: "/usr/local/share/sambmin/scripts"

# ─── Sessions ─────────────────────────────────────────────

# Session timeout in hours. Default: 8
session_timeout_hours: 8
```

## Minimal Config

The absolute minimum to connect to an existing Samba AD domain:

```yaml
bind_addr: "127.0.0.1"
port: 8443

domain_controllers:
  - hostname: "dc1.example.com"
    address: "10.0.0.1"
    port: 636
    primary: true

base_dn: "DC=example,DC=com"
bind_dn: "CN=sambmin-svc,CN=Users,DC=example,DC=com"
```

Then set the environment variable:
```bash
export SAMBMIN_BIND_PW="your-service-account-password"
```

## Mock Mode

If no domain controllers are configured (or LDAP connection fails), Sambmin starts in **mock mode** with synthetic data. This is useful for frontend development and testing without a live AD environment. Authentication is bypassed in mock mode.

## Config File Permissions

The config file should be readable only by the Sambmin service account:

```bash
chown sambmin-user:sambmin-group /usr/local/etc/sambmin/config.yaml
chmod 640 /usr/local/etc/sambmin/config.yaml
```

The secrets environment file should be even more restricted:

```bash
chmod 600 /path/to/secrets.env
```
