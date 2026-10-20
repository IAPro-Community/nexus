#!/usr/bin/env bash
# Read-only public history preflight; a clean result is not release certification.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
[ "$(git rev-parse --is-shallow-repository)" = false ] || { echo "NO-GO: shallow history" >&2; exit 2; }
command -v gitleaks >/dev/null || { echo "NO-GO: gitleaks is required" >&2; exit 2; }
git diff --quiet && git diff --cached --quiet || { echo "NO-GO: commit local changes first" >&2; exit 2; }
umask 077
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
echo "Auditing local refs for SHA $(git rev-parse HEAD). Fetch all refs before running."
if ! gitleaks git --redact --log-opts="--all" --report-format=json --report-path="$tmp/secrets.json" .; then
  echo "NO-GO: detected possible leaks or scanner failure; inspect locally, rotate confirmed secrets" >&2
  exit 1
fi
git ls-files | grep -Ei '(^|/)(DEV/|TODO/|[^/]*\.(env|pem|p12|pfx|key)$|[^/]*(credential|secret)[^/]*)' > "$tmp/manual.txt" || true
count="$(wc -l < "$tmp/manual.txt" | tr -d ' ')"
echo "Internal/sensitive-looking tracked paths requiring human review: $count"
if [ "$count" -gt 0 ]; then
  echo "NO-GO: manually classify internal paths and all public refs before approving publication" >&2
  exit 3
fi
echo "Automated preflight passed for locally available refs; privacy/license review is still required."
