#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
hud_bin="${NEXUS_HUD_BIN:-$repo_root/cmd/nexus-hud/build/bin/nexus-hud}"
evidence_dir="${NEXUS_HUD_EVIDENCE_DIR:-$repo_root/DEV/validation/linux-100-2026-10-25/screenshots/native-hud-current-build-xvfb-2026-10-26}"
temp_root="${TMPDIR:-/var/tmp}"
scratch="$(mktemp -d "$temp_root/nexus-hud-current-build-xvfb.XXXXXX")"
mkdir -p "$scratch/home" "$scratch/config" "$scratch/data" "$scratch/cache" "$scratch/runtime" "$evidence_dir"
chmod 700 "$scratch/runtime"

export HOME="$scratch/home"
export XDG_CONFIG_HOME="$scratch/config"
export XDG_DATA_HOME="$scratch/data"
export XDG_CACHE_HOME="$scratch/cache"
export XDG_RUNTIME_DIR="$scratch/runtime"
export NEXUS_DATA_DIR="$scratch/data/nexus"
export GDK_BACKEND=x11
export LIBGL_ALWAYS_SOFTWARE=1
unset OPENAI_API_KEY ANTHROPIC_API_KEY GOOGLE_API_KEY GEMINI_API_KEY
unset CLAUDE_CODE_OAUTH_TOKEN CODEX_API_KEY OPENROUTER_API_KEY GH_TOKEN GITHUB_TOKEN

if [[ ! -x "$hud_bin" ]]; then
  echo "FAIL: HUD binary is missing or not executable: $hud_bin" >&2
  exit 1
fi

"$hud_bin" >"$scratch/hud.log" 2>&1 &
app_pid=$!
cleanup() {
  if kill -0 "$app_pid" 2>/dev/null; then
    kill -TERM "$app_pid" 2>/dev/null || true
    for _ in $(seq 1 30); do
      kill -0 "$app_pid" 2>/dev/null || break
      sleep 0.1
    done
    kill -KILL "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

window_id=""
for _ in $(seq 1 45); do
  window_id="$(xwininfo -root -tree 2>/dev/null | python3 -c '
import re
import sys

windows = []
for line in sys.stdin:
    if "\"nexus-hud\"" not in line:
        continue
    match = re.search(r"(0x[0-9a-fA-F]+).*?(\d+)x(\d+)[+-]-?\d+[+-]-?\d+", line)
    if match:
        width, height = int(match.group(2)), int(match.group(3))
        if width >= 80 and height >= 48:
            windows.append((width * height, match.group(1)))
if windows:
    print(max(windows)[1])
')"
  if [[ -n "$window_id" ]]; then break; fi
  sleep 1
done
if [[ -z "$window_id" ]]; then
  echo "FAIL: current-build Wails HUD window did not open" >&2
  cat "$scratch/hud.log" >&2
  exit 1
fi

sleep 4
xwininfo -root -tree >"$evidence_dir/x11-window-tree.txt"
xwininfo -id "$window_id" >"$evidence_dir/x11-window-geometry.txt"
xwd -root -silent -out "$scratch/hud-first-paint.xwd"
convert "$scratch/hud-first-paint.xwd" "$evidence_dir/hud-first-paint-root.png"

window_geometry="$(xwininfo -id "$window_id" | awk '
  /Absolute upper-left X:/ { x=$4 }
  /Absolute upper-left Y:/ { y=$4 }
  /Width:/ { w=$2 }
  /Height:/ { h=$2 }
  END { if (x != "" && y != "" && w != "" && h != "") print x, y, w, h }
')"
read -r window_x window_y window_width window_height <<<"$window_geometry"
if [[ -z "${window_width:-}" || "$window_width" -lt 100 || "$window_height" -lt 80 ]]; then
  echo "FAIL: HUD window geometry is invalid: $window_geometry" >&2
  exit 1
fi
convert "$evidence_dir/hud-first-paint-root.png" \
  -crop "${window_width}x${window_height}+${window_x}+${window_y}" +repage \
  "$evidence_dir/hud-mascot.png"

python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" \
  click "$((window_x + window_width / 2))" "$((window_y + window_height / 2))"
sleep "${NEXUS_HUD_POPUP_CAPTURE_WAIT_SECONDS:-2}"
xwd -root -silent -out "$scratch/hud-expanded.xwd"
convert "$scratch/hud-expanded.xwd" "$evidence_dir/hud-expanded-root.png"

echo "hud_binary=$hud_bin"
echo "hud_sha256=$(sha256sum "$hud_bin" | awk '{print $1}')"
echo "isolated_home=$scratch/home"
echo "window_geometry=$window_x,$window_y ${window_width}x${window_height}"
echo "initial_hud_capture=$evidence_dir/hud-mascot.png"
echo "expanded_hud_capture=$evidence_dir/hud-expanded-root.png"
sha256sum "$evidence_dir/hud-mascot.png" "$evidence_dir/hud-expanded-root.png"
echo "embedded_core_log=$scratch/hud.log"
cat "$scratch/hud.log"
