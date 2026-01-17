#!/bin/bash
# Release script - builds and packages for distribution

set -e

cd "$(dirname "$0")"

VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo "dev")}
DATE=$(date -u +%Y-%m-%d)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "=========================================="
echo "db-catalyst Release Builder"
echo "=========================================="
echo "Version: $VERSION"
echo "Date: $DATE"
echo "Commit: $COMMIT"
echo ""

# Clean old builds
rm -rf dist/
mkdir -p dist

# Build for each platform
PLATFORMS=(
    "linux:amd64"
    "linux:arm64"
    "darwin:amd64"
    "darwin:arm64"
    "windows:amd64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS=':' read -r GOOS GOARCH <<< "$PLATFORM"
    EXT=""
    if [ "$GOOS" = "windows" ]; then
        EXT=".exe"
    fi

    OUTPUT="dist/db-catalyst-${VERSION}-${GOOS}-${GOARCH}${EXT}"

    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w -X main.version=${VERSION} -X main.date=${DATE} -X main.commit=${COMMIT}" \
        -o "$OUTPUT" \
        ./cmd/db-catalyst

    # Create checksum
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$OUTPUT" > "${OUTPUT}.sha256"
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$OUTPUT" > "${OUTPUT}.sha256"
    fi

    echo "  → $OUTPUT"
done

echo ""
echo "Release files in dist/:"
ls -la dist/

echo ""
echo "=========================================="
echo "Release build complete!"
echo "=========================================="
