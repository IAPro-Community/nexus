#!/usr/bin/env bash
set -euo pipefail
# Native Linux X11 HUD acceptance; Wayland must not be certified by Xvfb.
if [ "$#" -lt 2 ]; then echo "Usage: $0 HUD_BINARY EVIDENCE_DIRECTORY [CYCLES]" >&2; exit 2; fi
HUD="$(realpath "$1")"
OUT="$(realpath -m "$2")"
CYCLES=20
if [ "$#" -ge 3 ]; then CYCLES="$3"; fi
if ! [[ "$CYCLES" =~ ^[0-9]+$ ]] || (( CYCLES < 1 || CYCLES > 50 )); then
  echo "CYCLES must be between 1 and 50" >&2
  exit 2
fi
mkdir -p "$OUT"
SHA="$(git -C "$(cd "$(dirname "$0")/.." && pwd)" rev-parse HEAD)"
RESULT=INCONCLUSIVE
REASON=''
DONE=0
PRESENTATION=legacy_single_window
PID=''
POPUP=''
report() {
  HUD_SHA="$SHA" HUD_RESULT="$RESULT" HUD_REASON="$REASON" HUD_DONE="$DONE" HUD_CYCLES="$CYCLES" HUD_PRESENTATION="$PRESENTATION" HUD_BINARY_SHA="$(sha256sum "$HUD" 2>/dev/null | awk '{print $1}' || true)" \
  python3 - "$OUT/HUD_NATIVE_RESULTS.json" <<'PY'
import json,os,sys
data={"sha":os.getenv("HUD_SHA"),"platform":"linux-x11","presentation":os.getenv("HUD_PRESENTATION"),"cycles_requested":int(os.getenv("HUD_CYCLES")),
"cycles_completed":int(os.getenv("HUD_DONE")),"binary_sha256":os.getenv("HUD_BINARY_SHA"),"verdict":os.getenv("HUD_RESULT"),"reason":os.getenv("HUD_REASON"),
"visual_shadow_review":"NOT_TESTED","transparent_click_through":"NOT_TESTED",
"note":"True X11 mouse + geometry; compare PNGs for frame/shadow manually"}
with open(sys.argv[1],"w") as f: json.dump(data,f,indent=2)
PY
}
cleanup() {
  if [ -n "$PID" ]; then kill -TERM "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; fi
  report
}
trap cleanup EXIT
SESSION="$(printenv XDG_SESSION_TYPE || true)"
if [ "$SESSION" != x11 ] || [ -z "$(printenv DISPLAY || true)" ]; then
  RESULT=BLOCKED; REASON='Only real X11 desktop supported; Wayland needs compositor-specific testing.'
  echo "$REASON" >&2; exit 1
fi
for bin in xdotool import python3 git realpath pgrep sha256sum; do
  if ! command -v "$bin" >/dev/null; then
    RESULT=BLOCKED; REASON="Missing $bin"; echo "$REASON" >&2; exit 1
  fi
done
if pgrep -x nexus-hud >/dev/null 2>&1; then
  RESULT=BLOCKED; REASON='Existing Nexus HUD process; refusing to interfere.'
  echo "$REASON" >&2; exit 1
fi
if [ ! -x "$HUD" ]; then RESULT=BLOCKED; REASON='HUD binary not executable.'; exit 1; fi
export NEXUS_DATA_DIR="$OUT/isolated-data"
export NEXUS_CONFIG_DIR="$OUT/isolated-config"
export NEXUS_STATE_DIR="$OUT/isolated-state"
unset NEXUS_ENABLE_HUD NEXUS_DISABLE_HUD
mkdir -p "$NEXUS_DATA_DIR" "$NEXUS_CONFIG_DIR" "$NEXUS_STATE_DIR"
"$HUD" >"$OUT/private-stdout.log" 2>"$OUT/private-stderr.log" &
PID=$!
WID=''
for i in $(seq 1 100); do
  if ! kill -0 "$PID" 2>/dev/null; then RESULT=NO_GO; REASON='HUD exited at launch.'; exit 1; fi
  WID="$(xdotool search --onlyvisible --pid "$PID" --name '^Nexus HUD 2>/dev/null | head -n 1 || true)"
  [ -n "$WID" ] && break
  sleep .2
