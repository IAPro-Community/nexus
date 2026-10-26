#!/usr/bin/env bash
set -euo pipefail

if [[ "${XDG_SESSION_TYPE:-}" != "wayland" ]]; then
  echo "BLOCKED: run this from the user's GNOME Wayland session (XDG_SESSION_TYPE=wayland)." >&2
  exit 2
fi
for tool in pgrep busctl dbus-send curl ss sha256sum; do
  command -v "$tool" >/dev/null || { echo "BLOCKED: missing required tool: $tool" >&2; exit 2; }
done

mapfile -t desktop_pids < <(pgrep -x nexus-desktop || true)
if [[ "${#desktop_pids[@]}" -ne 1 ]]; then
  echo "BLOCKED: expected exactly one running nexus-desktop; found ${#desktop_pids[@]}." >&2
  exit 2
fi
desktop_pid="${desktop_pids[0]}"
mapfile -t hud_pids < <(pgrep -P "$desktop_pid" -x nexus-hud || true)
if [[ "${#hud_pids[@]}" -ne 1 ]]; then
  echo "BLOCKED: expected exactly one HUD linked to Desktop PID $desktop_pid; found ${#hud_pids[@]}." >&2
  exit 2
fi
hud_pid="${hud_pids[0]}"
item_service="org.kde.StatusNotifierItem-${hud_pid}-1"
state_file="${XDG_DATA_HOME:-$HOME/.local/share}/ai-manager/hud-window-state.json"
if [[ ! -f "$state_file" ]]; then
  echo "BLOCKED: HUD state file not found: $state_file" >&2
  exit 2
fi
if ! busctl --user status "$item_service" >/dev/null 2>&1; then
  echo "BLOCKED: HUD StatusNotifierItem is not registered on the user bus: $item_service" >&2
  exit 2
fi

health_port="$(ss -H -ltnp | sed -n "/pid=${desktop_pid},/s/.*127\\.0\\.0\\.1:\\([0-9][0-9]*\\).*/\\1/p" | head -n 1)"
if [[ -z "$health_port" ]]; then
  echo "BLOCKED: could not identify the embedded Core loopback port for PID $desktop_pid." >&2
  exit 2
fi

event() {
  dbus-send --session --print-reply=literal --dest="$item_service" \
    /StatusNotifierMenu com.canonical.dbusmenu.Event \
    "int32:$1" string:clicked variant:uint32:0 uint32:0
}

restore_visible() {
  event 1 >/dev/null 2>&1 || true
}
trap restore_visible EXIT

check_state() {
  local label="$1"
  local actual
  actual="$(sha256sum "$state_file" | awk '{print $1}')"
  printf '%s_sha256=%s\n' "$label" "$actual"
  if [[ "$actual" != "$baseline_hash" ]]; then
    printf '%s=FAIL\n' "$label"
    return 1
  fi
  printf '%s=PASS\n' "$label"
}

printf 'desktop_pid=%s\nhud_pid=%s\nstatus_notifier_item=%s\ncore_port=%s\n' \
  "$desktop_pid" "$hud_pid" "$item_service" "$health_port"
printf 'core_health_before='
curl -fsS "http://127.0.0.1:${health_port}/api/v1/health"
printf '\n'
baseline_hash="$(sha256sum "$state_file" | awk '{print $1}')"
printf 'state_before_sha256=%s\n' "$baseline_hash"
cat "$state_file"
printf 'hide_event='
event 2
sleep 2
check_state after_hide
printf 'show_event='
event 1
sleep 2
check_state after_show
sleep 10
check_state after_settle_10s
kill -0 "$hud_pid"
printf 'hud_alive=PASS\n'
printf 'core_health_after='
curl -fsS "http://127.0.0.1:${health_port}/api/v1/health"
printf '\nresult=PASS\n'
