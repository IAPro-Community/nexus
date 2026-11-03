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

/**
 * Font loading may finish after xterm's first FitAddon measurement. Refit the
 * grid once the fonts are ready, and when font loading restarts (for example
 * after a workspace theme or font package change). Return an exact disposer.
 */
export function watchTerminalFontMetrics(onChange: () => void): () => void {
  if (typeof document === 'undefined' || !document.fonts) return () => {};
  const fonts = document.fonts;
  let active = true;
  const handleChange = () => {
    if (active) onChange();
  };
  fonts.addEventListener('loadingdone', handleChange);
  void fonts.ready.then(handleChange).catch(() => {
    // A failed font load must not stop the terminal's normal resize path.
  });
  return () => {
    active = false;
    fonts.removeEventListener('loadingdone', handleChange);
  };
}
