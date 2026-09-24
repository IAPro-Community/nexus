#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

EXPECTED_VERSION="$(tr -d '\r\n' < VERSION)"
SHA="$(git rev-parse HEAD)"
SHORT_SHA="$(git rev-parse --short HEAD)"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
OUT="DEV/validation/local-beta2/${SHA}/${OS}-${ARCH}"
mkdir -p "$OUT"

log() { printf '\n==> %s\n' "$*"; }
require() { command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }

require git
require go
require bun

if [ "${NEXUS_ALLOW_DIRTY:-0}" != "1" ] && [ -n "$(git status --porcelain)" ]; then
  echo "working tree is dirty; homologation must run on an exact SHA (set NEXUS_ALLOW_DIRTY=1 only for exploratory runs)" >&2
  exit 1
fi

{
  echo "sha=$SHA"
  echo "short_sha=$SHORT_SHA"
  echo "version=$EXPECTED_VERSION"
  echo "os=$OS"
  echo "arch=$ARCH"
  echo "go=$(go version)"
  echo "bun=$(bun --version)"
  date -u '+started_at=%Y-%m-%dT%H:%M:%SZ'
} | tee "$OUT/environment.txt"

log "documentation"
bun scripts/docs-verify.mjs 2>&1 | tee "$OUT/docs-verify.log"

log "frontend dependencies"
bun --cwd web install --frozen-lockfile 2>&1 | tee "$OUT/bun-install.log"

log "frontend quality"
for task in format:check typecheck lint lint:styles check:styles test build verify; do
  echo "--- bun run $task" | tee -a "$OUT/frontend.log"
  bun --cwd web run "$task" 2>&1 | tee -a "$OUT/frontend.log"
done

log "Go formatting"
unformatted="$(gofmt -l . | grep -v '^\.\/\.worktrees\/' || true)"
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted" | tee "$OUT/gofmt-fail.txt"
  exit 1
fi

log "Go vet"
go vet ./... 2>&1 | tee "$OUT/go-vet.log"

log "Go race suite"
go test -race ./... 2>&1 | tee "$OUT/go-test-race.log"

log "regression suites: interactive routing and terminal input"
go test -race -count=1 ./internal/nexus/promotion ./internal/runtime 2>&1 | tee "$OUT/regression-interactive-terminal.log"

log "control/runtime E2E"
go test -race -count=1 ./internal/control/terminal/... ./internal/control/protocol/... ./internal/control/host/... ./internal/control/web/... 2>&1 | tee "$OUT/control-e2e.log"

log "build CLI"
make build 2>&1 | tee "$OUT/build-cli.log"

./nexus version --json | tee "$OUT/version.json"
grep -Fq "\"version\":\"$EXPECTED_VERSION\"" "$OUT/version.json" ||   grep -Fq "\"version\": \"$EXPECTED_VERSION\"" "$OUT/version.json"

./nexus doctor --json | tee "$OUT/doctor.json"
./nexus providers status --json | tee "$OUT/providers.json"
./nexus usage --json | tee "$OUT/usage.json" || true

if command -v opencode >/dev/null 2>&1; then
  log "OpenCode visibility regression"
  grep -qi 'opencode' "$OUT/providers.json" || {
    echo "OpenCode is installed but absent from nexus providers status --json" >&2
    exit 1
  }
  grep -qi 'opencode' "$OUT/usage.json" || {
    echo "OpenCode is installed but absent from nexus usage --json" >&2
    exit 1
  }
fi

if [ "${NEXUS_SKIP_DESKTOP:-0}" != "1" ]; then
  log "native Desktop + HUD build"
  make build-desktop-wails 2>&1 | tee "$OUT/build-desktop.log"
  make build-hud-wails 2>&1 | tee "$OUT/build-hud.log"
fi

cat > "$OUT/MANUAL_CHECKS.md" <<'EOF'
# Manual beta smoke still required

These checks require a real interactive/native session and are intentionally not faked by the script.

- Start `nexus codex`; press Up/Down/Left/Right repeatedly. No literal `[A`, `[B`, `[C`, `[D` may appear.
- In Direct/Lead mode, submit a simple request. It must remain with the Lead and must not be promoted to Mission unless decomposition/delegation/explicit Mission intent is present.
- Open Agents and create/edit an Agent. OpenCode must be offered independently from AGY.
- Open Usage. If OpenCode is installed but has no profile yet, it must still be shown as an installed provider.
- Launch Nexus HUD and verify show/hide, autostart toggle and basic projection.
- If voice is configured, verify microphone permission, transcription, provider fallback and terminal insertion without automatic Enter.
EOF

date -u '+completed_at=%Y-%m-%dT%H:%M:%SZ' | tee -a "$OUT/environment.txt"
printf '\nPASS automated local beta homologation: %s @ %s\nEvidence: %s\n' "$EXPECTED_VERSION" "$SHA" "$OUT"
