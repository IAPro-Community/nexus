import type { Project } from '../types';

/**
 * Resolve which project is selected.
 * When currentId is set and the poll returns an empty list, return null instead of
 * bouncing to projects[0] (that remounts WorkspaceProvider and flickers the rail).
 * When the catalog is non-empty but the id is gone, fall back to the first project.
 */
export function resolveProjectSelection(
  projects: Project[],
  currentId?: string | null,
): Project | null {
  const id = (currentId || '').trim();
  if (id) {
    const found = projects.find((project) => project.id === id);
    if (found) return found;
    if (projects.length === 0) return null;
    return projects[0] ?? null;
  }
  return projects[0] ?? null;
}

export type StaleProjectRoute =
  { kind: 'keep' } | { kind: 'wait' } | { kind: 'redirect'; projectId: string };

/**
 * URL /p/:id must not stay on an unknown id while the catalog already has
 * projects. Callers replace the route instead of silently selecting projects[0]
 * against a stale URL.
 */
export function resolveStaleProjectRoute(
  projects: Project[],
  routeProjectId?: string | null,
): StaleProjectRoute {
  const id = (routeProjectId || '').trim();
  if (!id) return { kind: 'keep' };
  if (projects.some((project) => project.id === id)) return { kind: 'keep' };
  if (projects.length === 0) return { kind: 'wait' };
  const fallback = projects[0];
  if (!fallback) return { kind: 'wait' };
  return { kind: 'redirect', projectId: fallback.id };
}