done
if [ -z "$WID" ]; then RESULT=BLOCKED; REASON='No X11 window attached to HUD PID.'; exit 1; fi
geometry() {
  local g
  g="$(xdotool getwindowgeometry --shell "$WID")"
  X="$(awk -F= '/^X=/{print $2}' <<<"$g")"
  Y="$(awk -F= '/^Y=/{print $2}' <<<"$g")"
  W="$(awk -F= '/^WIDTH=/{print $2}' <<<"$g")"
  H="$(awk -F= '/^HEIGHT=/{print $2}' <<<"$g")"
}
expect_size() {
  local expanded="$1"
  for i in $(seq 1 35); do
    geometry
    if [ "$expanded" = yes ] && (( W >= 300 && H >= 300 )); then return 0; fi
    if [ "$expanded" = no ] && (( W <= 330 && H <= 140 )); then return 0; fi
    sleep .15
  done
  return 1
}
capture() {
  local x=$((X-24)) y=$((Y-24)) w=$((W+48)) h=$((H+48))
  if (( x < 0 )); then w=$((w+x)); x=0; fi
  if (( y < 0 )); then h=$((h+y)); y=0; fi
  import -silent -window root -crop "$w"x"$h"+"$x"+"$y" "$OUT/$1"
}
click_capsule() {
  local cx=$(( X+W*68/100 )) cy=$(( Y+H-27 ))
  xdotool mousemove --sync "$cx" "$cy" click 1
}
popup_window() {
  xdotool search --onlyvisible --pid "$PID" --name '^Nexus HUD Popup
 2>/dev/null | head -n 1 || true)"
  [ -n "$WID" ] && break
  sleep .2
done
if [ -z "$WID" ]; then RESULT=BLOCKED; REASON='No X11 window attached to HUD PID.'; exit 1; fi
geometry() {
  local g
  g="$(xdotool getwindowgeometry --shell "$WID")"
  X="$(awk -F= '/^X=/{print $2}' <<<"$g")"
  Y="$(awk -F= '/^Y=/{print $2}' <<<"$g")"
  W="$(awk -F= '/^WIDTH=/{print $2}' <<<"$g")"
  H="$(awk -F= '/^HEIGHT=/{print $2}' <<<"$g")"
}
expect_size() {
  local expanded="$1"
  for i in $(seq 1 35); do
    geometry
    if [ "$expanded" = yes ] && (( W >= 300 && H >= 300 )); then return 0; fi
    if [ "$expanded" = no ] && (( W <= 330 && H <= 100 )); then return 0; fi
    sleep .15
  done
  return 1
}
capture() {
  local x=$((X-24)) y=$((Y-24)) w=$((W+48)) h=$((H+48))
  if (( x < 0 )); then w=$((w+x)); x=0; fi
  if (( y < 0 )); then h=$((h+y)); y=0; fi
  import -silent -window root -crop "$w"x"$h"+"$x"+"$y" "$OUT/$1"
}
click_capsule() {
  local cx=$(( X+W*68/100 )) cy=$(( Y+H-27 ))
  xdotool mousemove --sync "$cx" "$cy" click 1
}
if ! expect_size no; then RESULT=NO_GO; REASON='Compact HUD geometry invalid.'; exit 1; fi
capture HUD_BEFORE.png
for n in $(seq 1 "$CYCLES"); do
  geometry; click_capsule
  if ! expect_size yes; then RESULT=NO_GO; REASON="Popup failed to expand on cycle $n"; exit 1; fi
  if (( n == 1 )); then capture HUD_EXPANDED.png; fi
  click_capsule
  if ! expect_size no; then RESULT=NO_GO; REASON="Popup failed to collapse on cycle $n"; exit 1; fi
  DONE="$n"
