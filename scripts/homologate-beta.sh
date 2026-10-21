#!/usr/bin/env bash
set -euo pipefail

export GOTOOLCHAIN=local
export GOFLAGS=-mod=readonly

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

EXPECTED_VERSION="$(tr -d '\r\n' < VERSION)"
SHA="$(git rev-parse HEAD)"
SHORT_SHA="$(git rev-parse --short HEAD)"
if [ -n "${NEXUS_EXPECTED_SHA:-}" ]; then
  RESOLVED_EXPECTED_SHA="$(git rev-parse "$NEXUS_EXPECTED_SHA" 2>/dev/null || printf '%s' "$NEXUS_EXPECTED_SHA")"
  if [ "$SHA" != "$RESOLVED_EXPECTED_SHA" ]; then
    echo "homologation SHA mismatch: HEAD=$SHA expected=$RESOLVED_EXPECTED_SHA" >&2
    exit 1
  fi
fi
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
OUT="DEV/validation/local-beta/${EXPECTED_VERSION}/${SHA}/${OS}-${ARCH}"
mkdir -p "$OUT"

log() { printf '\n==> %s\n' "$*"; }
require() { command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }

require git
require go
require bun

if [ "$OS" != "linux" ]; then
  echo "Linux beta homologation requires Linux; got $OS" >&2
  exit 1
fi
case "$ARCH" in
  x86_64|amd64) ;;
  *) echo "Linux beta homologation requires amd64; got $ARCH" >&2; exit 1 ;;
esac

GO_VERSION_TEXT="$(go version)"
if [[ "$GO_VERSION_TEXT" != *"go1.26.10"* ]]; then
  echo "Linux beta homologation requires Go 1.26.10 to match the patched release toolchain; got: $GO_VERSION_TEXT" >&2
  exit 1
fi
BUN_VERSION="$(bun --version)"
if [ "$BUN_VERSION" != "1.3.9" ]; then
  echo "Linux beta homologation requires Bun 1.3.9 to match the patched release toolchain; got: $BUN_VERSION" >&2
  exit 1
fi

CANONICAL_PUBLIC_KEY="$(tr -d '[:space:]' < UPDATE_PUBLIC_KEY | tr '[:upper:]' '[:lower:]')"
if ! [[ "$CANONICAL_PUBLIC_KEY" =~ ^[0-9a-f]{64}$ ]]; then
  echo "UPDATE_PUBLIC_KEY must contain the canonical 32-byte Ed25519 public key as 64 hex characters" >&2
  exit 1
fi
if [ -n "${NEXUS_UPDATE_PUBLIC_KEY:-}" ] && [ "$(printf '%s' "$NEXUS_UPDATE_PUBLIC_KEY" | tr '[:upper:]' '[:lower:]')" != "$CANONICAL_PUBLIC_KEY" ]; then
  echo "NEXUS_UPDATE_PUBLIC_KEY does not match checked-in UPDATE_PUBLIC_KEY" >&2
  exit 1
fi
export NEXUS_UPDATE_PUBLIC_KEY="$CANONICAL_PUBLIC_KEY"

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
  echo "update_public_key=$CANONICAL_PUBLIC_KEY"
  date -u '+started_at=%Y-%m-%dT%H:%M:%SZ'
} | tee "$OUT/environment.txt"

log "documentation"
bun scripts/docs-verify.mjs 2>&1 | tee "$OUT/docs-verify.log"

log "frontend dependencies"
bun install --cwd web --frozen-lockfile 2>&1 | tee "$OUT/bun-install.log"

log "Playwright Chromium"
(
  cd web
  bunx playwright install chromium
) 2>&1 | tee "$OUT/playwright-chromium.log"

log "frontend quality"
for task in format:check typecheck lint lint:styles check:styles test build verify; do
  echo "--- bun run $task" | tee -a "$OUT/frontend.log"
  bun run --cwd web "$task" 2>&1 | tee -a "$OUT/frontend.log"
done

log "Go formatting"
tracked_go="$(git ls-files -- '*.go')"
unformatted=""
if [ -n "$tracked_go" ]; then
  unformatted="$(printf '%s\n' "$tracked_go" | xargs gofmt -l)"
fi
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted" | tee "$OUT/gofmt-fail.txt"
  exit 1
fi

log "Go vet"
go vet ./... 2>&1 | tee "$OUT/go-vet.log"

log "Go lint (pinned)"
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --timeout=5m ./... 2>&1 | tee "$OUT/golangci-lint.log"

log "Go security (pinned)"
GOTOOLCHAIN=go1.26.10 go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./... 2>&1 | tee "$OUT/govulncheck.log"

log "Version contract"
go run ./scripts/verify-version-contract.go . 2>&1 | tee "$OUT/version-contract.log"

