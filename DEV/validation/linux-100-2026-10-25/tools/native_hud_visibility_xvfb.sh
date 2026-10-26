#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
hud_bin="${NEXUS_HUD_BIN:-$repo_root/cmd/nexus-hud/build/bin/nexus-hud}"
input_helper="$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py"
scratch="$(mktemp -d "${TMPDIR:-/var/tmp}/nexus-hud-visibility.XXXXXX")"
mkdir -p "$scratch/home" "$scratch/config/nexus" "$scratch/data/nexus" \
  "$scratch/cache" "$scratch/runtime"
chmod 700 "$scratch/runtime"

export HOME="$scratch/home"
export XDG_CONFIG_HOME="$scratch/config"
export XDG_DATA_HOME="$scratch/data"
export XDG_CACHE_HOME="$scratch/cache"
export XDG_RUNTIME_DIR="$scratch/runtime"
export NEXUS_CONFIG_DIR="$scratch/config/nexus"
export NEXUS_DATA_DIR="$scratch/data/nexus"
export GDK_BACKEND=x11
export LIBGL_ALWAYS_SOFTWARE=1
export GTK_USE_PORTAL=0
export GIO_USE_PORTALS=0
unset OPENAI_API_KEY ANTHROPIC_API_KEY GOOGLE_API_KEY GEMINI_API_KEY
unset CLAUDE_CODE_OAUTH_TOKEN CODEX_API_KEY OPENROUTER_API_KEY GH_TOKEN GITHUB_TOKEN CURSOR_API_KEY

if [[ ! -x "$hud_bin" ]]; then
  echo "FAIL: HUD binary is missing or not executable: $hud_bin" >&2
  exit 1
fi
for tool in xwininfo xfwm4; do
  command -v "$tool" >/dev/null || { echo "BLOCKED: missing X11 tool $tool" >&2; exit 2; }
done

app_pid=""
second_pid=""
wm_pid=""
cleanup() {
  for pid in "$second_pid" "$app_pid"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || true
      for _ in $(seq 1 30); do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.1
      done
      kill -KILL "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
  if [[ -n "$wm_pid" ]] && kill -0 "$wm_pid" 2>/dev/null; then
    kill -TERM "$wm_pid" 2>/dev/null || true
    wait "$wm_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

xfwm4 --replace --compositor=off >"$scratch/xfwm.log" 2>&1 &
wm_pid=$!
sleep 2

find_hud_window() {
  for _ in $(seq 1 40); do
    local id
    id="$(xwininfo -root -tree 2>/dev/null | python3 -c '
import re, sys
windows = []
for line in sys.stdin:
    if "\"nexus-hud\"" not in line:
        continue
    match = re.search(r"(0x[0-9a-fA-F]+).*?(\d+)x(\d+)[+-]-?\d+[+-]-?\d+", line)
    if match and int(match.group(2)) >= 180 and int(match.group(3)) >= 48:
        windows.append((int(match.group(2)) * int(match.group(3)), match.group(1)))
if windows:
    print(max(windows)[1])
')"
    if [[ -n "$id" ]]; then printf '%s\n' "$id"; return 0; fi
    sleep 0.5
  done
  return 1
}

map_state() {
  xwininfo -id "$1" 2>/dev/null | awk -F': ' '/Map State:/ {print $2; exit}'
}

wait_map_state() {
  local id="$1" expected="$2"
  for _ in $(seq 1 40); do
    [[ "$(map_state "$id")" == "$expected" ]] && return 0
    sleep 0.25
  done
  echo "FAIL: window $id did not reach Map State $expected (got $(map_state "$id"))" >&2
  return 1
}

wait_visible_hud() {
  for _ in $(seq 1 40); do
    local id state
    id="$(find_hud_window 2>/dev/null || true)"
    if [[ -n "$id" ]]; then
      state="$(map_state "$id")"
      if [[ "$state" == IsViewable ]]; then
        printf '%s\n' "$id"
        return 0
      fi
    fi
    sleep 0.25
  done
  echo 'FAIL: no stable, visible HUD window appeared' >&2
  xwininfo -root -tree >&2 || true
  return 1
}

launch_hud() {
  local log="$1"
  "$hud_bin" >"$scratch/$log" 2>&1 &
  app_pid=$!
}

launch_second_instance() {
  local log="$1"
  "$hud_bin" >"$scratch/$log" 2>&1 &
  second_pid=$!
  for _ in $(seq 1 40); do
    if ! kill -0 "$second_pid" 2>/dev/null; then break; fi
    sleep 0.25
  done
  if kill -0 "$second_pid" 2>/dev/null; then
    echo "FAIL: second HUD instance did not forward and exit" >&2
    return 1
  fi
  wait "$second_pid"
  second_pid=""
}

stop_hud() {
  local status
  kill -TERM "$app_pid"
  set +e
  wait "$app_pid"
  status=$?
  set -e
  app_pid=""
  printf 'hud_exit_status=%s\n' "$status"
}

state_file="$NEXUS_DATA_DIR/hud-window-state.json"
printf '{"appearance":"TRAY","detachedPopover":false}\n' >"$state_file"
launch_hud tray-startup.log
# Hidden Wails windows may not have an X11 window until first shown. Accept
# either no X11 surface or an explicitly unmapped one, but never a visible HUD.
sleep 3
tray_window="$(find_hud_window 2>/dev/null || true)"
if [[ -n "$tray_window" ]] && [[ "$(map_state "$tray_window")" == IsViewable ]]; then
  echo "FAIL: tray-only HUD was visible at startup (window $tray_window)" >&2
  exit 1
fi
tray_state="not-created"
if [[ -n "$tray_window" ]]; then tray_state="$(map_state "$tray_window")"; fi
printf 'tray_startup=PASS window=%s state=%s\n' "${tray_window:-not-created}" "$tray_state"
launch_second_instance tray-second-instance.log
tray_window="$(wait_visible_hud)" || { cat "$scratch/tray-startup.log" "$scratch/tray-second-instance.log" >&2; exit 1; }
printf 'tray_second_instance_restore=PASS window=%s state=%s\n' "$tray_window" "$(map_state "$tray_window")"
stop_hud

printf '{"appearance":"MASCOT","autoHide":true,"detachedPopover":false}\n' >"$state_file"
launch_hud autohide-startup.log
hud_window="$(wait_visible_hud)" || { cat "$scratch/autohide-startup.log" >&2; exit 1; }
python3 "$input_helper" focus "$hud_window"
python3 "$input_helper" key ESC
wait_map_state "$hud_window" IsUnMapped
printf 'autohide_escape=PASS window=%s state=%s\n' "$hud_window" "$(map_state "$hud_window")"
launch_second_instance autohide-second-instance.log
hud_window="$(wait_visible_hud)"
printf 'autohide_second_instance_restore=PASS window=%s state=%s\n' "$hud_window" "$(map_state "$hud_window")"

stop_hud
printf 'hud_binary=%s\n' "$hud_bin"
printf 'hud_sha256=%s\n' "$(sha256sum "$hud_bin" | awk '{print $1}')"
printf 'isolated_scratch=%s\n' "$scratch"
echo 'result=PASS'
