#!/bin/bash
# Local Lambda testing script

set -e

# Get the script directory and navigate to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${PROJECT_ROOT}"

echo "🔨 Building for Lambda..."
make deploy

echo ""
echo "📦 Extracting libvips layer..."
rm -rf /tmp/lambda-layer
mkdir -p /tmp/lambda-layer
unzip -q build/libvips.zip -d /tmp/lambda-layer

echo ""
echo "🏗️  Building test Docker image..."
docker build -t imgop-test -f deployment-scripts/docker/Dockerfile.test .

echo ""
echo "🚀 Starting Lambda container on port 9000..."
echo ""
docker run --rm -p 9000:8080 imgop-test &
DOCKER_PID=$!

# Wait for container to start
sleep 3

echo ""
echo "🧪 Testing Lambda function..."
echo ""
curl -X POST "http://localhost:9000/2015-03-31/functions/function/invocations" \
  -H "Content-Type: application/json" \
  -d @deployment-scripts/test/test-event.json | jq .

# Cleanup
kill $DOCKER_PID 2>/dev/null || true

echo ""
echo "✅ Local test complete!"

