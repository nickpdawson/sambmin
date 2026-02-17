#!/bin/sh
# Set up Let's Encrypt TLS certificate for Sambmin
# Run as root on the FreeBSD server

set -e

DOMAIN="${1:-sambmin.dzsec.net}"
EMAIL="${2:-admin@dzsec.net}"

echo "Setting up Let's Encrypt for ${DOMAIN}..."

# Install certbot
pkg install -y py311-certbot py311-certbot-nginx

# Obtain certificate using nginx plugin
certbot --nginx \
    -d "${DOMAIN}" \
    --email "${EMAIL}" \
    --agree-tos \
    --non-interactive

# Enable auto-renewal via cron
if ! crontab -l 2>/dev/null | grep -q certbot; then
    (crontab -l 2>/dev/null; echo "0 0,12 * * * /usr/local/bin/certbot renew --quiet && service nginx reload") | crontab -
    echo "Added certbot auto-renewal to crontab"
fi

echo ""
echo "Certificate installed. nginx should now serve HTTPS for ${DOMAIN}."
echo "Auto-renewal is configured via crontab (runs twice daily)."
