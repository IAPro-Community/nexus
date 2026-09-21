#!/usr/bin/env bash
# restart-web.sh — finds a running `nexus web` process, captures its flags,
# kills it gracefully, and relaunches it fully detached so it survives after
# this script (and the make process that called it) exits.
#
# Usage:  scripts/restart-web.sh [binary-path]
#   binary-path defaults to $(which nexus) or ~/.local/bin/nexus

set -euo pipefail

BIN="${1:-}"
if [ -z "$BIN" ]; then
  BIN="$(command -v nexus 2>/dev/null || echo "$HOME/.local/bin/nexus")"
fi

if [ ! -x "$BIN" ]; then
  echo "✗  nexus binary not found or not executable: $BIN" >&2
  exit 1
fi

# Find the first Web process launched from the binary being installed. This
# specifically catches a Unix process whose executable became "(deleted)"
# after an atomic replacement, while leaving unrelated Nexus installations
# alone.
PID=""
for candidate in $(pgrep -f 'nexus web' 2>/dev/null || true); do
  RAW=$(ps -p "$candidate" -o args= 2>/dev/null || true)
  case "$RAW" in
    *"$BIN web"*)
      PID="$candidate"
      break
      ;;
  esac
done

if [ -z "$PID" ]; then
  echo "ℹ  No running 'nexus web' process found — skipping restart."
  exit 0
fi

# Extract the flags that were passed after `web` (e.g. --listen=0.0.0.0 --remote).
RAW=$(ps -p "$PID" -o args= 2>/dev/null || true)
# Strip everything up to and including 'nexus web', keep the rest.
FLAGS=$(echo "$RAW" | sed 's|.*nexus web||' | sed 's/^ *//')

echo "⟳  Restarting nexus web (PID $PID, flags: $FLAGS)..."

kill "$PID" 2>/dev/null || true
sleep 0.5

# Fully detach: new session + redirect all std fds + disown.
# shellcheck disable=SC2086
setsid "$BIN" web $FLAGS >/tmp/nexus-web.log 2>&1 </dev/null &
NEWPID=$!
disown "$NEWPID" 2>/dev/null || true

sleep 0.8

if kill -0 "$NEWPID" 2>/dev/null; then
  echo "✓  nexus web restarted (PID $NEWPID). Log: /tmp/nexus-web.log"
else
  echo "✗  Process died — check /tmp/nexus-web.log:" >&2
  cat /tmp/nexus-web.log >&2
  exit 1
fi
