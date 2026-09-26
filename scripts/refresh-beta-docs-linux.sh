#!/usr/bin/env bash
set -euo pipefail

export GOTOOLCHAIN=local
export GOFLAGS=-mod=readonly

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

EXPECTED_SHA="${NEXUS_EXPECTED_SHA:-}"
SHA="$(git rev-parse HEAD)"

if [ -n "$EXPECTED_SHA" ]; then
  RESOLVED_EXPECTED_SHA="$(git rev-parse "$EXPECTED_SHA" 2>/dev/null || printf '%s' "$EXPECTED_SHA")"
  if [ "$SHA" != "$RESOLVED_EXPECTED_SHA" ]; then
    echo "documentation capture SHA mismatch: HEAD=$SHA expected=$RESOLVED_EXPECTED_SHA" >&2
    exit 1
  fi
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "working tree is dirty; documentation evidence must be captured from an exact clean SHA" >&2
  exit 1
fi

command -v git >/dev/null
command -v go >/dev/null
command -v bun >/dev/null

echo "==> Install locked frontend dependencies"
bun install --cwd web --frozen-lockfile

echo "==> Ensure Playwright Chromium"
(
  cd web
  bunx playwright install chromium
)

echo "==> Capture documentation evidence from $SHA"
(
  cd web
  bun run docs:capture
)

echo "==> Verify captured documentation"
bun scripts/docs-verify.mjs

MANIFEST="$ROOT/docs/assets/screenshots/manifest.json"
test -f "$MANIFEST"
MANIFEST_SHA="$(node -e 'const fs=require("fs"); const m=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); process.stdout.write(String(m.source_sha||""))' "$MANIFEST")"
if [ "$MANIFEST_SHA" != "$SHA" ]; then
  echo "screenshot manifest source_sha=$MANIFEST_SHA does not match captured HEAD=$SHA" >&2
  exit 1
fi

mapfile -t EVIDENCE_CHANGES < <(git status --porcelain -- docs/assets/screenshots docs/assets/demos)
if [ "${#EVIDENCE_CHANGES[@]}" -eq 0 ]; then
  echo "capture completed but no documentation evidence changed; inspect docs-capture output before certifying" >&2
  exit 1
fi

printf '\nDocumentation evidence captured and verified for exact SHA %s.\n' "$SHA"
echo "Review and commit ONLY the generated documentation evidence before final homologation:"
printf '  %s\n' "${EVIDENCE_CHANGES[@]}"
echo
echo "Do not edit manifest.source_sha manually. After committing evidence, rerun scripts/homologate-beta.sh from the new exact SHA."
