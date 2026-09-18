import type { ThreadMessageLike } from '@assistant-ui/react';
import type {
  NexusActivityPayload,
  NexusAttachmentRef,
  NexusConversationEvent,
  NexusToolPayload,
} from './types';

type ContentPart = Exclude<ThreadMessageLike['content'], string>[number];

/**
 * Isolates Nexus event ↔ assistant-ui ThreadMessageLike mapping.
 * Never import Composer domain types here.
 */
export function nexusEventToThreadMessage(event: NexusConversationEvent): ThreadMessageLike {
  const createdAt = new Date(event.createdAt);

  switch (event.kind) {
    case 'user': {
      const parts: ContentPart[] = [];
      if (event.content?.trim()) {
        parts.push({ type: 'text', text: event.content });
      }
      const images = attachmentImages(event);
      for (const img of images) {
        parts.push({
          type: 'image',
          image: img.url || `nexus-attachment://${img.id}`,
        });
      }
      if (parts.length === 0) {
        parts.push({ type: 'text', text: '' });
      }
      return {
        id: event.id,
        role: 'user',
        content: parts,
        createdAt,
      };
    }
    case 'assistant':
    case 'completion':
      return {
        id: event.id,
        role: 'assistant',
        content: [{ type: 'text', text: event.content || '' }],
        createdAt,
        status:
          event.status === 'running'
            ? { type: 'running' }
            : event.status === 'error'
              ? { type: 'incomplete', reason: 'error', error: event.content || 'error' }
              : { type: 'complete', reason: 'stop' },
      };
    case 'system':
    case 'status':
    case 'warning':
      return {
        id: event.id,
        role: 'system',
        content: [{ type: 'text', text: event.content || '' }],
        createdAt,
      };
    case 'error':
      return {
        id: event.id,
        role: 'assistant',
        content: [{ type: 'text', text: event.content || '' }],
        createdAt,
        status: {
          type: 'incomplete',
          reason: 'error',
          error: String(event.payload?.detail || event.content || 'error'),
        },
      };
    case 'activity':
      return {
        id: event.id,
        role: 'assistant',
        content: [
          {
            type: 'data',
            name: 'nexus.activity',
            data: (event.payload || {}) as Record<string, never>,
          },
        ],
        createdAt,
        status:
          event.status === 'running' ? { type: 'running' } : { type: 'complete', reason: 'stop' },
      };
    case 'tool_call':
    case 'tool_result': {
      const tool = (event.payload || {}) as Partial<NexusToolPayload>;
      const args = (tool.args || {}) as Record<string, string | number | boolean | null>;
      return {
        id: event.id,
        role: 'assistant',
        content: [
          {
            type: 'tool-call',
            toolCallId: event.id,
            toolName: tool.toolName || 'tool',
            args,
            argsText: JSON.stringify(args),
            result: tool.summary || tool.detail,
          },
        ],
        createdAt,
        status:
          tool.status === 'running' || tool.status === 'pending'
            ? { type: 'running' }
            : { type: 'complete', reason: 'stop' },
      };
    }
    case 'artifact':
    case 'attachment':
    case 'image':
    case 'approval':
      return {
        id: event.id,
        role: 'assistant',
        content: [
          {
            type: 'data',
            name: `nexus.${event.kind}`,
            data: {
              content: event.content ?? null,
              ...Object.fromEntries(
                Object.entries(event.payload || {}).map(([k, v]) => [
                  k,
                  typeof v === 'string' ||
                  typeof v === 'number' ||
                  typeof v === 'boolean' ||
                  v === null
                    ? v
                    : JSON.stringify(v),
                ]),
              ),
            },
          },
        ],
        createdAt,
      };
    default:
      return {
        id: event.id,
        role: 'assistant',
        content: [{ type: 'text', text: event.content || '' }],
        createdAt,
      };
  }
}

export function nexusEventsToThreadMessages(
  events: readonly NexusConversationEvent[],
): ThreadMessageLike[] {
  return [...events]
    .sort((a, b) => a.sequence - b.sequence || a.createdAt.localeCompare(b.createdAt))
    .map(nexusEventToThreadMessage);
}

export function asActivityPayload(raw: unknown): NexusActivityPayload | null {
  if (!raw || typeof raw !== 'object') return null;
  const obj = raw as Record<string, unknown>;
  if (typeof obj.title !== 'string' || !Array.isArray(obj.items)) return null;
  return raw as NexusActivityPayload;
}

export function asToolPayload(raw: unknown): NexusToolPayload | null {
  if (!raw || typeof raw !== 'object') return null;
  const obj = raw as Record<string, unknown>;
  if (typeof obj.toolName !== 'string') return null;
  return raw as NexusToolPayload;
}

/** Convert legacy ComposerTurn-shaped objects into conversation events. */
export function composerTurnsToEvents(
  sessionId: string,
  turns: ReadonlyArray<{
    id: string;
    sequence: number;
    role: string;
    content: string;
    created_at: string;
    kind?: string;
    payload?: Record<string, unknown>;
    payload_json?: string;
  }>,
): NexusConversationEvent[] {
  return turns.map((turn) => {
    const role = String(turn.role || '').toUpperCase();
    const kind =
      (turn.kind as NexusConversationEvent['kind'] | undefined) ||
      (role === 'USER' ? 'user' : role === 'SYSTEM' ? 'system' : 'assistant');
    let payload = turn.payload;
    if (!payload && turn.payload_json) {
      try {
        payload = JSON.parse(turn.payload_json) as Record<string, unknown>;
      } catch {
        payload = undefined;
      }
    }
    return {
      id: turn.id,
      sessionId,
      sequence: turn.sequence,
      kind,
      content: turn.content,
      payload,
      createdAt: turn.created_at,
      status: 'complete' as const,
    };
  });
}

function attachmentImages(event: NexusConversationEvent): NexusAttachmentRef[] {
  const fromPayload = event.payload?.attachments;
  if (Array.isArray(fromPayload)) {
    return fromPayload
      .filter((item): item is NexusAttachmentRef =>
        Boolean(
          item && typeof item === 'object' && typeof (item as NexusAttachmentRef).id === 'string',
        ),
      )
      .map((item) => ({
        ...item,
        url: item.url || `/api/v1/attachments/${item.id}`,
      }));
  }
  return [];
}
