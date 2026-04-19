#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS="-s -w -X github.com/radial/uetx/internal/version.Version=${VERSION} -X github.com/radial/uetx/internal/version.Commit=${COMMIT}"

platforms=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "windows/amd64"
)

mkdir -p dist

for platform in "${platforms[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output="dist/uetx_${GOOS}_${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    output="${output}.exe"
  fi
  echo "Building ${output}..."
  GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$LDFLAGS" -o "$output" ./cmd/uetx/
done

echo "Done. Binaries in dist/"
