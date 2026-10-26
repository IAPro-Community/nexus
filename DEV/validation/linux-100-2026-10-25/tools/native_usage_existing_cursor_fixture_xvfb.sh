#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
desktop_bin="${NEXUS_DESKTOP_BIN:-$repo_root/cmd/nexus-desktop/build/bin/nexus-desktop}"
cli_bin="${NEXUS_CLI_BIN:-$repo_root/nexus}"
evidence_dir="${NEXUS_USAGE_EXISTING_SCREENSHOTS:-$repo_root/DEV/validation/linux-100-2026-10-25/p5-recheck-2026-10-26T0347Z/screenshots/desktop-existing-cursor}"
runtime_root="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
scratch="$(mktemp -d "$runtime_root/nexus-native-existing-cursor.XXXXXX")"

mkdir -p "$scratch/home/repository" "$scratch/config/nexus" "$scratch/data/nexus" \
  "$scratch/cache" "$scratch/home/.codex" "$scratch/home/.claude" "$scratch/bin" "$evidence_dir"
git -C "$scratch/home/repository" init -q

cat >"$scratch/bin/cursor-agent" <<'CURSOR_FIXTURE'
#!/usr/bin/env bash
case "${1:-}" in
  --version) printf 'Cursor Agent fixture 1.0.0\n' ;;
  status) printf '✓ Login successful!\n' ;;
  *) printf 'Cursor fixture command: %s\n' "${1:-}" ;;
esac
CURSOR_FIXTURE
chmod 700 "$scratch/bin/cursor-agent"

export HOME="$scratch/home"
export XDG_CONFIG_HOME="$scratch/config"
export XDG_DATA_HOME="$scratch/data"
export XDG_CACHE_HOME="$scratch/cache"
export XDG_RUNTIME_DIR="$runtime_root"
export NEXUS_CONFIG_DIR="$scratch/config/nexus"
export NEXUS_DATA_DIR="$scratch/data/nexus"
export CODEX_HOME="$scratch/home/.codex"
export CLAUDE_CONFIG_DIR="$scratch/home/.claude"
export PATH="$scratch/bin:$PATH"
export GDK_BACKEND=x11
export LIBGL_ALWAYS_SOFTWARE=1
unset OPENAI_API_KEY ANTHROPIC_API_KEY GOOGLE_API_KEY GEMINI_API_KEY
unset CLAUDE_CODE_OAUTH_TOKEN CODEX_API_KEY OPENROUTER_API_KEY GH_TOKEN GITHUB_TOKEN

if [[ ! -x "$desktop_bin" || ! -x "$cli_bin" ]]; then
  echo "FAIL: Nexus CLI or Desktop binary is missing" >&2
  exit 1
fi

"$cli_bin" add cursor fixture --no-login >"$scratch/profile-create.log" 2>&1
"$desktop_bin" >"$scratch/desktop.log" 2>&1 &
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

for _ in $(seq 1 45); do
  if xwininfo -root -tree 2>/dev/null | rg -q 'IAPro Nexus'; then break; fi
  sleep 1
done
if ! xwininfo -root -tree 2>/dev/null | rg -q 'IAPro Nexus'; then
  echo "FAIL: Wails desktop window did not open" >&2
  cat "$scratch/profile-create.log" "$scratch/desktop.log" >&2
  exit 1
fi
sleep 8

port="$(sed -n 's#.*http://127\.0\.0\.1:\([0-9][0-9]*\).*#\1#p' "$scratch/desktop.log" | head -n 1)"
if [[ -z "$port" ]]; then
  echo "FAIL: embedded Core did not publish its health endpoint" >&2
  exit 1
fi
curl --fail --silent --show-error "http://127.0.0.1:$port/api/v1/health" >"$scratch/health.json"

python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" click 700 490
python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" type "$scratch/home/repository"
python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" click 700 535
python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" type nexus-cursor-fixture
python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" click 510 578
sleep 6

python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" click 112 792
sleep 5
xwd -root -silent -out "$scratch/usage-before-import.xwd"
convert "$scratch/usage-before-import.xwd" "$evidence_dir/usage-before-import.png"
python3 "$repo_root/DEV/validation/linux-100-2026-10-25/tools/x11_input.py" click 1038 189
sleep 10
xwd -root -silent -out "$scratch/usage-after-import.xwd"
convert "$scratch/usage-after-import.xwd" "$evidence_dir/usage-after-import.png"

echo "desktop_binary=$desktop_bin"
echo "desktop_sha256=$(sha256sum "$desktop_bin" | awk '{print $1}')"
echo "fixture_provider=cursor; fixture_profile=fixture; fake_status=Login successful"
echo "scratch_home=$scratch/home"
echo "isolated_nexus_data=$NEXUS_DATA_DIR"
echo "core_port=$port"
echo "core_health=$(cat "$scratch/health.json")"
echo "result=Review usage-after-import.png; expected 1 already-authenticated profile and no copied host credential."
sha256sum "$evidence_dir/usage-before-import.png" "$evidence_dir/usage-after-import.png"
echo "fixture_profile_log=$scratch/profile-create.log"
echo "embedded_core_log=$scratch/desktop.log"
