# Linux Installation Guide

This guide covers installation on Ubuntu/Debian. Adapt package names for other distributions.

## Prerequisites

- Ubuntu 22.04+ or Debian 12+
- A running Samba AD domain with at least one DC
- Root or sudo access
- Network access to your DCs on port 636 (LDAPS)

## 1. Install Packages

```bash
sudo apt update
sudo apt install -y python3 postgresql nginx
```

## 2. PostgreSQL Setup

```bash
sudo systemctl enable postgresql
sudo systemctl start postgresql

sudo -u postgres createuser sambmin
sudo -u postgres createdb -O sambmin sambmin
```

## 3. Create Service Account

Same as FreeBSD — create a dedicated read-only service account in your Samba AD:

```bash
samba-tool user create sambmin-svc --description="Sambmin read-only service account"
samba-tool user setpassword sambmin-svc
samba-tool user setexpiry sambmin-svc --noexpiry
```

## 4. Deploy Sambmin

### Build (on your development machine)

```bash
cd api && GOOS=linux GOARCH=amd64 go build -o sambmin ./cmd/sambmin/
cd web && npm install && npm run build
```

### Copy to server

```bash
sudo mkdir -p /opt/sambmin/{web,scripts}

sudo cp api/sambmin /opt/sambmin/
sudo cp -r web/dist/* /opt/sambmin/web/
sudo cp -r scripts/* /opt/sambmin/scripts/
sudo cp api/config.example.yaml /opt/sambmin/config.yaml
```

## 5. Create System User

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin sambmin
sudo chown -R sambmin:sambmin /opt/sambmin
```

## 6. Configure

Edit `/opt/sambmin/config.yaml` — see [CONFIGURATION.md](../CONFIGURATION.md) for full reference.

Create the secrets file:

```bash
sudo tee /opt/sambmin/secrets.env > /dev/null << 'EOF'
SAMBMIN_BIND_PW=your-service-account-password
SAMBMIN_CONFIG=/opt/sambmin/config.yaml
EOF
sudo chmod 600 /opt/sambmin/secrets.env
sudo chown sambmin:sambmin /opt/sambmin/secrets.env
```

## 7. systemd Service

Copy the provided unit file:

```bash
sudo cp deploy/linux/sambmin.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable sambmin
sudo systemctl start sambmin
```

Check status:

```bash
sudo systemctl status sambmin
sudo journalctl -u sambmin -f
```

## 8. TLS Setup

### Let's Encrypt (recommended for public-facing)

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d sambmin.yourdomain.com
```

### Self-Signed (internal deployments)

Use the provided script:

```bash
sudo sh deploy/tls/local-ca.sh sambmin.yourdomain.com
```

## 9. nginx Configuration

Create `/etc/nginx/sites-available/sambmin`:

```nginx
upstream sambmin_api {
    server 127.0.0.1:8443;
    keepalive 32;
}

server {
    listen 443 ssl;
    server_name sambmin.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/sambmin.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/sambmin.yourdomain.com/privkey.pem;

    root /opt/sambmin/web;
    index index.html;

    # Security headers
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location /api/ {
        proxy_pass http://sambmin_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
        proxy_read_timeout 300;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /assets/ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}

server {
    listen 80;
    server_name sambmin.yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

Enable and reload:

```bash
sudo ln -s /etc/nginx/sites-available/sambmin /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Platform Differences from FreeBSD

| Aspect | FreeBSD | Linux (Ubuntu/Debian) |
|--------|---------|----------------------|
| Package manager | `pkg` | `apt` |
| Service management | `rc.d` / `service` | `systemd` / `systemctl` |
| Default install path | `/home/administrator/sambmin` | `/opt/sambmin` |
| nginx config | `/usr/local/etc/nginx/` | `/etc/nginx/` |
| TLS certs (Let's Encrypt) | `/usr/local/etc/letsencrypt/` | `/etc/letsencrypt/` |
| Log viewing | `tail /var/log/sambmin.log` | `journalctl -u sambmin` |
| Python binary | `python3.11` | `python3` |

## Troubleshooting

### Service won't start
```bash
sudo journalctl -u sambmin --no-pager -n 50
```

### "permission denied" on binary
```bash
sudo chmod +x /opt/sambmin/sambmin
```

### SELinux (RHEL/CentOS/Fedora)
If running on a distribution with SELinux, you may need to set the correct context:
```bash
sudo semanage fcontext -a -t bin_t '/opt/sambmin/sambmin'
sudo restorecon -v /opt/sambmin/sambmin
```

Or configure SELinux to allow the proxy connection:
```bash
sudo setsebool -P httpd_can_network_connect 1
```