done
RESULT=PASS_FUNCTIONAL_PENDING_VISUAL
echo "X11 mouse popup cycles: $DONE/$CYCLES; images in $OUT"
 2>/dev/null | head -n 1 || true
}
popup_geometry() {
  local g
  g="$(xdotool getwindowgeometry --shell "$POPUP")" || return 1
  PX="$(awk -F= '/^X=/{print $2}' <<<"$g")"
  PY="$(awk -F= '/^Y=/{print $2}' <<<"$g")"
  PW="$(awk -F= '/^WIDTH=/{print $2}' <<<"$g")"
  PH="$(awk -F= '/^HEIGHT=/{print $2}' <<<"$g")"
}
wait_popup() {
  local shown="$1"
  for i in $(seq 1 35); do
    POPUP="$(popup_window)"
    if [ "$shown" = no ] && [ -z "$POPUP" ]; then return 0; fi
    if [ "$shown" = yes ] && [ -n "$POPUP" ]; then
      popup_geometry
      if (( PW >= 280 && PH >= 240 )); then return 0; fi
    fi
    sleep .12
  done
  return 1
}
capture_popup() {
  popup_geometry
  local old_x="$X" old_y="$Y" old_w="$W" old_h="$H"
  X="$PX"; Y="$PY"; W="$PW"; H="$PH"
  capture "$1"
  X="$old_x"; Y="$old_y"; W="$old_w"; H="$old_h"
}
if ! expect_size no; then RESULT=NO_GO; REASON='Compact HUD geometry invalid.'; exit 1; fi
capture HUD_BEFORE.png
for n in $(seq 1 "$CYCLES"); do
  geometry; click_capsule
  if (( n == 1 )); then
    # Current installs default to Guardian + detached detail window; older
    # profiles may still expand the same native surface.
    for detect in $(seq 1 30); do
      POPUP="$(popup_window)"
      if [ -n "$POPUP" ]; then
        PRESENTATION=detached_popover
        break
      fi
      geometry
      if (( W >= 300 && H >= 300 )); then break; fi
      sleep .12
    done
  fi
  if [ "$PRESENTATION" = detached_popover ]; then
    if ! expect_size no; then RESULT=NO_GO; REASON="Mascot/bar capsule resized unexpectedly on cycle $n"; exit 1; fi
    if ! wait_popup yes; then RESULT=NO_GO; REASON="Detached popup failed to open on cycle $n"; exit 1; fi
    if (( n == 1 )); then capture_popup HUD_EXPANDED.png; fi
    popup_geometry
    xdotool mousemove --sync "$((PX+PW-36))" "$((PY+36))" click 1
    if ! wait_popup no; then RESULT=NO_GO; REASON="Detached popup failed to close on cycle $n"; exit 1; fi
    if ! expect_size no; then RESULT=NO_GO; REASON="Compact capsule lost after popup close cycle $n"; exit 1; fi
  else
    if ! expect_size yes; then RESULT=NO_GO; REASON="Legacy popup failed to expand on cycle $n"; exit 1; fi
    if (( n == 1 )); then capture HUD_EXPANDED.png; fi
    click_capsule
    if ! expect_size no; then RESULT=NO_GO; REASON="Legacy popup failed to collapse on cycle $n"; exit 1; fi
  fi
  DONE="$n"
done
RESULT=PASS_FUNCTIONAL_PENDING_VISUAL
echo "X11 $PRESENTATION mouse cycles: $DONE/$CYCLES; screenshots in $OUT"
 2>/dev/null | head -n 1 || true)"
  [ -n "$WID" ] && break
  sleep .2
done
if [ -z "$WID" ]; then RESULT=BLOCKED; REASON='No X11 window attached to HUD PID.'; exit 1; fi
geometry() {
  local g
  g="$(xdotool getwindowgeometry --shell "$WID")"
  X="$(awk -F= '/^X=/{print $2}' <<<"$g")"
  Y="$(awk -F= '/^Y=/{print $2}' <<<"$g")"
  W="$(awk -F= '/^WIDTH=/{print $2}' <<<"$g")"
  H="$(awk -F= '/^HEIGHT=/{print $2}' <<<"$g")"
}
expect_size() {
  local expanded="$1"
  for i in $(seq 1 35); do
    geometry
    if [ "$expanded" = yes ] && (( W >= 300 && H >= 300 )); then return 0; fi
    if [ "$expanded" = no ] && (( W <= 330 && H <= 100 )); then return 0; fi
    sleep .15
  done
  return 1
}
capture() {
  local x=$((X-24)) y=$((Y-24)) w=$((W+48)) h=$((H+48))
  if (( x < 0 )); then w=$((w+x)); x=0; fi
  if (( y < 0 )); then h=$((h+y)); y=0; fi
  import -silent -window root -crop "$w"x"$h"+"$x"+"$y" "$OUT/$1"
}
click_capsule() {
  local cx=$(( X+W*68/100 )) cy=$(( Y+H-27 ))
  xdotool mousemove --sync "$cx" "$cy" click 1
}
if ! expect_size no; then RESULT=NO_GO; REASON='Compact HUD geometry invalid.'; exit 1; fi
capture HUD_BEFORE.png
for n in $(seq 1 "$CYCLES"); do
  geometry; click_capsule
  if ! expect_size yes; then RESULT=NO_GO; REASON="Popup failed to expand on cycle $n"; exit 1; fi
  if (( n == 1 )); then capture HUD_EXPANDED.png; fi
  click_capsule
  if ! expect_size no; then RESULT=NO_GO; REASON="Popup failed to collapse on cycle $n"; exit 1; fi
  DONE="$n"
done
RESULT=PASS_FUNCTIONAL_PENDING_VISUAL
echo "X11 mouse popup cycles: $DONE/$CYCLES; images in $OUT"