TEST_HOME="$OUT/isolated-test-home"
mkdir -p "$TEST_HOME"
EXISTING_GOPATH="$(go env GOPATH)"
EXISTING_GOMODCACHE="$(go env GOMODCACHE)"
EXISTING_GOCACHE="$(go env GOCACHE)"
run_isolated_go() {
  env \
    HOME="$TEST_HOME" \
    USERPROFILE="$TEST_HOME" \
    XDG_CONFIG_HOME="$TEST_HOME/.config" \
    XDG_DATA_HOME="$TEST_HOME/.local/share" \
    XDG_CACHE_HOME="$TEST_HOME/.cache" \
    NEXUS_DATA_DIR="$TEST_HOME/nexus-data" \
    AI_MANAGER_DATA_DIR="$TEST_HOME/nexus-data" \
    AI_CLI_DATA_DIR="$TEST_HOME/nexus-data" \
    GOPATH="$EXISTING_GOPATH" \
    GOMODCACHE="$EXISTING_GOMODCACHE" \
    GOCACHE="$EXISTING_GOCACHE" \
    "$@"
}

log "provider-native Mission/terminal regression preflight"
run_isolated_go go test -race -count=1 ./internal/app ./internal/nexus/promotion ./internal/runtime 2>&1 | tee "$OUT/regression-interactive-terminal.log"

log "Go race suite (isolated profile)"
run_isolated_go go test -race -count=1 ./... 2>&1 | tee "$OUT/go-test-race.log"

log "control/runtime E2E"
run_isolated_go go test -race -count=1 ./internal/control/terminal/... ./internal/control/protocol/... ./internal/control/host/... ./internal/control/web/... 2>&1 | tee "$OUT/control-e2e.log"

log "Agent Interop / OpenCode Projection regressions"
run_isolated_go go test -race -count=1 ./internal/core/provider/agentprojection/opencode ./internal/control/launcher ./internal/control/web ./internal/nexus ./internal/nexus/store 2>&1 | tee "$OUT/agent-interop.log"

log "build CLI"
make build 2>&1 | tee "$OUT/build-cli.log"

./nexus version --json | tee "$OUT/version.json"
grep -Fq "\"version\":\"$EXPECTED_VERSION\"" "$OUT/version.json" || \
  grep -Fq "\"version\": \"$EXPECTED_VERSION\"" "$OUT/version.json"

log "Browser E2E on the current Linux binary"
SAVED_SKIP_BUILD="${SKIP_BUILD-__NEXUS_UNSET__}"
export SKIP_BUILD=1
for task in \
  test:e2e \
  test:e2e:composer \
  test:e2e:surface \
  test:e2e:capacity \
  test:e2e:hud \
  test:e2e:composer-spine \
  test:e2e:composer-vague-spine \
  test:e2e:composer-themes \
  test:a11y \
  test:visual
do
  log_file="$OUT/$(printf '%s' "$task" | tr ':' '-').log"
  bun run --cwd web "$task" 2>&1 | tee "$log_file"
done
if [ "$SAVED_SKIP_BUILD" = "__NEXUS_UNSET__" ]; then
  unset SKIP_BUILD
else
  export SKIP_BUILD="$SAVED_SKIP_BUILD"
fi

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

log "GoReleaser snapshot (Linux amd64)"
# GoReleaser v2.18.0 requires Go >= 1.27.0, so `go run ...@v2.18.0` cannot run
# under this script's GOTOOLCHAIN=local + Go 1.25.14 contract. Use the pinned
# standalone binary (same distribution as goreleaser-action@v6 in CI) and keep
# every product build on Go 1.25.14.
GORELEASER_VERSION="v2.18.0"
export PATH="$(go env GOPATH)/bin:$PATH"
if ! command -v goreleaser >/dev/null 2>&1; then
  log "bootstrapping GoReleaser $GORELEASER_VERSION (build tool only)"
  GOTOOLCHAIN=auto GOFLAGS= go install "github.com/goreleaser/goreleaser/v2@${GORELEASER_VERSION}"
fi
GORELEASER_GIT_VERSION="$(goreleaser --version 2>/dev/null | sed -n 's/^GitVersion:[[:space:]]*//p' | head -n1)"
if [ "$GORELEASER_GIT_VERSION" != "$GORELEASER_VERSION" ]; then
  echo "GoReleaser $GORELEASER_VERSION is required; got: ${GORELEASER_GIT_VERSION:-unknown}" >&2
  exit 1
