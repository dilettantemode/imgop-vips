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

docker build --platform linux/${PLATFORM} --build-arg PLATFORM=${PLATFORM} -f Dockerfile.build-vips -t libvips-builder:8.17.2 .
docker run cat /home/libvips.zip > libvips.zip

# Move the file to build folder in root project
# Make dir build if not exists
mkdir -p ../build
mv libvips.zip ../build/libvips-${LABEL_DATE}.zip

echo "Wrote: libvips.zip"

