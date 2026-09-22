import type { ComposerSessionView } from '../../types';
import type { NexusActivityItem, NexusActivityPayload } from '../conversation';

export type ComposerBootstrapStepEvent = {
  type: 'step';
  id: string;
  status: 'started' | 'done' | 'skipped' | 'error' | string;
};

export type ComposerBootstrapHeartbeat = {
  type: 'heartbeat';
  elapsed_ms?: number;
};

export type ComposerBootstrapComplete = {
  type: 'complete';
  view: ComposerSessionView;
};

export type ComposerBootstrapError = {
  type: 'error';
  message: string;
};

export type ComposerBootstrapEvent =
  | ComposerBootstrapStepEvent
  | ComposerBootstrapHeartbeat
  | ComposerBootstrapComplete
  | ComposerBootstrapError;

const STEP_LABELS: Record<string, string> = {
  brief: 'Building brief',
  git_context: 'Reading repository context',
  memories: 'Applying memories',
  project_intel: 'Loading project intelligence',
  context_discovery: 'Selecting bounded context',
  extract_brief: 'Extracting prompt structure',
  unknowns: 'Identifying gaps',
  persist: 'Saving session',
};

/**
 * Streams POST /composer-sessions with Accept: application/x-ndjson.
 * Falls back callers should use JSON create if this throws before first byte.
 */
export async function createComposerSessionWithProgress(
  projectId: string,
  body: { goal: string; input_mode: string; source_prompt?: string },
  options: {
    signal?: AbortSignal;
    onEvent: (event: ComposerBootstrapEvent) => void;
    csrfToken?: string | null;
  },
): Promise<ComposerSessionView> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/x-ndjson',
  };
  if (options.csrfToken) headers['X-CSRF-Token'] = options.csrfToken;

  const res = await fetch(`/api/v1/projects/${projectId}/composer-sessions`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
    signal: options.signal,
    credentials: 'include',
  });

  if (!res.ok) {
    const errBody = (await res.json().catch(() => ({ error: res.statusText }))) as {
      error?: string;
    };
    throw new Error(errBody.error || `HTTP ${res.status}`);
  }

  if (!res.body) {
    throw new Error('empty progress stream');
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let complete: ComposerSessionView | null = null;

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      const event = JSON.parse(trimmed) as ComposerBootstrapEvent;
      options.onEvent(event);
      if (event.type === 'complete') complete = event.view;
      if (event.type === 'error') throw new Error(event.message || 'create failed');
    }
  }

  if (buffer.trim()) {
    const event = JSON.parse(buffer.trim()) as ComposerBootstrapEvent;
    options.onEvent(event);
    if (event.type === 'complete') complete = event.view;
    if (event.type === 'error') throw new Error(event.message || 'create failed');
  }

  if (!complete) throw new Error('create stream ended without a session');
  return complete;
}

export function bootstrapStepsToActivity(
  steps: Map<string, NexusActivityItem['status']>,
  title: string,
): NexusActivityPayload {
  const order = [
    'brief',
    'git_context',
    'memories',
    'project_intel',
    'context_discovery',
    'extract_brief',
    'unknowns',
    'persist',
  ];
  const items: NexusActivityItem[] = order.map((id) => ({
    id,
    label: STEP_LABELS[id] || id,
    status: steps.get(id) || 'pending',
  }));
  const current =
    [...items].reverse().find((i) => i.status === 'running') ||
    items.find((i) => i.status === 'pending');
  return {
    title,
    currentId: current?.id,
    items,
    expandable: true,
  };
}

export function mapNetworkError(
  err: unknown,
  t: (key: string, fallback: string) => string,
): string {
  if (err instanceof DOMException && err.name === 'AbortError') {
    return t('work.composer.cancelled', 'Elaboration cancelled.');
  }
  const message = err instanceof Error ? err.message : String(err);
  if (/failed to fetch|networkerror|load failed/i.test(message)) {
    return t(
      'work.composer.networkError',
      'Could not reach the Nexus server. Check that nexus web is running and try again.',
    );
  }
  return message;
}
