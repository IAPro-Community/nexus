import type { HudHitRegion } from '../../platform/wailsV3';

export const MAX_HUD_HIT_REGIONS = 256;

const IGNORE_ATTRIBUTE = 'ignore';

export interface HudHitCandidate {
  ignore: boolean;
  pointerEvents: string;
  display: string;
  visibility: string;
  opacity: string;
  rect: { left: number; top: number; width: number; height: number; right: number; bottom: number };
}

/**
 * Converts one DOM node snapshot into a hit rectangle, or null when the node
 * must never capture native mouse input.
 *
 * Pure on purpose: the HUD tests run without a DOM, so the filtering rules stay
 * verifiable without a browser environment.
 */
export function hudRegionFromCandidate(candidate: HudHitCandidate): HudHitRegion | null {
  if (candidate.ignore) return null;
  if (
    candidate.pointerEvents === 'none' ||
    candidate.display === 'none' ||
    candidate.visibility === 'hidden'
  ) {
    return null;
  }
  const opacity = candidate.opacity === '' ? 1 : Number(candidate.opacity);
  if (!Number.isFinite(opacity) || opacity === 0) return null;

  const { left, top, width, height, right, bottom } = candidate.rect;
  if (width < 1 || height < 1 || right <= 0 || bottom <= 0) return null;

  return { x: left, y: top, width, height };
}

/**
 * Collects the rectangles of the HUD surface that must keep native mouse input.
 *
 * The native window is a rectangle, but most of it is a transparent surface.
 * Windows routes every click to the top-most native window, so `pointer-events:
 * none` in CSS is not enough: the rectangles reported here are the only areas
 * where the native window may swallow the click.
 *
 * Visible interactive controls opt in with `data-hud-hit-area`. Decorative
 * overlays can opt out with `data-hud-hit="ignore"`.
 */
export function collectHudHitRegions(root: ParentNode): HudHitRegion[] {
  const regions: HudHitRegion[] = [];
  // A Document has no ownerDocument; the root itself is the document then.
  const doc = root.ownerDocument ?? (root as Document);
  const view = doc.defaultView ?? (typeof window === 'undefined' ? undefined : window);
  if (!view) return regions;

  // The panel itself stretches to the viewport in native mode, so its
  // bounding rect would make the entire transparent native window absorb
  // clicks. Only visible interactive surfaces opt in to native hit-testing.
  // The popover opts in only for the legacy *same-window* presentation.
  for (const node of Array.from(root.querySelectorAll<HTMLElement>('[data-hud-hit-area]'))) {
    const style = view.getComputedStyle(node);
    const region = hudRegionFromCandidate({
      ignore: node.dataset.hudHit === IGNORE_ATTRIBUTE,
      pointerEvents: style.pointerEvents,
      display: style.display,
      visibility: style.visibility,
      opacity: style.opacity,
      rect: node.getBoundingClientRect(),
    });
    if (!region) continue;
    regions.push(region);
    if (regions.length >= MAX_HUD_HIT_REGIONS) break;
  }

  return regions;
}
