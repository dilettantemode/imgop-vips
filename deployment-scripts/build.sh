# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Get label today as yyyymmdd
LABEL_DATE=$(date +%Y%m%d)

docker build --output "build" -f Dockerfile.build-vips -t libvips-builder:8.17.2 . 

# Debug: show what was extracted
echo "Contents of build directory:"
ls -lh build/

# Remove app.zip and libvips.zip if exist
rm -f ../build/app.zip
rm -f ../build/libvips.zip

# Move the libvips.zip file from local build directory to project build directory
if [ -f "build/libvips.zip" ]; then
    FILE_SIZE=$(du -h build/libvips.zip | cut -f1)
    mv build/libvips.zip ../build/libvips.zip
    echo "✓ Success: Wrote libvips.zip (${FILE_SIZE}) to ../build/"
    # Clean up the local build directory
    rm -rf build
else
    echo "✗ Error: build/libvips.zip not found"
    echo "Available files in build/:"
    ls -la build/ 2>/dev/null || echo "  (build directory doesn't exist)"
    exit 1
fi

# Zip bootstrap to bootstrap.zip
zip -j ../build/app.zip ../build/bootstrap
