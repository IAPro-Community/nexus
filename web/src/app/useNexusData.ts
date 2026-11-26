import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api';
import { nexus } from '../nexus/api';
import { asArray } from '../lib/safeArray';
import type {
  Agent,
  EventRecord,
  ProfileInfo,
  Project,
  ProviderInfo,
  RuntimeSession,
  Workspace,
} from '../types';

/** Runtime/event cadence: sessions react to agent work in progress. */
export const FAST_POLL_MS = 5000;
/** Metadata cadence: projects, workspaces, providers and profiles. */
export const METADATA_POLL_MS = 30000;

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

function sameEventSnapshot(a: EventRecord[], b: EventRecord[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    const left = a[i];
    const right = b[i];
    if (
      left.id !== right.id ||
      left.type !== right.type ||
      left.timestamp !== right.timestamp ||
      left.summary !== right.summary
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

/**
 * Shared promise wrapper for the two polling lanes: concurrent callers reuse the
 * request already in flight, and the slot is cleared once it settles so the
 * next tick starts a fresh one.
 */
const runDeduped = <T>(
  slot: { current: Promise<T> | null },
  task: () => Promise<T>,
): Promise<T> => {
  if (slot.current) return slot.current;
  const request = task().finally(() => {
    slot.current = null;
  });
  slot.current = request;
  return request;
};

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
  const fastInFlight = useRef<Promise<{ runtimes: RuntimeSession[] }> | null>(null);
  const metadataInFlight = useRef<Promise<void> | null>(null);

  // Fast lane (5s): runtime/session state and the event feed change while an
  // agent works, so they stay on the short cadence.
  const refreshFast = useCallback(() => {
    return runDeduped(fastInFlight, async () => {
      try {
        const [runtimeList, eventList] = await Promise.all([api.getRuntimes(), api.getEvents()]);
        const nextRuntimes = asArray<RuntimeSession>(runtimeList);
        const nextEvents = asArray<EventRecord>(eventList);
        setRuntimes((current) =>
          sameRuntimeSnapshot(current, nextRuntimes) ? current : nextRuntimes,
        );
        setEvents((current) => (sameEventSnapshot(current, nextEvents) ? current : nextEvents));
        setError('');
        return { runtimes: nextRuntimes };
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Unable to load Nexus data');
        return { runtimes: [] };
      } finally {
        setLoading(false);
      }
    });
  }, []);

  // Metadata lane (30s): projects, workspaces, providers and profiles only
  // change after an explicit user action, and those actions call refreshGlobal()
  // so the slow lane is synchronous there too.
  const refreshMetadata = useCallback(() => {
    return runDeduped(metadataInFlight, async () => {
      try {
        const [projectList, workspaceList, providerList, profileList] = await Promise.all([
          nexus.listProjects(),
          api.getWorkspaces(),
          api.getProviders(),
          api.getProfiles(),
        ]);
        const nextProjects = asArray<Project>(projectList);
        const nextWorkspaces = asArray<Workspace>(workspaceList);
        const nextProviders = asArray<ProviderInfo>(providerList);
        const nextProfiles = asArray<ProfileInfo>(profileList);

        setProjects((current) =>
          sameProjectSnapshot(current, nextProjects) ? current : nextProjects,
        );
        setWorkspaces((current) =>
          sameIdList(current, nextWorkspaces, (item) => item.id || '') ? current : nextWorkspaces,
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
        setError('');
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Unable to load Nexus data');
      } finally {
        setLoading(false);
      }
    });
  }, []);

  // Public entry point used by user actions (create/remove project, login,
  // explicit reloads): both lanes run in parallel and the fast result is
  // returned, so callers keep the previous `{ runtimes }` contract.
  const refreshGlobal = useCallback(async () => {
    const [, runtimes] = await Promise.all([refreshMetadata(), refreshFast()]);
    return runtimes;
  }, [refreshFast, refreshMetadata]);

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
      if (!document.hidden) void refreshFast();
    };
    const refreshMetadataWhenVisible = () => {
      if (!document.hidden) void refreshMetadata();
    };
    const fastTimer = window.setInterval(refreshWhenVisible, FAST_POLL_MS);
    const metadataTimer = window.setInterval(refreshMetadataWhenVisible, METADATA_POLL_MS);
    document.addEventListener('visibilitychange', refreshWhenVisible);
    document.addEventListener('visibilitychange', refreshMetadataWhenVisible);
    return () => {
      window.clearInterval(fastTimer);
      window.clearInterval(metadataTimer);
      document.removeEventListener('visibilitychange', refreshWhenVisible);
      document.removeEventListener('visibilitychange', refreshMetadataWhenVisible);
    };
  }, [refreshFast, refreshGlobal, refreshMetadata]);

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
