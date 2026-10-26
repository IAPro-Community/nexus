#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
hud_bin="${NEXUS_HUD_BIN:-$repo_root/cmd/nexus-hud/build/bin/nexus-hud}"
evidence_dir="${NEXUS_HUD_EVIDENCE_DIR:-$repo_root/DEV/validation/linux-100-2026-10-25/screenshots/native-hud-persist-xvfb-2026-10-26}"
temp_root="${TMPDIR:-/var/tmp}"
scratch="$(mktemp -d "$temp_root/nexus-hud-persist.XXXXXX")"

mkdir -p "$scratch/home" "$scratch/config/nexus" "$scratch/data/nexus" "$scratch/cache" "$scratch/runtime" "$evidence_dir"
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
unset OPENAI_API_KEY ANTHROPIC_API_KEY GOOGLE_API_KEY GEMINI_API_KEY
unset CLAUDE_CODE_OAUTH_TOKEN CODEX_API_KEY OPENROUTER_API_KEY GH_TOKEN GITHUB_TOKEN CURSOR_API_KEY

if [[ ! -x "$hud_bin" ]]; then
  echo "FAIL: HUD binary is missing or not executable: $hud_bin" >&2
  exit 1
fi

input_helper="$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py"
state_file="$NEXUS_DATA_DIR/hud-window-state.json"
app_pid=""
wm_pid=""

cleanup() {
  if [[ -n "$app_pid" ]] && kill -0 "$app_pid" 2>/dev/null; then
    kill -TERM "$app_pid" 2>/dev/null || true
    for _ in $(seq 1 50); do
      kill -0 "$app_pid" 2>/dev/null || break
      sleep 0.1
    done
    kill -KILL "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
  fi
  if [[ -n "$wm_pid" ]] && kill -0 "$wm_pid" 2>/dev/null; then
    kill -TERM "$wm_pid" 2>/dev/null || true
    wait "$wm_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# Xvfb has no window manager by default, so keyboard focus is not delivered as
# it is in a desktop session. Use the installed WM only inside this Xvfb.
if command -v xfwm4 >/dev/null 2>&1; then
  xfwm4 --replace --compositor=off >"$scratch/xfwm.log" 2>&1 &
  wm_pid=$!
  sleep 2
fi

find_window() {
  for _ in $(seq 1 45); do
    local id
    id="$(xwininfo -root -tree 2>/dev/null | python3 -c '
import re
import sys
for line in sys.stdin:
    if "\"nexus-hud\"" not in line:
        continue
    match = re.search(r"(0x[0-9a-fA-F]+).*?(\d+)x(\d+)[+-]-?\d+[+-]-?\d+", line)
    if match and int(match.group(2)) >= 100 and int(match.group(3)) >= 80:
        print(match.group(1))
        break
')"
    if [[ -n "$id" ]]; then
      printf '%s\n' "$id"
      return 0
    fi
    sleep 1
  done
  return 1
}

window_geometry() {
  xwininfo -id "$1" | awk '
    /Absolute upper-left X:/ { x=$4 }
    /Absolute upper-left Y:/ { y=$4 }
    /Width:/ { w=$2 }
    /Height:/ { h=$2 }
    END { if (x != "" && y != "" && w != "" && h != "") print x, y, w, h }
  '
}

capture_window() {
  local window_id="$1"
  local name="$2"
  local geometry x y width height
  geometry="$(window_geometry "$window_id")"
  read -r x y width height <<<"$geometry"
  xwd -root -silent -out "$scratch/$name.xwd"
  convert "$scratch/$name.xwd" -crop "${width}x${height}+${x}+${y}" +repage "$evidence_dir/$name.png"
  printf '%s\n' "$geometry"
}

launch_hud() {
  local round="$1"
  "$hud_bin" >"$scratch/hud-$round.log" 2>&1 &
  app_pid=$!
  window_id="$(find_window)" || {
    echo "FAIL: HUD window did not open in round $round" >&2
    xwininfo -root -tree >&2 || true
    cat "$scratch/hud-$round.log" >&2 || true
    exit 1
  }
  sleep 3
}

stop_hud() {
  local exit_status
  kill -TERM "$app_pid"
  set +e
  wait "$app_pid"
  exit_status=$?
  set -e
  app_pid=""
  printf 'hud_exit_status=%s\n' "$exit_status"
}

launch_hud first
first_window="$window_id"
first_geometry="$(window_geometry "$first_window")"
read -r initial_x initial_y initial_width initial_height <<<"$first_geometry"
if [[ -z "${initial_width:-}" || "$initial_width" -lt 100 || "$initial_height" -lt 80 ]]; then
  echo "FAIL: invalid initial HUD geometry: $first_geometry" >&2
  exit 1
fi

# Capture the real collapsed mascot, then focus its native window before using
# the keyboard. Xvfb has no interactive desktop to do this automatically.
  capture_window "$first_window" hud-position-initial >/dev/null
python3 "$input_helper" focus "$first_window"
python3 "$input_helper" key TAB
sleep 0.2
capture_window "$first_window" hud-position-focused >/dev/null
python3 "$input_helper" key LEFT
sleep 1
moved_geometry="$(window_geometry "$first_window")"
read -r moved_x moved_y moved_width moved_height <<<"$moved_geometry"
if (( moved_x >= initial_x || moved_y != initial_y )); then
  capture_window "$first_window" hud-position-after-key-attempt >/dev/null
  echo "FAIL: keyboard move did not move HUD left: before=$first_geometry after=$moved_geometry" >&2
  exit 1
fi

python3 - "$state_file" "$moved_x" "$moved_y" "$moved_width" "$moved_height" <<'PY'
import json
import sys
with open(sys.argv[1], encoding='utf-8') as stream:
    state = json.load(stream)
expected = tuple(map(int, sys.argv[2:6]))
actual = (state.get('x'), state.get('y'), state.get('width'), state.get('height'))
if actual != expected:
    raise SystemExit(f"FAIL: persisted geometry {actual} does not match window {expected}")
print(f"persisted_geometry={actual[0]},{actual[1]} {actual[2]}x{actual[3]}")
PY

capture_window "$first_window" hud-position-before-restart >/dev/null
printf 'initial_geometry=%s\n' "$first_geometry"
printf 'moved_geometry=%s\n' "$moved_geometry"
stop_hud

launch_hud second
second_geometry="$(window_geometry "$window_id")"
read -r restored_x restored_y restored_width restored_height <<<"$second_geometry"
if [[ "$restored_x" != "$moved_x" || "$restored_y" != "$moved_y" || "$restored_width" != "$moved_width" || "$restored_height" != "$moved_height" ]]; then
  echo "FAIL: HUD did not restore its geometry: saved=$moved_geometry restored=$second_geometry" >&2
  exit 1
fi
capture_window "$window_id" hud-position-after-restart >/dev/null
printf 'restored_geometry=%s\n' "$second_geometry"
stop_hud

printf 'hud_binary=%s\n' "$hud_bin"
printf 'hud_sha256=%s\n' "$(sha256sum "$hud_bin" | awk '{print $1}')"
printf 'state_file=%s\n' "$state_file"
printf 'isolated_scratch=%s\n' "$scratch"
sha256sum "$evidence_dir/hud-position-before-restart.png" "$evidence_dir/hud-position-after-restart.png"
echo 'result=PASS'
