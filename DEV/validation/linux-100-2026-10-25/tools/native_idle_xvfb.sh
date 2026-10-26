#!/usr/bin/env bash
# Measure the installed Wails Desktop's WebKit CPU/RSS after an isolated first-run setup.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
original_home="${HOME:?HOME must be set before running this test}"
desktop_bin="${NEXUS_DESKTOP_BIN:-$original_home/.local/bin/nexus-desktop}"
test_tools="$repo_root/DEV/validation/linux-100-2026-10-25/tools"
screenshots="${NEXUS_IDLE_SCREENSHOTS:-$repo_root/DEV/validation/linux-100-2026-10-25/screenshots/native-idle-xvfb}"
scratch="$(mktemp -d "${XDG_RUNTIME_DIR:-/tmp}/nexus-native-idle-xvfb.XXXXXX")"
mkdir -p "$scratch/home/repository" "$scratch/home/.codex" "$scratch/home/.claude" \
  "$scratch/config" "$scratch/data" "$scratch/cache" "$scratch/runtime" "$screenshots"
chmod 700 "$scratch/runtime"
git -C "$scratch/home/repository" init -q

export HOME="$scratch/home"
export XDG_CONFIG_HOME="$scratch/config"
export XDG_DATA_HOME="$scratch/data"
export XDG_CACHE_HOME="$scratch/cache"
export XDG_RUNTIME_DIR="$scratch/runtime"
export NEXUS_DATA_DIR="$scratch/data/nexus"
export CODEX_HOME="$scratch/home/.codex"
export CLAUDE_CONFIG_DIR="$scratch/home/.claude"
export GDK_BACKEND=x11
export LIBGL_ALWAYS_SOFTWARE=1
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

# Complete first-run setup with a synthetic empty repository, then open Usage/Accounts
# through the keyboard command palette instead of relying on sidebar coordinates.
python3 "$test_tools/x11_input.py" click 700 441
python3 "$test_tools/x11_input.py" type "$scratch/home/repository"
python3 "$test_tools/x11_input.py" click 700 485
python3 "$test_tools/x11_input.py" type nexus-idle-homologation
python3 "$test_tools/x11_input.py" key ENTER
sleep 8
xwd -root -silent -out "$scratch/project-created.xwd"
convert "$scratch/project-created.xwd" "$screenshots/project-created.png"
python3 "$test_tools/x11_input.py" ctrl k
sleep 2
python3 "$test_tools/x11_input.py" type accounts
python3 "$test_tools/x11_input.py" key ENTER
sleep 10
xwd -root -silent -out "$scratch/usage-idle.xwd"
convert "$scratch/usage-idle.xwd" "$screenshots/usage-accounts-idle.png"
python3 "$test_tools/x11_input.py" move 10 10
sleep 30

python3 - "$app_pid" <<'PY'
import pathlib
import sys
import time

root_pid = int(sys.argv[1])
proc = pathlib.Path('/proc')

def process_info(pid):
    try:
        raw = (proc / str(pid) / 'stat').read_text()
        tail = raw[raw.rfind(')') + 2:].split()
        parent = int(tail[1])
        ticks = int(tail[11]) + int(tail[12])
        comm = (proc / str(pid) / 'comm').read_text().strip()
        status = (proc / str(pid) / 'status').read_text()
        rss_kb = next(int(line.split()[1]) for line in status.splitlines() if line.startswith('VmRSS:'))
        return parent, ticks, comm, rss_kb
    except (FileNotFoundError, ProcessLookupError, StopIteration, IndexError, ValueError):
        return None

def descendants(parent_pid):
    found = []
    for item in proc.iterdir():
        if not item.name.isdigit():
            continue
        info = process_info(int(item.name))
        if info and info[0] == parent_pid:
            found.append((int(item.name), info))
            found.extend(descendants(int(item.name)))
    return found

def webkit_processes():
    return [(pid, info) for pid, info in descendants(root_pid) if info[2].startswith('WebKitWebProces')]

before = webkit_processes()
if not before:
    raise SystemExit('FAIL: no WebKitWebProcess child found under the Wails Desktop')
time.sleep(20)
after = webkit_processes()
hz = int(__import__('os').sysconf('SC_CLK_TCK'))
print(f'root_desktop_pid={root_pid}')
print('idle_surface=Usage/Accounts after isolated first-run setup; pointer outside app')
print(f'webkit_processes_start={len(before)}')
print(f'webkit_processes_end={len(after)}')
print('webkit_idle_sample_seconds=20')
for pid, info in after:
    prior = next((old for old_pid, old in before if old_pid == pid), None)
    if prior is None:
        print(f'pid={pid} comm={info[2]} cpu_percent=NEW rss_kib={info[3]}')
        continue
    cpu_seconds = (info[1] - prior[1]) / hz
    print(f'pid={pid} comm={info[2]} cpu_seconds={cpu_seconds:.2f} cpu_percent={cpu_seconds/20*100:.1f} rss_kib={info[3]}')
PY

echo "desktop_binary=$desktop_bin"
echo "desktop_sha256=$(sha256sum "$desktop_bin" | awk '{print $1}')"
echo "isolated_home=$scratch/home"
echo "core_health=$(cat "$scratch/health.json")"
sha256sum "$screenshots/usage-accounts-idle.png"
echo "no host provider sessions or credentials were exposed to this process"
