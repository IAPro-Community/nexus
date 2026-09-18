import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api';
import { nexus } from '../nexus/api';
import type {
  Agent,
  EventRecord,
  ProfileInfo,
  Project,
  ProviderInfo,
  RuntimeSession,
  Workspace,
} from '../types';

function sameProjectSnapshot(a: Project[], b: Project[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    const left = a[i];
    const right = b[i];
    if (
      left.id !== right.id ||
      left.name !== right.name ||
      left.canonical_path !== right.canonical_path ||
      left.default_branch !== right.default_branch ||
      left.last_opened_at !== right.last_opened_at ||
      left.updated_at !== right.updated_at
    ) {
      return false;
    }
  }
  return true;
}

function sameRuntimeSnapshot(a: RuntimeSession[], b: RuntimeSession[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    const left = a[i];
    const right = b[i];
    if (
      left.runtime_id !== right.runtime_id ||
      left.agent_id !== right.agent_id ||
      left.state !== right.state ||
      left.attention_kind !== right.attention_kind ||
      left.attention_reason !== right.attention_reason ||
      left.dynamic_title !== right.dynamic_title ||
      left.provider_id !== right.provider_id ||
      left.profile_id !== right.profile_id
    ) {
      return false;
    }
  }
  return true;
}

function sameIdList<T extends { id?: string }>(a: T[], b: T[], pick: (item: T) => string): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    if (pick(a[i]) !== pick(b[i])) return false;
  }
  return true;
}

export function useNexusData() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const agentsRef = useRef(agents);
  agentsRef.current = agents;
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [runtimes, setRuntimes] = useState<RuntimeSession[]>([]);
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [profiles, setProfiles] = useState<ProfileInfo[]>([]);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const refreshInFlight = useRef<Promise<{ runtimes: RuntimeSession[] }> | null>(null);

  const refreshGlobal = useCallback(async () => {
    if (refreshInFlight.current) return refreshInFlight.current;

    const request = (async () => {
      try {
        const [projectList, workspaceList, runtimeList, providerList, profileList, eventList] =
          await Promise.all([
            nexus.listProjects(),
            api.getWorkspaces(),
            api.getRuntimes(),
            api.getProviders(),
            api.getProfiles(),
            api.getEvents(),
          ]);
        const nextProjects = Array.isArray(projectList) ? projectList : [];
        const nextWorkspaces = Array.isArray(workspaceList) ? workspaceList : [];
        const nextRuntimes = Array.isArray(runtimeList) ? runtimeList : [];
        const nextProviders = Array.isArray(providerList) ? providerList : [];
        const nextProfiles = Array.isArray(profileList) ? profileList : [];
        const nextEvents = Array.isArray(eventList) ? eventList : [];

        setProjects((current) =>
          sameProjectSnapshot(current, nextProjects) ? current : nextProjects,
        );
        setWorkspaces((current) =>
          sameIdList(current, nextWorkspaces, (item) => item.id || '') ? current : nextWorkspaces,
        );
        setRuntimes((current) =>
          sameRuntimeSnapshot(current, nextRuntimes) ? current : nextRuntimes,
        );
        setProviders((current) =>
          sameIdList(current, nextProviders, (item) => item.id || '') ? current : nextProviders,
        );
        setProfiles((current) => {
          if (current.length !== nextProfiles.length) return nextProfiles;
          for (let i = 0; i < current.length; i += 1) {
            if (
              current[i].name !== nextProfiles[i].name ||
              current[i].provider !== nextProfiles[i].provider ||
              current[i].authenticated !== nextProfiles[i].authenticated
            ) {
              return nextProfiles;
            }
          }
          return current;
        });
        setEvents((current) =>
          sameIdList(current, nextEvents, (item) => item.id || '') ? current : nextEvents,
        );
        setError('');
        return { runtimes: nextRuntimes };
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Unable to load Nexus data');
        return { runtimes: [] };
      } finally {
        setLoading(false);
        refreshInFlight.current = null;
      }
    })();
    refreshInFlight.current = request;
    return request;
  }, []);

  const refreshAgents = useCallback(async (projectId?: string | null) => {
    if (!projectId) {
      setAgents([]);
      return;
    }
    try {
      const list = await nexus.listAgents(projectId);
      const next = Array.isArray(list) ? list : [];
      const current = agentsRef.current;
      let changed = current.length !== next.length;
      if (!changed) {
        for (let i = 0; i < current.length; i += 1) {
          if (
            current[i].id !== next[i].id ||
            current[i].status !== next[i].status ||
            current[i].name !== next[i].name
          ) {
            changed = true;
            break;
          }
        }
      }
      if (changed) {
        setAgents(next);
        window.dispatchEvent(new CustomEvent('nexus:agents-changed', { detail: { projectId } }));
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load Agents');
    }
  }, []);

  useEffect(() => {
    void refreshGlobal();
    const refreshWhenVisible = () => {
      if (!document.hidden) void refreshGlobal();
    };
    const timer = window.setInterval(refreshWhenVisible, 5000);
    document.addEventListener('visibilitychange', refreshWhenVisible);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', refreshWhenVisible);
    };
  }, [refreshGlobal]);

  return {
    projects,
    setProjects,
    agents,
    setAgents,
    workspaces,
    runtimes,
    providers,
    profiles,
    events,
    loading,
    error,
    refreshGlobal,
    refreshAgents,
  };
}
