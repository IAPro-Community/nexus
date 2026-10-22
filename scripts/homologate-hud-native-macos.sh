#!/usr/bin/env bash
set -euo pipefail
# Native macOS HUD acceptance. Run from an unlocked graphical user session.
# Requires accessibility access for the terminal that executes this script.
if [ "$#" -lt 2 ]; then echo "Usage: bash $0 HUD_BINARY EVIDENCE_DIRECTORY [CYCLES]" >&2; exit 2; fi
HUD="$1"
OUT="$2"
CYCLES=20
if [ "$#" -ge 3 ]; then CYCLES="$3"; fi
if ! [[ "$CYCLES" =~ ^[0-9]+$ ]] || (( CYCLES < 1 || CYCLES > 50 )); then
  echo "CYCLES must be between 1 and 50" >&2
  exit 2
fi
mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SHA="$(git -C "$ROOT" rev-parse HEAD)"
RESULT=INCONCLUSIVE
REASON=''
DONE=0
PRESENTATION=legacy_single_window
PID=''
report() {
  HUD_SHA="$SHA" HUD_RESULT="$RESULT" HUD_REASON="$REASON" HUD_DONE="$DONE" HUD_CYCLES="$CYCLES" HUD_PRESENTATION="$PRESENTATION" HUD_BINARY_SHA="$(shasum -a 256 "$HUD" 2>/dev/null | awk '{print $1}' || true)" \
  python3 - "$OUT/HUD_NATIVE_RESULTS.json" <<'PY'
import json,os,sys
data={"sha":os.getenv("HUD_SHA"),"platform":"darwin","presentation":os.getenv("HUD_PRESENTATION"),"type":"native_interactive_mouse",
"cycles_requested":int(os.getenv("HUD_CYCLES")),"cycles_completed":int(os.getenv("HUD_DONE")),
"binary_sha256":os.getenv("HUD_BINARY_SHA"),"verdict":os.getenv("HUD_RESULT"),"reason":os.getenv("HUD_REASON"),
"visual_shadow_review":"NOT_TESTED","transparent_click_through":"NOT_TESTED",
"note":"Real CoreGraphics mouse and native geometry. Inspect screenshots for window shadow."}
with open(sys.argv[1],"w") as f: json.dump(data,f,indent=2)
PY
}
cleanup() {
  if [ -n "$PID" ]; then
    kill -TERM "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
  report
}
trap cleanup EXIT
if [ "$(uname -s)" != "Darwin" ]; then RESULT=BLOCKED; REASON='Requires macOS'; exit 1; fi
for bin in swiftc swift screencapture python3 git shasum; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    RESULT=BLOCKED; REASON="Missing $bin"; echo "$REASON" >&2; exit 1
  fi
done
if [ ! -x "$HUD" ]; then RESULT=BLOCKED; REASON='HUD executable missing'; exit 1; fi
if pgrep -x 'nexus-hud' >/dev/null 2>&1; then
  RESULT=BLOCKED; REASON='Another Nexus HUD is running; use an isolated session.'
  echo "$REASON" >&2; exit 1
fi
# macOS native APIs: query the window owned by the launch PID and send clicks.
# Do not fetch or install any automation dependency from the network.
cat >"$OUT/hud_native_probe.swift" <<'SWIFT'
import Foundation
import AppKit
import ApplicationServices
import CoreGraphics

