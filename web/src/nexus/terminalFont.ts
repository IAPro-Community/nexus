// Canvas-based terminal renderers cannot reliably parse CSS var() inside
// their font shorthand. Resolve the workspace token before xterm measures
// character cells, otherwise the cursor/grid can diverge from drawn glyphs.
export const TERMINAL_FONT_FALLBACK = 'Menlo, Monaco, Consolas, "Courier New", monospace';

export function resolveTerminalFontFamily(element: Element): string {
  if (typeof getComputedStyle !== 'function') return TERMINAL_FONT_FALLBACK;
  const family = getComputedStyle(element).getPropertyValue('--nx-font-mono').trim();
  if (!family || /\bvar\s*\(/i.test(family)) return TERMINAL_FONT_FALLBACK;
  return family.replace(/\s+/g, ' ');
}
