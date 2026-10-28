/**
 * Shared target resolution for the global tour and per-project tours.
 * A scoped tour MUST NOT fall back to document: split panes can render
 * identical data-tour markers for multiple projects at once.
 */
export function findTourTarget(
  scope: ParentNode | null | undefined,
  selector: string,
): HTMLElement | null {
  if (!scope) return null;
  for (const candidate of scope.querySelectorAll<HTMLElement>(selector)) {
    const rect = candidate.getBoundingClientRect();
    // Off-screen targets remain valid because the tour scrolls them into view.
    if (rect.width > 0 && rect.height > 0 && rect.right > 0 && rect.bottom > 0) {
      return candidate;
    }
  }
  return null;
}
