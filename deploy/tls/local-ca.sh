#!/bin/sh
# Generate a self-signed CA and server certificate for Sambmin
# Use when Let's Encrypt isn't available (internal-only deployments)
# Run as root on the FreeBSD server

set -e

DOMAIN="${1:-sambmin.dzsec.net}"
CA_DIR="/usr/local/etc/ssl/sambmin-ca"
CERT_DIR="/usr/local/etc/ssl"
DAYS_CA=3650    # CA valid for 10 years
DAYS_CERT=365   # Server cert valid for 1 year

echo "Generating local CA and server certificate for ${DOMAIN}..."

mkdir -p "${CA_DIR}" "${CERT_DIR}"

# Generate CA key and certificate
if [ ! -f "${CA_DIR}/ca.key" ]; then
    openssl genrsa -out "${CA_DIR}/ca.key" 4096
    openssl req -x509 -new -nodes \
        -key "${CA_DIR}/ca.key" \
        -sha256 -days ${DAYS_CA} \
        -out "${CA_DIR}/ca.crt" \
        -subj "/C=US/ST=Montana/L=Bozeman/O=Sambmin/CN=Sambmin CA"
    echo "CA certificate generated: ${CA_DIR}/ca.crt"
else
    echo "CA already exists, reusing."
fi

# Generate server key
openssl genrsa -out "${CERT_DIR}/${DOMAIN}.key" 2048

# Generate CSR with SAN
openssl req -new \
    -key "${CERT_DIR}/${DOMAIN}.key" \
    -out "${CERT_DIR}/${DOMAIN}.csr" \
    -subj "/C=US/ST=Montana/L=Bozeman/O=Sambmin/CN=${DOMAIN}" \
    -addext "subjectAltName=DNS:${DOMAIN}"

# Sign with CA
openssl x509 -req \
    -in "${CERT_DIR}/${DOMAIN}.csr" \
    -CA "${CA_DIR}/ca.crt" \
    -CAkey "${CA_DIR}/ca.key" \
    -CAcreateserial \
    -out "${CERT_DIR}/${DOMAIN}.crt" \
    -days ${DAYS_CERT} \
    -sha256 \
    -extfile <(printf "subjectAltName=DNS:${DOMAIN}")

# Clean up CSR
rm "${CERT_DIR}/${DOMAIN}.csr"

# Set permissions
chmod 600 "${CERT_DIR}/${DOMAIN}.key"
chmod 644 "${CERT_DIR}/${DOMAIN}.crt"

echo ""
echo "Server certificate: ${CERT_DIR}/${DOMAIN}.crt"
echo "Server key:         ${CERT_DIR}/${DOMAIN}.key"
echo "CA certificate:     ${CA_DIR}/ca.crt"
echo ""
echo "To trust this CA on client machines, distribute ${CA_DIR}/ca.crt"
echo "and add it to the system trust store."
echo ""
echo "Update nginx.conf to use these certificates:"
echo "  ssl_certificate ${CERT_DIR}/${DOMAIN}.crt;"
echo "  ssl_certificate_key ${CERT_DIR}/${DOMAIN}.key;"
