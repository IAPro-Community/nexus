#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
hud_bin="${NEXUS_HUD_BIN:-$repo_root/cmd/nexus-hud/build/bin/nexus-hud}"
max_cpu="${NEXUS_HUD_CPU_MAX_PERCENT:-15}"
sample_seconds=20
scratch="$(mktemp -d "${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/nexus-hud-idle-cpu.XXXXXX")"
mkdir -p "$scratch/home" "$scratch/config" "$scratch/data" "$scratch/cache" "$scratch/runtime"
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
unset CLAUDE_CODE_OAUTH_TOKEN CODEX_API_KEY OPENROUTER_API_KEY GH_TOKEN GITHUB_TOKEN CURSOR_API_KEY

if [[ ! -x "$hud_bin" ]]; then
  echo "FAIL: HUD binary is missing or not executable: $hud_bin" >&2
  exit 1
fi
if ! [[ "$max_cpu" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
  echo "FAIL: NEXUS_HUD_CPU_MAX_PERCENT must be numeric" >&2
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

python3 - "$app_pid" "$hud_bin" "$max_cpu" "$sample_seconds" "$scratch/hud.log" <<'PY'
import os
import pathlib
import re
import subprocess
import sys
import time

root_pid = int(sys.argv[1])
hud_bin = pathlib.Path(sys.argv[2])
max_cpu = float(sys.argv[3])
sample_seconds = int(sys.argv[4])
hud_log = pathlib.Path(sys.argv[5])
proc = pathlib.Path('/proc')

def process_info(pid):
    try:
        raw = (proc / str(pid) / 'stat').read_text()
        fields = raw[raw.rfind(')') + 2:].split()
        status = (proc / str(pid) / 'status').read_text()
        rss_kib = int(next(line.split()[1] for line in status.splitlines() if line.startswith('VmRSS:')))
        return int(fields[1]), int(fields[11]) + int(fields[12]), (proc / str(pid) / 'comm').read_text().strip(), rss_kib
    except (OSError, ValueError, StopIteration, IndexError):
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

def visible_hud_window():
    try:
        result = subprocess.run(
            ['xwininfo', '-root', '-tree'], check=True, capture_output=True, text=True,
        )
    except (OSError, subprocess.CalledProcessError):
        return None
    for line in result.stdout.splitlines():
        if '"nexus-hud"' not in line:
            continue
        match = re.search(r'\s(\d+)x(\d+)[+-]-?\d+[+-]-?\d+', line)
        if match and int(match.group(1)) >= 100 and int(match.group(2)) >= 80:
            return line.strip()
    return None

window = None
for _ in range(90):
    window = visible_hud_window()
    if window and webkit_processes():
        break
    time.sleep(1)
if not window or not webkit_processes():
    print('FAIL: native HUD window or WebKit process did not become ready')
    print(hud_log.read_text(errors='replace')[-5000:])
    raise SystemExit(1)

# Let startup and the one-shot entrance animation settle before measuring.
time.sleep(3)
before = webkit_processes()
time.sleep(sample_seconds)
after = webkit_processes()
hz = os.sysconf('SC_CLK_TCK')
print(f'hud_window={window}')
print(f'hud_idle_sample_seconds={sample_seconds}')
print(f'cpu_limit_percent_per_webkit_process={max_cpu:.1f}')
print(f'webkit_processes_start={len(before)} webkit_processes_end={len(after)}')
failures = []
for pid, info in after:
    previous = next((entry for old_pid, entry in before if old_pid == pid), None)
    if previous is None:
        print(f'pid={pid} comm={info[2]} cpu_percent=NEW rss_kib={info[3]}')
        failures.append(pid)
        continue
    cpu_seconds = (info[1] - previous[1]) / hz
    cpu_percent = cpu_seconds / sample_seconds * 100
    print(f'pid={pid} comm={info[2]} cpu_seconds={cpu_seconds:.2f} cpu_percent={cpu_percent:.1f} rss_kib={info[3]}')
    if cpu_percent > max_cpu:
        failures.append(pid)
if len(before) != len(after):
    print('FAIL: WebKit process set changed during the idle sample')
    raise SystemExit(1)
if failures:
    print(f'FAIL: idle WebKit CPU exceeded {max_cpu:.1f}% for PIDs {failures}')
    raise SystemExit(1)
print(f'PASS: all idle WebKit processes stayed at or below {max_cpu:.1f}%')
print(f'hud_binary={hud_bin}')
subprocess.run(['sha256sum', str(hud_bin)], check=True)
PY
