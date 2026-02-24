#!/bin/sh
# Install Sambmin dependencies on FreeBSD
# Run as root: sh pkg-install.sh

set -e

echo "Installing Sambmin dependencies..."

# Python (for samba-tool wrapper scripts)
pkg install -y python311

# nginx (reverse proxy)
pkg install -y nginx

echo ""
echo "=== Post-install steps ==="
echo ""
echo "1. Enable services in /etc/rc.conf:"
echo '   sysrc nginx_enable="YES"'
echo '   sysrc sambmin_enable="YES"'
echo ""
echo "2. Copy config:"
echo "   mkdir -p /usr/local/etc/sambmin"
echo "   cp config.example.yaml /usr/local/etc/sambmin/config.yaml"
echo "   # Edit config.yaml for your environment"
echo ""
echo "3. Deploy Sambmin:"
echo "   cp sambmin /usr/local/bin/"
echo "   cp -r scripts /usr/local/share/sambmin/"
echo "   cp -r web/dist /usr/local/share/sambmin/web"
echo "   cp deploy/freebsd/rc.d/sambmin /usr/local/etc/rc.d/"
echo "   chmod +x /usr/local/etc/rc.d/sambmin"
echo ""
echo "4. Configure nginx:"
echo "   cp deploy/freebsd/nginx.conf /usr/local/etc/nginx/conf.d/sambmin.conf"
echo "   # Edit for your TLS certificates"
echo "   service nginx reload"
echo ""
echo "5. Start Sambmin:"
echo "   service sambmin start"