fi
# Snapshot artifacts must report the release version from VERSION (the beta tag
# is only created after homologation), otherwise the DEB smoke below is unable
# to assert the packaged identity.
export NEXUS_VERSION="$EXPECTED_VERSION"
goreleaser release --snapshot --clean 2>&1 | tee "$OUT/goreleaser-snapshot.log"
test -f dist/checksums.txt
test -f dist/nexus_Linux_x86_64.tar.gz
test "$(find dist -maxdepth 1 -name 'nexus_Linux_*.tar.gz' | wc -l)" -eq 1
DEB_PATH="$(find dist -maxdepth 1 \( -name '*_linux_amd64.deb' -o -name '*_linux_x86_64.deb' \) | head -n1)"
test -n "$DEB_PATH"
test "$(find dist -maxdepth 1 -name '*.rpm' | wc -l)" -eq 0
test "$(find dist -maxdepth 1 \( -name '*Darwin*' -o -name '*Windows*' -o -name '*arm64*' \) | wc -l)" -eq 0

log "DEB inspect/extract/execute smoke"
require dpkg-deb
dpkg-deb --info "$DEB_PATH" > "$OUT/deb-info.txt"
DEB_ROOT="$OUT/deb-root"
rm -rf "$DEB_ROOT"
mkdir -p "$DEB_ROOT"
dpkg-deb -x "$DEB_PATH" "$DEB_ROOT"
DEB_NEXUS="$(find "$DEB_ROOT" -type f -name nexus -perm -111 | head -n1)"
test -n "$DEB_NEXUS"
"$DEB_NEXUS" version --json | tee "$OUT/deb-version.json"
grep -Fq "\"version\":\"$EXPECTED_VERSION\"" "$OUT/deb-version.json" || grep -Fq "\"version\": \"$EXPECTED_VERSION\"" "$OUT/deb-version.json"

cat > "$OUT/MANUAL_CHECKS.md" <<'EOF'
# Manual beta smoke still required

These checks require a real interactive/native session and are intentionally not faked by the script.

- Start `nexus codex`; press Up/Down/Left/Right repeatedly. No literal `[A`, `[B`, `[C`, `[D` may appear.
- Start `nexus codex` and `nexus agy` with no Lead/delegation flags. Submit both a trivial prompt and a complex request such as "corrija a API". Neither may create, require, or ask for a Mission; provider-native Enter/commands must continue normally.
- Start `nexus codex --lead` with `NEXUS_DELEGATION_MODE` unset. The Lead may provide Nexus context, but ordinary prompts still must not be promoted to Mission because the persisted delegation mode starts OFF.
- Opt in deliberately with `NEXUS_DELEGATION_MODE=ASK` plus `--lead`. A Mission-worthy prompt may ask for promotion, while a simple prompt stays provider-owned. Repeat with `AUTO` only as an explicit orchestration smoke.
- In normal provider mode, submit prompts containing `permission`, `submission` and `transmission`. They must never be treated as explicit Mission requests.
- Explicitly request a Mission while strict admission is intentionally unavailable. The prompt must NOT execute as Direct; Nexus must show the exact blocker and retain the rejected plan id for remediation.
- With a hard/emergency cost budget requiring confirmation, an automatic heuristic promotion must not bypass the budget gate. It may return to Lead before a run exists; an explicit Mission request must stay blocked until the current cost state is explicitly confirmed.
- Keep the project default isolation as `project` and run an AUTO/CREATE Mission in a real Git repository. Nexus must use/create a worktree-safe Agent instead of blocking only because the project default is not `worktree`.
- Pin an EXISTING Agent with `Isolation=project` and attempt an autonomous Mission. Admission must fail before `run_id` with an actionable `worktree_isolation` blocker.
- Set project default to `worktree` but explicitly configure the pinned Agent with `Isolation=project`. The Agent override must win and admission must still fail; autonomous execution must never reach the canonical checkout.
- Open Agents and create/edit an Agent. OpenCode must be offered independently from AGY.
- Open Usage. If OpenCode is installed but has no profile yet, it must still be shown as an installed provider.
- Select an OpenCode Review or Investigate execution role. Nexus must materialize/discover the projected agent without invoking the model.
- Click Certify only after acknowledging quota/cost. Activation must select the projected agent; enforcement must prove effective write-tool denial/removal, keep the canary absent and only then show ENFORCED.
- Repeat the custom-target smoke with the provider absent on the host when feasible. For VPN Dev Workspace on Windows, use target_workspace=/workspace; no host-visible profile home means overlay/BEST_EFFORT rather than a fabricated ENFORCED state.
- Launch Nexus HUD and verify show/hide, autostart toggle and basic projection.
- If voice is configured, verify microphone permission, capture, transcription, provider fallback, terminal insertion without automatic Enter, and one non-destructive global voice command.
EOF

date -u '+completed_at=%Y-%m-%dT%H:%M:%SZ' | tee -a "$OUT/environment.txt"
printf '\nPASS automated Linux beta homologation: %s @ %s\nEvidence: %s\n' "$EXPECTED_VERSION" "$SHA" "$OUT"
