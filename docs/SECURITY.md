# Security

## Overview

Sambmin manages Active Directory — a security-critical system. The security model is designed around the principle of least privilege, defense in depth, and honest attribution of all mutations.

## Two-Tier Authentication

### Service Account (Reads)

The service account configured via `bind_dn` handles all LDAP read operations. It should have:

- **Read access** to the directory tree (users, groups, computers, DNS, schema)
- **No write permissions** — it cannot create, modify, or delete objects
- **No Domain Admin membership** — it does not need elevated privileges

The service account password is provided via the `SAMBMIN_BIND_PW` environment variable. It is never written to config files or logs.

### User Credentials (Writes)

When a user logs in, their password is encrypted with AES-256-GCM and stored in the server's memory for the duration of the session. Write operations decrypt the user's password and pass it to `samba-tool`, ensuring:

- AD audit logs attribute changes to the actual user
- The service account cannot be exploited for unauthorized writes
- Users can only perform operations their AD permissions allow

## TLS Requirements

All LDAP connections use LDAPS (port 636) with TLS 1.2+. The Go backend validates server certificates against the system trust store (or the DC's hostname via `ServerName` in the TLS config).

The frontend should be served over HTTPS. Session cookies are set with `Secure: true` and will not be sent over plain HTTP.

### Certificate Setup

- **Let's Encrypt**: See `deploy/tls/letsencrypt.sh` for automated certificate setup
- **Self-Signed CA**: See `deploy/tls/local-ca.sh` for internal deployments without public DNS

## Service Account Setup

Create a dedicated service account in your Samba AD domain:

```bash
# Create the service account
samba-tool user create sambmin-svc --description="Sambmin read-only service account"

# Set a strong password (store it securely)
samba-tool user setpassword sambmin-svc

# Prevent password expiry
samba-tool user setexpiry sambmin-svc --noexpiry

# The account only needs default read access — no group memberships required
```

The service account's DN will be something like:
```
CN=sambmin-svc,CN=Users,DC=yourdomain,DC=com
```

Use this as the `bind_dn` in your config.yaml.

## RBAC Group Mapping

Sambmin derives user roles from AD group membership at login time:

| Role | Required AD Group Membership | Capabilities |
|------|------------------------------|--------------|
| **Admin** | Domain Admins or Enterprise Admins | All operations: password policy, GPO, replication, FSMO transfers, SPN, delegation, keytab export |
| **DNS Admin** | DnsAdmins, Domain Admins, or Enterprise Admins | DNS zone and record management |
| **Operator** | Account Operators, Domain Admins, or Enterprise Admins | User, group, computer, contact, and OU create/modify/delete |
| **Authenticated** | Any valid AD account | Read access to all objects, self-service profile and password change, global search |

Role checks are enforced at the HTTP handler level — the `RequireRole` middleware rejects unauthorized requests with `403 Forbidden` before any business logic executes.

## Session Security

- **Encryption**: User passwords stored in session are encrypted with AES-256-GCM using a random 256-bit key generated at server startup
- **Session Cookie** (`sambmin_session`): `HttpOnly`, `Secure`, `SameSite=Strict` — JavaScript cannot access it, it's only sent over HTTPS, and it won't be included in cross-site requests
- **Timeout**: Configurable (default 8 hours), with background cleanup every 5 minutes
- **Server-side storage**: Sessions are stored in server memory, not in the cookie — the cookie contains only a 256-bit random session ID

## CSRF Protection

Sambmin uses the double-submit cookie pattern:

1. On login, the server sets a `sambmin_csrf` cookie (readable by JavaScript, `SameSite=Strict`)
2. The frontend reads this cookie and includes its value as an `X-CSRF-Token` header on all mutation requests (POST, PUT, DELETE)
3. The CSRF middleware verifies the header matches the cookie
4. Requests without a matching token receive `403 Forbidden`

Safe methods (GET, HEAD, OPTIONS) and the login endpoint are exempt.

## Rate Limiting

Login attempts are rate-limited with a sliding window algorithm:

- **Per IP**: 10 failed attempts per 1-minute window
- **Per username**: 5 failed attempts per 15-minute window

When rate-limited, the server returns `429 Too Many Requests` with a `Retry-After` header. Successful logins do not count toward the limit. The rate limiter runs in memory with automatic cleanup of stale entries.

## Input Validation

- **LDAP filter escaping**: All user-supplied values used in LDAP search filters are escaped using `ldap.EscapeFilter()` to prevent LDAP injection
- **DN validation**: Distinguished Names from user input are validated before use
- **Request body limits**: JSON request bodies are bounded by Go's default `http.MaxBytesReader`
- **Error sanitization**: Internal error details are logged server-side but not exposed to clients — `respondSafeError()` returns a generic message while logging the real error

## Security Hardening

Sambmin underwent a security audit prior to beta release. Three HIGH-severity issues were identified and remediated:

1. **RBAC enforcement** — All write endpoints are now wrapped with role-checking middleware, preventing privilege escalation through direct API calls
2. **LDAP injection prevention** — All user-supplied search terms are escaped before inclusion in LDAP filters, preventing directory traversal or data exfiltration
3. **Credential exposure mitigation** — User passwords are encrypted at rest in session memory with AES-256-GCM; error messages and logs are sanitized to prevent credential leakage

## HTTP Security Headers

The nginx configuration sets the following headers:

```
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'
```

The Go API additionally sets `Content-Security-Policy: default-src 'none'` on API responses.

## Server Configuration

The Go HTTP server is configured with timeouts to prevent slowloris and resource exhaustion attacks:

- **ReadTimeout**: 15 seconds
- **WriteTimeout**: 30 seconds
- **IdleTimeout**: 60 seconds

## Reporting Vulnerabilities

If you discover a security vulnerability in Sambmin, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email the maintainer directly with details of the vulnerability
3. Include steps to reproduce if possible
4. Allow reasonable time for a fix before public disclosure

We take security reports seriously and will respond promptly.
