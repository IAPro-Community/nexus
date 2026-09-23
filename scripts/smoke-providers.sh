#!/usr/bin/env bash
set -euo pipefail

NEXUS_BIN="${NEXUS_BIN:-nexus}"
PYTHON_BIN="${PYTHON_BIN:-python3}"
REQUIRED_PROVIDERS="${SMOKE_REQUIRED_PROVIDERS:-codex agy claude gemini}"
MARKER="${SMOKE_PROVIDER_MARKER:-NEXUS_PROVIDER_SMOKE_OK}"
TMP_ROOT="${NEXUS_SMOKE_TMPDIR:-${TMPDIR:-/tmp}}"

if [[ -z "${MARKER//[[:space:]]/}" || "$MARKER" == *$'\n'* || "$MARKER" == *$'\r'* ]]; then
  echo "SMOKE_PROVIDER_MARKER must be one non-empty line" >&2
  exit 2
fi

read -r -a providers <<<"$REQUIRED_PROVIDERS"
if (("${#providers[@]}" == 0)); then
  echo "SMOKE_REQUIRED_PROVIDERS must contain at least one provider" >&2
  exit 2
fi
for provider in "${providers[@]}"; do
  if [[ ! "$provider" =~ ^[a-z0-9._-]+$ ]]; then
    echo "invalid provider id in SMOKE_REQUIRED_PROVIDERS: $provider" >&2
    exit 2
  fi
done

mkdir -p "$TMP_ROOT"
usage_file="$TMP_ROOT/nexus-usage-smoke.json"

echo "Refreshing live usage (requires authenticated local profiles)…"
"$NEXUS_BIN" usage --refresh --json >"$usage_file"

"$PYTHON_BIN" - "$usage_file" "$REQUIRED_PROVIDERS" <<'PY'
import json
import sys

path, required = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as handle:
    rows = json.load(handle)

wanted = set(required.split())
live = {
    row.get("provider_id") or row.get("ProviderID", "")
    for row in rows
    if (row.get("status") or row.get("Status", "")) == "LIVE"
}
missing = sorted(wanted - live)
print("LIVE providers:", ", ".join(sorted(live)) or "(none)")
if missing:
    raise SystemExit("missing LIVE for: " + ", ".join(missing))
PY

failures=()
for provider in "${providers[@]}"; do
  out="$TMP_ROOT/nexus-$provider-smoke.txt"
  err="$TMP_ROOT/nexus-$provider-smoke.err"
  rm -f "$out" "$err"

  echo "direct smoke $provider"
  if "$NEXUS_BIN" "$provider" --direct --print "Reply with exactly $MARKER and nothing else." >"$out" 2>"$err"; then
    if "$PYTHON_BIN" - "$out" "$MARKER" <<'PY'
import re
import sys

path, marker = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8", errors="replace") as handle:
    output = handle.read()

output = re.sub(r"\x1b\[[0-?]*[ -/]*[@-~]", "", output)
if not any(line.strip() == marker for line in output.splitlines()):
    raise SystemExit(1)
PY
    then
      echo "PASS $provider"
    else
      echo "FAIL $provider: provider exited 0 but did not return the exact marker" >&2
      failures+=("$provider")
    fi
  else
    code=$?
    echo "FAIL $provider: direct smoke exited $code" >&2
    failures+=("$provider")
  fi
done

if (("${#failures[@]}" > 0)); then
  printf 'provider smoke failed for: %s\n' "${failures[*]}" >&2
  echo "inspect $TMP_ROOT/nexus-<provider>-smoke.{txt,err} for local diagnostics" >&2
  exit 1
fi

echo "smoke-providers PASS: $REQUIRED_PROVIDERS"
