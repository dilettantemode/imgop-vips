#!/bin/bash
# Test Lambda function locally using AWS SAM CLI
# Install SAM CLI: https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html

set -e

# Get the script directory and navigate to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${PROJECT_ROOT}"

echo "🔨 Preparing test environment..."

# Use the already-built bootstrap binary
if [ ! -f "build/bootstrap" ]; then
    echo "❌ build/bootstrap not found. Run 'make dev' first."
    exit 1
fi

# Extract libvips layer locally for testing
echo "📦 Extracting libvips layer..."
rm -rf /tmp/lambda-layer
mkdir -p /tmp/lambda-layer/lib
unzip -q build/libvips.zip -d /tmp/lambda-layer/

echo ""
echo "🧪 Testing with AWS SAM CLI..."
echo ""

# Test using SAM local
sam local invoke ImgOpFunction \
    --template deployment-scripts/test/template.yaml \
    --event deployment-scripts/test/test-event.json \
    --docker-volume-basedir /tmp/lambda-layer \
    --env-vars "LD_LIBRARY_PATH=/opt/lib"

echo ""
echo "✅ Test complete!"

