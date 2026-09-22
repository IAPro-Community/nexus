#!/usr/bin/env bash
# Real-provider black-box Mission certification.
# Requires at least one authenticated provider/profile eligible for autonomous
# execution on the host. This is intentionally NOT part of ordinary CI.
set -euo pipefail

NEXUS_BIN="${NEXUS_BIN:-./nexus}"
TIMEOUT_BIN="${TIMEOUT_BIN:-timeout}"
MISSION_TIMEOUT="${NEXUS_LIVE_MISSION_TIMEOUT:-45m}"
MARKER="${NEXUS_LIVE_MISSION_MARKER:-NEXUS_LIVE_MISSION_OK}"

if [ ! -x "$NEXUS_BIN" ] && ! command -v "$NEXUS_BIN" >/dev/null 2>&1; then
    echo "NEXUS_BIN is not executable: $NEXUS_BIN" >&2
    exit 2
fi
command -v git >/dev/null
command -v node >/dev/null
command -v "$TIMEOUT_BIN" >/dev/null

root="$(mktemp -d)"
project="$root/project"
data="$root/data"
mkdir -p "$project" "$data"

cleanup() {
    rm -rf "$root"
}
trap cleanup EXIT

git -C "$project" init -q
git -C "$project" config user.email "nexus-certification@example.invalid"
git -C "$project" config user.name "Nexus Certification"

cat > "$project/package.json" <<'JSON'
{
  "name": "nexus-live-mission-certification",
  "private": true,
  "scripts": {
    "test": "node verify.mjs"
  }
}
JSON

cat > "$project/verify.mjs" <<EOF
import { readFileSync } from 'node:fs';
const value = readFileSync('mission-output.txt', 'utf8').trim();
if (value !== '${MARKER}') {
  throw new Error(`mission-output.txt=${JSON.stringify(value)}, expected ${JSON.stringify('${MARKER}')}`);
}
console.log('NEXUS_LIVE_MISSION_VERIFIED');
EOF

cat > "$project/README.md" <<EOF
# Nexus live Mission certification fixture

The certification task must create `mission-output.txt` containing exactly:

```
${MARKER}
```

Verification is `npm test`.
EOF

git -C "$project" add package.json verify.mjs README.md
git -C "$project" commit -qm "certification fixture"

export NEXUS_DATA_DIR="$data"
export AI_CLI_DATA_DIR="$data"

"$NEXUS_BIN" projects add "$project" >"$root/project-add.log"

goal="In this project create mission-output.txt containing exactly ${MARKER}. Do not change package.json or verify.mjs. Run npm test. The task is complete only when npm test exits successfully and the final Mission verification passes."

(
    cd "$project"
    "$TIMEOUT_BIN" "$MISSION_TIMEOUT" "$NEXUS_BIN" run "$goal"
) >"$root/mission.json" 2>"$root/mission.stderr" || {
    code=$?
    echo "live Mission command failed (exit=$code)" >&2
    cat "$root/mission.stderr" >&2 || true
    cat "$root/mission.json" >&2 || true
    exit "$code"
}

python3 - "$root/mission.json" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
raw = path.read_text(encoding="utf-8").strip()
if not raw:
    raise SystemExit("live Mission produced no JSON result")
try:
    result = json.loads(raw)
except json.JSONDecodeError as exc:
    raise SystemExit(f"live Mission output is not one JSON object: {exc}\n{raw}")
if result.get("state") != "COMPLETED_VERIFIED":
    raise SystemExit(f"live Mission state={result.get('state')!r}, expected COMPLETED_VERIFIED")
if not result.get("run_id"):
    raise SystemExit("live Mission result has no run_id")
print("live Mission PASS:", result["run_id"])
PY

echo "certify-live-mission PASS"
