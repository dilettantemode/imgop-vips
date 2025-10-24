#!/usr/bin/env bash
set -euo pipefail

V=8.17.2
IMAGE_NAME="libvips-builder:${V}"
OUT_ZIP_HOST="./libvips-${V}-lambda-x86_64.zip"

# Build for x86_64 (change --platform to linux/arm64 for ARM builds if you have buildx)
docker build --no-cache --platform linux/amd64 -f deployment-scripts/Dockerfile.build-vips -t ${IMAGE_NAME} .

# run the container and capture the zip emitted by CMD
docker run --rm ${IMAGE_NAME} cat /libvips-${V}-lambda.zip > "${OUT_ZIP_HOST}"

echo "Wrote: ${OUT_ZIP_HOST}"

