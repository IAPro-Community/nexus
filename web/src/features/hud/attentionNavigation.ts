import type { HudRuntimeAttention } from './hudModel';

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
