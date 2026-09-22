#!/usr/bin/env bash
# Real process/provider/Mission soak for stable release certification.
# This is intentionally separate from the in-memory runner soak harness.
set -euo pipefail

NEXUS_BIN="${NEXUS_BIN:-./nexus}"
DURATION="${NEXUS_LIVE_SOAK_DURATION:-8h}"
PROBE_INTERVAL="${NEXUS_LIVE_SOAK_PROBE_INTERVAL:-60s}"
REQUIRED_PROVIDERS="${SMOKE_REQUIRED_PROVIDERS:-codex agy claude gemini}"
PRESERVE_ROOT=false

duration_seconds() {
    local raw="$1"
    case "$raw" in
        *h) echo $(( ${raw%h} * 3600 )) ;;
        *m) echo $(( ${raw%m} * 60 )) ;;
        *s) echo "${raw%s}" ;;
        *) echo "$raw" ;;
    esac
}

duration_sec="$(duration_seconds "$DURATION")"
interval_sec="$(duration_seconds "$PROBE_INTERVAL")"
if ! [[ "$duration_sec" =~ ^[0-9]+$ ]] || [ "$duration_sec" -lt 60 ]; then
    echo "NEXUS_LIVE_SOAK_DURATION must resolve to at least 60 seconds" >&2
    exit 2
fi
if ! [[ "$interval_sec" =~ ^[0-9]+$ ]] || [ "$interval_sec" -lt 5 ]; then
    echo "NEXUS_LIVE_SOAK_PROBE_INTERVAL must resolve to at least 5 seconds" >&2
    exit 2
fi

if [ -n "${NEXUS_LIVE_SOAK_WORKDIR:-}" ]; then
    root="$NEXUS_LIVE_SOAK_WORKDIR"
    mkdir -p "$root"
    PRESERVE_ROOT=true
else
    root="$(mktemp -d)"
fi
web_data="$root/web-data"
web_log="$root/web.log"
mkdir -p "$web_data"

command -v curl >/dev/null
command -v python3 >/dev/null

port="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(('127.0.0.1', 0))
print(s.getsockname()[1])
s.close()
PY
)"
web_pid=""
restart_done=false

stop_web() {
    if [ -n "${web_pid:-}" ] && kill -0 "$web_pid" 2>/dev/null; then
        kill -TERM "$web_pid" 2>/dev/null || true
        for _ in $(seq 1 30); do
            if ! kill -0 "$web_pid" 2>/dev/null; then
                break
            fi
            sleep 1
        done
        if kill -0 "$web_pid" 2>/dev/null; then
            kill -KILL "$web_pid" 2>/dev/null || true
        fi
        wait "$web_pid" 2>/dev/null || true
    fi
    web_pid=""
}

cleanup() {
    stop_web
    if [ "$PRESERVE_ROOT" != true ]; then
        rm -rf "$root"
    fi
}
trap cleanup EXIT

start_web() {
    stop_web
    echo "[$(date -u +%FT%TZ)] starting Nexus Web on 127.0.0.1:$port" | tee -a "$web_log"
    NEXUS_DATA_DIR="$web_data" AI_CLI_DATA_DIR="$web_data"         "$NEXUS_BIN" web --no-open --listen 127.0.0.1 --port "$port" >>"$web_log" 2>&1 &
    web_pid=$!
    for _ in $(seq 1 60); do
        if ! kill -0 "$web_pid" 2>/dev/null; then
            echo "Nexus Web exited during startup" >&2
            tail -n 200 "$web_log" >&2 || true
            return 1
        fi
        if curl -fsS --max-time 5 "http://127.0.0.1:$port/" >/dev/null; then
            echo "[$(date -u +%FT%TZ)] Nexus Web ready pid=$web_pid" | tee -a "$web_log"
            return 0
        fi
        sleep 1
    done
    echo "Nexus Web did not become ready" >&2
    tail -n 200 "$web_log" >&2 || true
    return 1
}

probe_web() {
    kill -0 "$web_pid" 2>/dev/null || {
        echo "Nexus Web process died during soak" >&2
        tail -n 200 "$web_log" >&2 || true
        return 1
    }
    curl -fsS --max-time 10 "http://127.0.0.1:$port/" >/dev/null
}

live_checkpoint() {
    local label="$1"
    echo "[$(date -u +%FT%TZ)] live checkpoint: $label" | tee -a "$web_log"
    NEXUS_BIN="$NEXUS_BIN"       SMOKE_REQUIRED_PROVIDERS="$REQUIRED_PROVIDERS"       bash ./scripts/smoke-providers.sh
    NEXUS_BIN="$NEXUS_BIN" bash ./scripts/certify-live-mission.sh
}

start_web
live_checkpoint "start"

started="$(date +%s)"
deadline=$((started + duration_sec))
restart_at=$((started + duration_sec / 2))

while [ "$(date +%s)" -lt "$deadline" ]; do
    probe_web
    now="$(date +%s)"
    if [ "$restart_done" = false ] && [ "$now" -ge "$restart_at" ]; then
        echo "[$(date -u +%FT%TZ)] deliberate Web restart checkpoint" | tee -a "$web_log"
        stop_web
        start_web
        restart_done=true
    fi
    remaining=$((deadline - now))
    if [ "$remaining" -le 0 ]; then
        break
    fi
    sleep_for="$interval_sec"
    if [ "$remaining" -lt "$sleep_for" ]; then
        sleep_for="$remaining"
    fi
    sleep "$sleep_for"
done

probe_web
live_checkpoint "end"
echo "[$(date -u +%FT%TZ)] stable live soak PASS duration=$DURATION restart=$restart_done" | tee -a "$web_log"
