import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { nexus } from '../../nexus/api';
import { asArray } from '../../lib/safeArray';
import type { Agent, AttentionItem, ProjectAgentSummary } from '../../types';

const EXPANDED_KEY = 'nx_rail_expanded_projects';

export type ProjectOutlineAttention = {
  needsYou: AttentionItem[];
  failed: AttentionItem[];
};

function loadExpandedIds(): Set<string> {
  try {
    const raw = localStorage.getItem(EXPANDED_KEY);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw) as unknown;
    return new Set(asArray<string>(parsed).filter((id) => typeof id === 'string'));
  } catch {
    return new Set();
  }
}

function persistExpandedIds(ids: Set<string>) {
  try {
    localStorage.setItem(EXPANDED_KEY, JSON.stringify([...ids]));
  } catch {
    /* ignore quota / private mode */
  }
}

export function groupAttentionByProject(attention: {
  needs_you?: AttentionItem[] | null;
  failed?: AttentionItem[] | null;
}): Map<string, ProjectOutlineAttention> {
  const map = new Map<string, ProjectOutlineAttention>();
  const ensure = (projectId: string): ProjectOutlineAttention => {
    let entry = map.get(projectId);
    if (!entry) {
      entry = { needsYou: [], failed: [] };
      map.set(projectId, entry);
    }
    return entry;
  };
  for (const item of asArray<AttentionItem>(attention.needs_you)) {
    if (!item?.project_id) continue;
    ensure(item.project_id).needsYou.push(item);
  }
  for (const item of asArray<AttentionItem>(attention.failed)) {
    if (!item?.project_id) continue;
    ensure(item.project_id).failed.push(item);
  }
  return map;
}

export function summariesByProjectId(
  summaries: ProjectAgentSummary[] | null | undefined,
): Map<string, ProjectAgentSummary> {
  const map = new Map<string, ProjectAgentSummary>();
  for (const summary of asArray<ProjectAgentSummary>(summaries)) {
    if (!summary?.project_id) continue;
    map.set(summary.project_id, summary);
  }
  return map;
}

export function useProjectRailOutline() {
  const [summaries, setSummaries] = useState<ProjectAgentSummary[]>([]);
  const [attentionByProject, setAttentionByProject] = useState<
    Map<string, ProjectOutlineAttention>
  >(() => new Map());
  const [expandedIds, setExpandedIds] = useState<Set<string>>(loadExpandedIds);
  const [agentsByProject, setAgentsByProject] = useState<Map<string, Agent[]>>(() => new Map());
  const [loadingAgents, setLoadingAgents] = useState<Set<string>>(() => new Set());
  const agentsCacheRef = useRef(agentsByProject);
  const loadingRef = useRef(loadingAgents);
  const loadGenRef = useRef(new Map<string, number>());
  const expandedIdsRef = useRef(expandedIds);
  agentsCacheRef.current = agentsByProject;
  loadingRef.current = loadingAgents;
  expandedIdsRef.current = expandedIds;

  const refreshOutline = useCallback(async () => {
    const [summaryList, attention] = await Promise.all([
      nexus.listProjectSummaries().catch(() => [] as ProjectAgentSummary[]),
      nexus.getAttention().catch(() => ({
        needs_you: [],
        completed: [],
        failed: [],
        all_items: [],
        total_needs: 0,
      })),
    ]);
    setSummaries(asArray<ProjectAgentSummary>(summaryList));
    setAttentionByProject(groupAttentionByProject(attention));
  }, []);

  useEffect(() => {
    void refreshOutline();
  }, [refreshOutline]);

  const summaryMap = useMemo(() => summariesByProjectId(summaries), [summaries]);

  const setExpanded = useCallback((projectId: string, expanded: boolean) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (expanded) next.add(projectId);
      else next.delete(projectId);
      persistExpandedIds(next);
      return next;
    });
  }, []);

  const toggleExpanded = useCallback(
    (projectId: string) => {
      const willExpand = !expandedIds.has(projectId);
      setExpanded(projectId, willExpand);
      return willExpand;
    },
    [expandedIds, setExpanded],
  );

  const loadAgents = useCallback(async (projectId: string) => {
    // Cached or already in-flight: do not flip loading and flash the outline.
    if (agentsCacheRef.current.has(projectId) || loadingRef.current.has(projectId)) return;
    const gen = (loadGenRef.current.get(projectId) || 0) + 1;
    loadGenRef.current.set(projectId, gen);
    setLoadingAgents((prev) => {
      const next = new Set(prev).add(projectId);
      loadingRef.current = next;
      return next;
    });
    try {
      const agents = asArray<Agent>(await nexus.listAgents(projectId));
      if (loadGenRef.current.get(projectId) !== gen) return;
      setAgentsByProject((prev) => {
        const next = new Map(prev);
        next.set(projectId, agents);
        return next;
      });
    } catch {
      if (loadGenRef.current.get(projectId) !== gen) return;
      setAgentsByProject((prev) => {
        const next = new Map(prev);
        next.set(projectId, []);
        return next;
      });
    } finally {
      if (loadGenRef.current.get(projectId) === gen) {
        setLoadingAgents((prev) => {
          const next = new Set(prev);
          next.delete(projectId);
          loadingRef.current = next;
          return next;
        });
      }
    }
  }, []);

  useEffect(() => {
    for (const projectId of expandedIds) {
      void loadAgents(projectId);
    }
  }, [expandedIds, loadAgents]);

  const invalidateAgents = useCallback(
    (projectId: string) => {
      const next = new Map(agentsCacheRef.current);
      next.delete(projectId);
      agentsCacheRef.current = next;
      setAgentsByProject(next);
      loadGenRef.current.set(projectId, (loadGenRef.current.get(projectId) || 0) + 1);
      if (loadingRef.current.has(projectId)) {
        const loading = new Set(loadingRef.current);
        loading.delete(projectId);
        loadingRef.current = loading;
        setLoadingAgents(loading);
      }
      if (expandedIdsRef.current.has(projectId)) {
        void loadAgents(projectId);
      }
    },
    [loadAgents],
  );

  useEffect(() => {
    const onAgentsChanged = (event: Event) => {
      const projectId = (event as CustomEvent<{ projectId?: string }>).detail?.projectId;
      if (!projectId) return;
      invalidateAgents(projectId);
      void refreshOutline();
    };
    window.addEventListener('nexus:agents-changed', onAgentsChanged);
    return () => window.removeEventListener('nexus:agents-changed', onAgentsChanged);
  }, [invalidateAgents, refreshOutline]);

  return {
    summaryMap,
    attentionByProject,
    expandedIds,
    agentsByProject,
    loadingAgents,
    toggleExpanded,
    setExpanded,
    loadAgents,
    invalidateAgents,
    refreshOutline,
  };
}
