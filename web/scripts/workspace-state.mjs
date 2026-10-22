export async function dumpWorkspaceStateInPage() {
  const summarizeWorkspaceModel = (model) => {
    if (!model || model.root?.kind !== 'stack') return null;
    return {
      root: model.root.kind,
      tabs: (model.root.tabs || []).map((tab) => ({
        id: tab.id,
        type: tab.type,
        runtimeId: tab.data?.runtimeId || '',
      })),
      activeId: model.root.activeId,
    };
  };
  const out = { localStorage: {}, serverLayout: null, layoutRevision: null };
  for (let index = 0; index < localStorage.length; index++) {
    const key = localStorage.key(index) || '';
    if (!key.includes('workspace')) continue;
    let parsed = null;
    try {
      parsed = JSON.parse(localStorage.getItem(key) || 'null');
    } catch {
      // Keep invalid local values out of the summary.
    }
    out.localStorage[key] = typeof parsed === 'object' ? summarizeWorkspaceModel(parsed) : parsed;
  }
  try {
    const projects = await (
      await fetch('/api/v1/projects', { headers: { Accept: 'application/json' } })
    ).json();
    const projectId = Array.isArray(projects) ? projects[0]?.id : projects?.projects?.[0]?.id;
    out.projectId = projectId || '';
    if (projectId) {
      const detail = await (
        await fetch(`/api/v1/projects/${projectId}`, { headers: { Accept: 'application/json' } })
      ).json();
      out.layoutRevision = detail.revision ?? null;
      out.serverLayout = summarizeWorkspaceModel(JSON.parse(detail.layout || 'null'));
    }
  } catch (error) {
    out.layoutError = String(error?.message || error);
  }
  return out;
}
