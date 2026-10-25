// Pure presentation contract shared by the compact capsule, native popup,
// and legacy inline HUD. Rendering detached content while its window is
// hidden prevents a blank first frame during the native Show() transition.
export function shouldRenderHudPopover(
  expanded: boolean,
  native: boolean,
  nativeView: 'capsule' | 'popover',
  detached: boolean,
): boolean {
  if (!native) return expanded;
  if (nativeView === 'popover') return true;
  return expanded && !detached;
}

/** Keep the mascot in a compact native capsule but give a single-window fallback all expanded space. */
export function shouldRenderHudMascot(
  native: boolean,
  nativeView: 'capsule' | 'popover',
  appearance: 'BAR' | 'MASCOT' | 'TRAY',
  expanded: boolean,
  detached: boolean,
): boolean {
  return native && nativeView === 'capsule' && appearance === 'MASCOT' && (!expanded || detached);
}

/** The mascot/bar compact control stays visible while a detached popup is open. */
export function shouldRenderHudCompactControl(
  native: boolean,
  nativeView: 'capsule' | 'popover',
  expanded: boolean,
  detached: boolean,
): boolean {
  if (nativeView === 'popover') return false;
  return !native || !expanded || detached;
}
