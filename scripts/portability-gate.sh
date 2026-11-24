#!/usr/bin/env bash
# Portability gate: proves, from a Linux host, that the release binary and the
# test graph build for every supported target and that OS-sensitive packages
# keep their contract when executed on the host.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

export GOTOOLCHAIN=local
export GOFLAGS=-mod=readonly
export GOCACHE="${GOCACHE:-/tmp/opencode/go-build}"

targets=(linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64)

echo "==> cross-building the release CLI for ${#targets[@]} targets"
for target in "${targets[@]}"; do
  GOOS="${target%/*}" GOARCH="${target#*/}" CGO_ENABLED=0 \
    go build -trimpath -buildvcs=false -o /dev/null ./cmd/nexus
  echo "    built $target"
done

echo "==> compiling the test graph for windows and darwin"
for target in windows/amd64 darwin/arm64; do
  GOOS="${target%/*}" GOARCH="${target#*/}" CGO_ENABLED=0 \
    go test -exec=/bin/true -run '^$' \
      ./cmd/nexus ./internal/runtime ./internal/browser ./internal/update \
      ./internal/control/... ./internal/core/... ./internal/nexus/... >/dev/null
  echo "    compiled $target"
done

echo "==> vetting every package on the host"
go vet ./...

echo "==> running OS-sensitive packages on the host"
go test -count=1 \
  ./internal/app \
  ./internal/runtime \
  ./internal/doctor \
  ./internal/core/device \
  ./internal/release \
  ./internal/core/memoryrecovery

echo "portability gate passed"
