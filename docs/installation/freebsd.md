# FreeBSD Installation Guide

FreeBSD is Sambmin's primary deployment platform. This guide covers a complete installation from scratch.

## Prerequisites

- FreeBSD 13+ (tested on 14.2)
- A running Samba AD domain with at least one DC
- Root or sudo access on the FreeBSD host
- Network access to your DCs on port 636 (LDAPS)

## 1. Install Packages

Run the provided script or install manually:

```bash
# Using the provided script
sh deploy/freebsd/pkg-install.sh

# Or manually
pkg install -y python311 postgresql15-server postgresql15-client nginx
```

## 2. PostgreSQL Setup

```bash
# Enable and initialize
sysrc postgresql_enable="YES"
service postgresql initdb
service postgresql start

# Create database and user
su - postgres -c 'createuser sambmin'
su - postgres -c 'createdb -O sambmin sambmin'
```

## 3. Create Service Account

Create a dedicated service account in your Samba AD for Sambmin's read-only LDAP queries:

```bash
# On a machine with samba-tool installed
samba-tool user create sambmin-svc --description="Sambmin read-only service account"
samba-tool user setpassword sambmin-svc
samba-tool user setexpiry sambmin-svc --noexpiry
```

The account needs only default read access — no additional group memberships required.

## 4. Deploy Sambmin

### Build (on your development machine)

```bash
# Backend — cross-compile for FreeBSD
cd api && GOOS=freebsd GOARCH=amd64 go build -o sambmin ./cmd/sambmin/

# Frontend
cd web && npm install && npm run build
```

### Copy to server

```bash
# Create directories
ssh root@server 'mkdir -p /home/administrator/sambmin/web'

# Copy files
scp api/sambmin root@server:/home/administrator/sambmin/
scp -r web/dist/* root@server:/home/administrator/sambmin/web/
scp -r scripts root@server:/home/administrator/sambmin/
scp api/config.example.yaml root@server:/home/administrator/sambmin/config.yaml
```

## 5. Configure

Edit `/home/administrator/sambmin/config.yaml`:

```yaml
bind_addr: "127.0.0.1"
port: 8443

domain_controllers:
  - hostname: "dc1.yourdomain.com"
    address: "10.0.0.1"
    port: 636
    primary: true

base_dn: "DC=yourdomain,DC=com"
bind_dn: "CN=sambmin-svc,CN=Users,DC=yourdomain,DC=com"

scripts_path: "/home/administrator/sambmin/scripts"

database:
  host: "localhost"
  port: 5432
  name: "sambmin"
  user: "sambmin"
  ssl_mode: "disable"

session_timeout_hours: 8
```

Create the secrets file:

```bash
cat > /home/administrator/sambmin/secrets.env << 'EOF'
SAMBMIN_BIND_PW="your-service-account-password"
EOF
chmod 600 /home/administrator/sambmin/secrets.env
```

## 6. TLS Setup

### Option A: Let's Encrypt (public-facing)

```bash
pkg install -y py311-certbot py311-certbot-nginx
certbot --nginx -d sambmin.yourdomain.com --email admin@yourdomain.com --agree-tos --non-interactive
```

Or use the provided script: `sh deploy/tls/letsencrypt.sh sambmin.yourdomain.com admin@yourdomain.com`

### Option B: Self-Signed CA (internal)

```bash
sh deploy/tls/local-ca.sh sambmin.yourdomain.com
```

This generates a CA and server certificate. Distribute the CA cert to client machines.

## 7. nginx Configuration

Copy and edit the provided nginx config:

```bash
cp deploy/freebsd/nginx.conf /usr/local/etc/nginx/nginx.conf
```

Key sections to edit:
- `server_name` — your Sambmin hostname
- `ssl_certificate` / `ssl_certificate_key` — paths to your TLS certificates
- `root` — path to `web/dist/` directory
- `upstream sambmin_api` — should point to `127.0.0.1:8443` (matching your config.yaml)

```bash
sysrc nginx_enable="YES"
service nginx start
```

## 8. rc.d Service Setup

```bash
# Install the service script
cp deploy/freebsd/rc.d/sambmin /usr/local/etc/rc.d/sambmin
chmod +x /usr/local/etc/rc.d/sambmin

# Enable
sysrc sambmin_enable="YES"

# Optional: customize paths (defaults match this guide)
# sysrc sambmin_config="/home/administrator/sambmin/config.yaml"
# sysrc sambmin_secrets="/home/administrator/sambmin/secrets.env"

# Start
service sambmin start
```

The service script handles:
- Loading secrets from the env file
- Running as the configured user
- PID file management
- Log file at `/var/log/sambmin.log`

## 9. Verify

```bash
# Check the service is running
service sambmin status

# Check the API responds
curl -k https://localhost:8443/api/health

# Check nginx proxy
curl https://sambmin.yourdomain.com/api/health
```

Then open `https://sambmin.yourdomain.com` in a browser and log in with a Domain Admin account.

## Troubleshooting

### "LDAP health check failed"
- Verify the DC is reachable: `openssl s_client -connect dc1.yourdomain.com:636`
- Check the service account credentials in secrets.env
- Verify the bind_dn matches the actual account DN in AD

### "authentication not configured"
- This means the LDAP connection failed at startup. Check the log: `tail /var/log/sambmin.log`
- Sambmin falls back to mock mode if LDAP is unreachable

### "connection refused" on port 8443
- Check if the process is running: `sockstat -l | grep 8443`
- Verify `bind_addr` in config.yaml matches what nginx expects

### nginx returns 502 Bad Gateway
- The Go backend isn't running or isn't listening on the expected port
- Check: `service sambmin status` and `tail /var/log/sambmin.log`

### Certificate errors
- For self-signed certs, ensure the CA is trusted on the FreeBSD host: copy `ca.crt` to `/usr/local/share/certs/` and run `certctl rehash`
- For Let's Encrypt, check `certbot renew` runs successfully

## Updating

```bash
# Stop the service
service sambmin stop

# Replace the binary
scp new-sambmin root@server:/home/administrator/sambmin/sambmin

# Replace frontend if changed
scp -r web/dist/* root@server:/home/administrator/sambmin/web/

# Start
service sambmin start
```
