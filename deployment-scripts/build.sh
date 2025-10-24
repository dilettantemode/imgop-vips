#!/usr/bin/env bash
set -euo pipefail

echo "Building bootstrap binary in Amazon Linux 2 (GLIBC 2.26)..."

# Build using Docker
docker build --output build -f deployment-scripts/docker/Dockerfile.build-bootstrap -t bootstrap-builder:al2 .

echo "✅ Bootstrap binary built for Amazon Linux 2"
echo ""
echo "Verifying binary..."
file build/bootstrap
echo ""
echo "Checking GLIBC version requirements..."
objdump -T build/bootstrap 2>/dev/null | grep GLIBC | sed 's/.*GLIBC_/GLIBC_/' | sort -Vu | tail -3 || echo "Could not check GLIBC version"

