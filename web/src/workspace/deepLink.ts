/**
 * Deep link support for Nexus workspace.
 *
 * Format: /projects/:projectId?focus=agent:<id> or ?focus=flow-run:<id>
 *
 * Deep links are stable - they reference logical keys, not runtime IDs.
 * If a view is not open, it will be reconstructed by logical key.
 */

export interface DeepLinkTarget {
  projectId: string;
  focus?: {
    type: 'agent' | 'flow-run' | 'view';
    id: string;
  };
}

/**
 * Parse a deep link URL into a structured target.
 */
export function parseDeepLink(url: string, base?: string): DeepLinkTarget | null {
  try {
    const baseOrigin = base || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost');
    const parsed = new URL(url, baseOrigin);

    // Must be a project path
    if (!parsed.pathname.startsWith('/projects/')) {
      return null;
    }

    const pathParts = parsed.pathname.split('/').filter(Boolean);
    if (pathParts.length < 2 || pathParts[0] !== 'projects') {
      return null;
    }

    const projectId = pathParts[1];
    if (!projectId) {
      return null;
    }

    const focusParam = parsed.searchParams.get('view');
    let focus: DeepLinkTarget['focus'];

    if (focusParam) {
      const [type, ...idParts] = focusParam.split(':');
      const id = idParts.join(':');
      if (type && id) {
        focus = { type: type as DeepLinkTarget['focus']['type'], id };
      }
    }

    return { projectId, focus };
  } catch {
    return null;
  }
}

/**
 * Generate a deep link URL for a project and optional focus target.
 */
export function generateDeepLink(projectId: string, focus?: { type: string; id: string }, base?: string): string {
  const baseOrigin = base || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost');
  const url = new URL(`/projects/${projectId}`, baseOrigin);
  if (focus) {
    url.searchParams.set('view', `${focus.type}:${focus.id}`);
  }
  return url.pathname + url.search;
}

/**
 * Apply a deep link target to the current workspace.
 * This reconstructs views by logical key if they are not already open.
 */
export function applyDeepLink(
  target: DeepLinkTarget,
  openView: (projectId: string, logicalKey: string) => void,
  focusView: (logicalKey: string) => void,
): void {
  if (target.focus) {
    // First try to focus the existing view
    const focused = focusView(target.focus.id);
    if (!focused) {
      // If not found, open it by logical key
      openView(target.projectId, target.focus.id);
    }
  }
}
