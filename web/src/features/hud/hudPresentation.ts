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
