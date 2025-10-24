#!/usr/bin/env bash
set -euo pipefail

V=8.17.2
IMAGE_NAME="libvips-builder:${V}"
OUT_ZIP_HOST="./libvips-${V}-lambda-x86_64.zip"

# Build for x86_64 (change --platform to linux/arm64 for ARM builds if you have buildx)
docker build --platform linux/amd64 -f deployment-scripts/docker/Dockerfile.build-vips -t ${IMAGE_NAME} .

# Extract the zip file from the scratch-based image using docker cp
# Create a container (but don't run it) and copy the file out
# Note: scratch images need a dummy command even though it won't be executed
CONTAINER_ID=$(docker create ${IMAGE_NAME} /bin/true)
docker cp ${CONTAINER_ID}:/libvips.zip "${OUT_ZIP_HOST}"
docker rm ${CONTAINER_ID}

rm -f ./build/libvips.zip
mv ${OUT_ZIP_HOST} ./build/libvips.zip
echo "Wrote: ${OUT_ZIP_HOST}"

