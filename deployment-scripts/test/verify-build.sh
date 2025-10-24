#!/bin/bash
# Verify build artifacts are correct for Lambda deployment

set -e

# Get the script directory and navigate to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${PROJECT_ROOT}"

echo "🔍 Verifying Lambda deployment artifacts..."
echo ""

# Check bootstrap binary
if [ ! -f "build/bootstrap" ]; then
    echo "❌ build/bootstrap not found"
    exit 1
fi

BOOTSTRAP_TYPE=$(file build/bootstrap | grep -o "ELF 64-bit.*")
if [[ "$BOOTSTRAP_TYPE" == *"ELF 64-bit"* ]]; then
    echo "✅ bootstrap is ELF 64-bit executable"
else
    echo "❌ bootstrap is not a valid Linux executable"
    echo "   Found: $(file build/bootstrap)"
    exit 1
fi

# Check app.zip
if [ ! -f "build/app.zip" ]; then
    echo "❌ build/app.zip not found"
    exit 1
fi

APP_SIZE=$(du -h build/app.zip | cut -f1)
echo "✅ app.zip exists ($APP_SIZE)"

# Check libvips.zip
if [ ! -f "build/libvips.zip" ]; then
    echo "❌ build/libvips.zip not found"
    exit 1
fi

LIBVIPS_SIZE=$(du -h build/libvips.zip | cut -f1)
echo "✅ libvips.zip exists ($LIBVIPS_SIZE)"

# Verify libvips.zip contains libraries
LIB_COUNT=$(unzip -l build/libvips.zip | grep -c "\.so" || true)
if [ "$LIB_COUNT" -gt 0 ]; then
    echo "✅ libvips.zip contains $LIB_COUNT shared library files"
else
    echo "❌ libvips.zip doesn't contain any .so files"
    exit 1
fi

# Check if binary links to libvips
if ldd build/bootstrap 2>/dev/null | grep -q "libvips"; then
    echo "✅ bootstrap binary links to libvips"
    echo ""
    echo "📋 Required shared libraries:"
    ldd build/bootstrap | grep -E "(libvips|libglib|libgobject)" | head -5
else
    echo "❌ bootstrap binary doesn't link to libvips"
    exit 1
fi

echo ""
echo "🎉 All checks passed!"
echo ""
echo "📦 Ready to deploy:"
echo "   1. Upload build/libvips.zip as Lambda Layer named 'libvips-runtime'"
echo "   2. Create/update Lambda function with build/app.zip"
echo "   3. Attach the libvips-runtime layer to your function"
echo "   4. Set env var: LD_LIBRARY_PATH=/opt/lib"
echo ""
echo "📖 See DEPLOY_GUIDE.md for detailed instructions"

