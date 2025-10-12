# Script for runnin the dockerfile build, it will accept param platform x86_64 or arm64

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Get label today as yyyymmdd
LABEL_DATE=$(date +%Y%m%d)

if [ -z "$1" ]; then
    echo "Usage: $0 <platform>"
    echo "Platform options: x86_64, arm64"
    exit 1
fi

PLATFORM=""

if [ "$1" == "x86_64" ]; then
    PLATFORM="amd64"
elif [ "$1" == "arm64" ]; then 
    PLATFORM="arm64"
else
    echo "Invalid platform: $1"
    echo "Platform options: x86_64, arm64"
    exit 1
fi

docker build --platform linux/${PLATFORM} --build-arg PLATFORM=${PLATFORM} --output "build" -f Dockerfile.build-vips -t libvips-builder:8.17.2 . 

# Debug: show what was extracted
echo "Contents of build directory:"
ls -lh build/

# Move the file to build folder in root project
# Make dir build if not exists
mkdir -p ../build

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
