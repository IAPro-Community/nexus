#!/usr/bin/env bash
# Native Wails smoke for Usage / Accounts without accessing host provider sessions.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
original_home="${HOME:?HOME must be set before running this test}"
desktop_bin="${NEXUS_DESKTOP_BIN:-$original_home/.local/bin/nexus-desktop}"
test_tools="$repo_root/DEV/validation/linux-100-2026-10-25/tools"
screenshots="${NEXUS_USAGE_SCREENSHOTS:-$repo_root/DEV/validation/linux-100-2026-10-25/screenshots/native-usage-import-xvfb}"
scratch="$(mktemp -d "${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/nexus-native-usage-xvfb.XXXXXX")"
mkdir -p "$scratch/home/repository" "$scratch/home/.codex" "$scratch/home/.claude" \
  "$scratch/config" "$scratch/data" "$scratch/cache" "$screenshots"
git -C "$scratch/home/repository" init -q

export HOME="$scratch/home"
export XDG_CONFIG_HOME="$scratch/config"
export XDG_DATA_HOME="$scratch/data"
export XDG_CACHE_HOME="$scratch/cache"
export NEXUS_DATA_DIR="$scratch/data/nexus"
export CODEX_HOME="$scratch/home/.codex"
export CLAUDE_CONFIG_DIR="$scratch/home/.claude"
export GDK_BACKEND=x11
export LIBGL_ALWAYS_SOFTWARE=1

# The application only sees an empty HOME/profile and isolated provider homes.
# Drop ambient credentials too, so the safety boundary is explicit.
unset OPENAI_API_KEY ANTHROPIC_API_KEY GOOGLE_API_KEY GEMINI_API_KEY
unset CLAUDE_CODE_OAUTH_TOKEN CODEX_API_KEY OPENROUTER_API_KEY GH_TOKEN GITHUB_TOKEN

"$desktop_bin" >"$scratch/desktop.log" 2>&1 &
app_pid=$!
trap 'kill "$app_pid" 2>/dev/null || true; wait "$app_pid" 2>/dev/null || true' EXIT

for _ in $(seq 1 30); do
  if xwininfo -root -tree 2>/dev/null | rg -q 'IAPro Nexus'; then break; fi
  sleep 1
done
sleep 10
if ! xwininfo -root -tree 2>/dev/null | rg -q 'IAPro Nexus'; then
  echo "FAIL: Wails desktop window did not open" >&2
  cat "$scratch/desktop.log" >&2
  exit 1
fi

port="$(sed -n 's#.*http://127\.0\.0\.1:\([0-9][0-9]*\).*#\1#p' "$scratch/desktop.log" | head -n 1)"
if [[ -z "$port" ]]; then
  echo "FAIL: embedded Core did not publish its health endpoint" >&2
  exit 1
fi
curl --fail --silent --show-error "http://127.0.0.1:$port/api/v1/health" >"$scratch/health.json"

# Complete first-run setup using a synthetic, empty Git repository in the isolated HOME.
python3 "$test_tools/x11_input.py" click 700 490
python3 "$test_tools/x11_input.py" type "$scratch/home/repository"
python3 "$test_tools/x11_input.py" click 700 535
python3 "$test_tools/x11_input.py" type nexus-homologation
python3 "$test_tools/x11_input.py" click 510 578
sleep 6

# Open the native Usage surface, capture it, and click the real import button once.
python3 "$test_tools/x11_input.py" click 112 792
sleep 4
xwd -root -silent -out "$scratch/usage-before-import.xwd"
convert "$scratch/usage-before-import.xwd" "$screenshots/usage-before-import.png"
python3 "$test_tools/x11_input.py" click 1038 189
sleep 8
xwd -root -silent -out "$scratch/usage-after-import.xwd"
convert "$scratch/usage-after-import.xwd" "$screenshots/usage-after-import.png"

echo "desktop_binary=$desktop_bin"
echo "desktop_sha256=$(sha256sum "$desktop_bin" | awk '{print $1}')"
echo "scratch_home=$scratch/home"
echo "isolated_nexus_data=$NEXUS_DATA_DIR"
echo "core_port=$port"
echo "core_health=$(cat "$scratch/health.json")"
echo "result=Review usage-after-import.png; HOME and data are isolated, so host sessions cannot be copied."
sha256sum "$screenshots/usage-before-import.png" "$screenshots/usage-after-import.png"
