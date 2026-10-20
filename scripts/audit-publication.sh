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
private_dir=""
report="$tmp/secrets.json"
if [ -n "${NEXUS_PRIVATE_AUDIT_DIR:-}" ]; then
  case "$NEXUS_PRIVATE_AUDIT_DIR" in
    /*) ;;
    *) echo "NO-GO: NEXUS_PRIVATE_AUDIT_DIR must be an absolute path" >&2; exit 2 ;;
  esac
  mkdir -p "$NEXUS_PRIVATE_AUDIT_DIR"
  private_dir="$(cd "$NEXUS_PRIVATE_AUDIT_DIR" && pwd -P)"
  repository_dir="$(pwd -P)"
  if [ "$private_dir" = "$repository_dir" ] || [[ "$private_dir" == "$repository_dir/"* ]]; then
    echo "NO-GO: security reports must be outside the Git checkout" >&2
    exit 2
  fi
  chmod 700 "$private_dir"
  report="$private_dir/gitleaks-history-$(git rev-parse --short=12 HEAD).json"
fi
echo "Auditing local refs for SHA $(git rev-parse HEAD). Fetch all remote refs before running."
if ! gitleaks git --no-banner --redact --log-opts="--all" --report-format=json --report-path="$report" . > "$tmp/scanner.log" 2>&1; then
  echo "NO-GO: potential leaks found or scanner failed. Classify privately; rotate confirmed active credentials." >&2
  if [ -n "$private_dir" ]; then
    echo "Redacted findings retained outside the checkout at: $report" >&2
  else
    echo "To retain a redacted report locally, set NEXUS_PRIVATE_AUDIT_DIR to a secure directory outside the checkout." >&2
  fi
  exit 1
fi
git ls-files | grep -Ei '(^|/)(DEV/|TODO/|[^/]*\.(env|pem|p12|pfx|key)$|[^/]*(credential|secret)[^/]*)' > "$tmp/manual.txt" || true
if [ -n "$private_dir" ]; then
  cp "$tmp/manual.txt" "$private_dir/internal-path-review-$(git rev-parse --short=12 HEAD).txt"
fi
count="$(wc -l < "$tmp/manual.txt" | tr -d ' ')"
echo "Internal/sensitive-looking tracked paths requiring human review: $count"
if [ "$count" -gt 0 ]; then
  echo "NO-GO: manually classify internal paths and all public refs before approving publication" >&2
  exit 3
fi
echo "Automated preflight passed for locally available refs; privacy/license review is still required."
