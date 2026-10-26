import type { HudRuntimeAttention } from './hudModel';
import type { MascotAttention } from './mascotModel';

// Deep link to the owning Nexus workspace, not just the homepage. When an
// Agent is known, use the agent terminal surface; otherwise attach directly
// to the exact supervised runtime in the Terminals product.
export function hudAttentionRoute(item: HudRuntimeAttention): string {
  if (!item.project_id) return '/projects';
  const project = encodeURIComponent(item.project_id);
  if (item.agent_id) {
    return `/p/${project}/agents/${encodeURIComponent(item.agent_id)}`;
  }
  return `/p/${project}/terminals/${encodeURIComponent(item.runtime_id)}`;
}

export function mascotAttentionRoute(item: MascotAttention): string {
  if (!item.projectId) return '/projects';
  const project = encodeURIComponent(item.projectId);
  const record = item.recordId ? encodeURIComponent(item.recordId) : undefined;
  switch (item.target) {
    case 'missions':
      return record ? `/p/${project}/missions/${record}` : `/p/${project}/missions`;
    case 'agents':
      return record ? `/p/${project}/agents/${record}` : `/p/${project}/agents`;
    case 'usage':
      return `/p/${project}/resources`;
    default:
      return `/p/${project}/overview`;
  }
}