func bounds(pid: Int32, role: String) -> String? {
    guard let windows = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID)
            as? [[String: Any]] else { return nil }
    // The always-on-top HUD uses a floating CoreGraphics layer, not necessarily zero.
    let candidates = windows.filter {
        ($0[kCGWindowOwnerPID as String] as? NSNumber)?.int32Value == pid
    }
    let valid = candidates.compactMap { window -> (name: String, rect: [String: Any], area: Int)? in
        guard let raw = window[kCGWindowBounds as String] as? [String: Any],
              let w = raw["Width"] as? NSNumber, let h = raw["Height"] as? NSNumber,
              w.intValue >= 80, h.intValue >= 48 else { return nil }
        return (window[kCGWindowName as String] as? String ?? "", raw, w.intValue * h.intValue)
    }
    let selected: (name: String, rect: [String: Any], area: Int)?
    if role == "popup" {
        selected = valid.first(where: { $0.name == "Nexus HUD Popup" }) ??
          (valid.count > 1 ? valid.filter({ $0.area >= 280 * 240 }).max(by: { $0.area < $1.area }) : nil)
    } else {
        selected = valid.first(where: { $0.name == "Nexus HUD" }) ??
          valid.min(by: { $0.area < $1.area })
    }
    guard let raw = selected?.rect,
          let x = raw["X"] as? NSNumber, let y = raw["Y"] as? NSNumber,
          let w = raw["Width"] as? NSNumber, let h = raw["Height"] as? NSNumber else { return nil }
    return "\(x.intValue),\(y.intValue),\(w.intValue),\(h.intValue)"
}

let args = CommandLine.arguments
guard args.count >= 3 else { exit(2) }
if args[1] == "bounds" {
    guard args.count >= 4, let pid = Int32(args[2]),
          let result = bounds(pid: pid, role: args[3]) else { exit(3) }
    print(result)
} else if args[1] == "click" {
    guard args.count >= 4, AXIsProcessTrusted(),
          let x = Double(args[2]), let y = Double(args[3]) else { exit(4) }
    let point = CGPoint(x: x, y: y)
    let events: [CGEventType] = [.mouseMoved, .leftMouseDown, .leftMouseUp]
    for kind in events {
        CGEvent(mouseEventSource: nil, mouseType: kind,
                mouseCursorPosition: point, mouseButton: .left)?.post(tap: .cghidEventTap)
        usleep(60_000)
    }
} else if args[1] == "accessibility" {
    exit(AXIsProcessTrusted() ? 0 : 4)
} else {
    exit(2)
}
SWIFT
if ! swiftc "$OUT/hud_native_probe.swift" -o "$OUT/hud_native_probe" 2>"$OUT/private-build.log"; then
  RESULT=BLOCKED; REASON='Could not compile the CoreGraphics probe; see private-build.log.'
  echo "$REASON" >&2; exit 1
fi
if ! "$OUT/hud_native_probe" accessibility; then
  RESULT=BLOCKED; REASON='Accessibility permission required for local terminal; do not mark this run PASS.'
  echo "$REASON" >&2; exit 1
fi
export NEXUS_DATA_DIR="$OUT/isolated-data"
export NEXUS_CONFIG_DIR="$OUT/isolated-config"
export NEXUS_STATE_DIR="$OUT/isolated-state"
unset NEXUS_ENABLE_HUD NEXUS_DISABLE_HUD
mkdir -p "$NEXUS_DATA_DIR" "$NEXUS_CONFIG_DIR" "$NEXUS_STATE_DIR"
"$HUD" >"$OUT/private-stdout.log" 2>"$OUT/private-stderr.log" &
PID=$!
read_geometry() {
  local b
  b="$("$OUT/hud_native_probe" bounds "$PID" capsule 2>/dev/null)" || return 1
  IFS=',' read -r X Y W H <<< "$b"
}
wait_geometry() {
  local expanded="$1"
  for i in $(seq 1 40); do
    if read_geometry; then
      if [ "$expanded" = yes ] && (( W >= 300 && H >= 300 )); then return 0; fi
      if [ "$expanded" = no ] && (( W <= 330 && H <= 140 )); then return 0; fi
    fi
    sleep .15
  done
  return 1
}
click_capsule() {
  local x=$(( X+W*68/100 )) y=$(( Y+H-27 ))
  "$OUT/hud_native_probe" click "$x" "$y"
}
capture() {
  local x=$((X-24)) y=$((Y-24)) w=$((W+48)) h=$((H+48))
  if (( x < 0 )); then w=$((w+x)); x=0; fi
  if (( y < 0 )); then h=$((h+y)); y=0; fi
  screencapture -x -R "$x,$y,$w,$h" "$OUT/$1"
}
for i in $(seq 1 100); do
  if ! kill -0 "$PID" 2>/dev/null; then RESULT=NO_GO; REASON='HUD exited before native window appeared.'; exit 1; fi
  if read_geometry; then break; fi
  sleep .2
done
if ! read_geometry; then RESULT=BLOCKED; REASON='Cannot inspect native HUD window; check Screen Recording/session access.'; exit 1; fi
read_popup_geometry() {
  local b
  b="$("$OUT/hud_native_probe" bounds "$PID" popup 2>/dev/null)" || return 1
  IFS=',' read -r PX PY PW PH <<< "$b"
}
wait_popup() {
  local shown="$1"
  for i in $(seq 1 40); do
    if read_popup_geometry; then
      if [ "$shown" = yes ] && (( PW >= 280 && PH >= 240 )); then return 0; fi
    elif [ "$shown" = no ]; then
      return 0
    fi
    sleep .15
  done
  return 1
}
capture_popup() {
  read_popup_geometry
  local old_x="$X" old_y="$Y" old_w="$W" old_h="$H"
  X="$PX"; Y="$PY"; W="$PW"; H="$PH"
  capture "$1"
  X="$old_x"; Y="$old_y"; W="$old_w"; H="$old_h"
}
if ! wait_geometry no; then RESULT=NO_GO; REASON='Native HUD compact geometry invalid.'; exit 1; fi
if ! capture HUD_BEFORE.png; then RESULT=BLOCKED; REASON='Screen capture denied or unavailable.'; exit 1; fi
for n in $(seq 1 "$CYCLES"); do
  read_geometry
  if ! click_capsule; then RESULT=BLOCKED; REASON='Accessibility click was not authorized.'; exit 1; fi
  if (( n == 1 )); then
    for detect in $(seq 1 30); do
      if read_popup_geometry; then
        PRESENTATION=detached_popover
        break
      fi
      read_geometry || true
      if (( W >= 300 && H >= 300 )); then break; fi
      sleep .15
    done
  fi
  if [ "$PRESENTATION" = detached_popover ]; then
    if ! wait_geometry no; then RESULT=NO_GO; REASON="Capsule resized unexpectedly on cycle $n"; exit 1; fi
    if ! wait_popup yes; then RESULT=NO_GO; REASON="Detached popup failed to open on cycle $n"; exit 1; fi
    if (( n == 1 )); then
      if ! capture_popup HUD_EXPANDED.png; then RESULT=BLOCKED; REASON='Screen capture unavailable'; exit 1; fi
    fi
    read_popup_geometry
    if ! "$OUT/hud_native_probe" click "$((PX+PW-36))" "$((PY+36))"; then
      RESULT=BLOCKED; REASON='Cannot click detached popup native close control'; exit 1
    fi
    if ! wait_popup no; then RESULT=NO_GO; REASON="Detached popup failed to close on cycle $n"; exit 1; fi
    if ! wait_geometry no; then RESULT=NO_GO; REASON="Compact capsule lost on cycle $n"; exit 1; fi
  else
    if ! wait_geometry yes; then RESULT=NO_GO; REASON="Legacy HUD failed to expand on cycle $n"; exit 1; fi
    if (( n == 1 )); then
      if ! capture HUD_EXPANDED.png; then RESULT=BLOCKED; REASON='Screen capture unavailable'; exit 1; fi
    fi
    click_capsule
    if ! wait_geometry no; then RESULT=NO_GO; REASON="Legacy HUD failed to collapse on cycle $n"; exit 1; fi
  fi
  DONE="$n"
done
RESULT=PASS_FUNCTIONAL_PENDING_VISUAL
echo "macOS $PRESENTATION native mouse cycles: $DONE/$CYCLES; review images in $OUT"
