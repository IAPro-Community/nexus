import { describe, expect, it } from 'vitest';
import {
  composerTurnsToEvents,
  nexusEventToThreadMessage,
  nexusEventsToThreadMessages,
} from './nexusConversationMapper';
import type { NexusConversationEvent } from './types';
import { isAllowedImageMime } from './types';

describe('nexusConversationMapper', () => {
  it('orders events by sequence', () => {
    const events: NexusConversationEvent[] = [
      {
        id: '2',
        sessionId: 's',
        sequence: 2,
        kind: 'assistant',
        content: 'b',
        createdAt: '2026-01-01T00:00:02Z',
      },
      {
        id: '1',
        sessionId: 's',
        sequence: 1,
        kind: 'user',
        content: 'a',
        createdAt: '2026-01-01T00:00:01Z',
      },
    ];
    const msgs = nexusEventsToThreadMessages(events);
    expect(msgs.map((m) => m.id)).toEqual(['1', '2']);
  });

  it('maps user text and images', () => {
    const msg = nexusEventToThreadMessage({
      id: 'u1',
      sessionId: 's',
      sequence: 1,
      kind: 'user',
      content: 'hello',
      payload: {
        attachments: [{ id: 'a1', name: 'x.png', mimeType: 'image/png', size: 10, url: 'blob:1' }],
      },
      createdAt: '2026-01-01T00:00:00Z',
    });
    expect(msg.role).toBe('user');
    expect(Array.isArray(msg.content)).toBe(true);
    const parts = msg.content as unknown as Array<{ type: string }>;
    expect(parts.some((p) => p.type === 'text')).toBe(true);
    expect(parts.some((p) => p.type === 'image')).toBe(true);
  });

  it('maps activity to data part', () => {
    const msg = nexusEventToThreadMessage({
      id: 'act',
      sessionId: 's',
      sequence: 3,
      kind: 'activity',
      payload: { title: 'Working', items: [{ id: '1', label: 'Step', status: 'running' }] },
      createdAt: '2026-01-01T00:00:00Z',
      status: 'running',
    });
    expect(msg.role).toBe('assistant');
    const part = (msg.content as unknown as Array<{ type: string; name?: string }>)[0];
    expect(part.type).toBe('data');
    expect(part.name).toBe('nexus.activity');
  });

  it('converts legacy composer turns', () => {
    const events = composerTurnsToEvents('sess', [
      {
        id: 't1',
        sequence: 1,
        role: 'USER',
        content: 'hi',
        created_at: '2026-01-01T00:00:00Z',
      },
      {
        id: 't2',
        sequence: 2,
        role: 'ASSISTANT',
        content: 'yo',
        created_at: '2026-01-01T00:00:01Z',
      },
    ]);
    expect(events[0].kind).toBe('user');
    expect(events[1].kind).toBe('assistant');
  });

  it('parses payload_json attachments on composer turns', () => {
    const events = composerTurnsToEvents('sess', [
      {
        id: 't1',
        sequence: 1,
        role: 'USER',
        content: 'with image',
        created_at: '2026-01-01T00:00:00Z',
        payload_json: JSON.stringify({
          attachments: [
            { id: 'a1', name: 'x.png', mimeType: 'image/png', url: '/api/v1/attachments/a1' },
          ],
        }),
      },
    ]);
    expect(events[0].payload?.attachments).toEqual([
      { id: 'a1', name: 'x.png', mimeType: 'image/png', url: '/api/v1/attachments/a1' },
    ]);
  });
});

describe('isAllowedImageMime', () => {
  it('accepts png jpeg webp only', () => {
    expect(isAllowedImageMime('image/png')).toBe(true);
    expect(isAllowedImageMime('image/jpeg')).toBe(true);
    expect(isAllowedImageMime('image/webp')).toBe(true);
    expect(isAllowedImageMime('image/gif')).toBe(false);
    expect(isAllowedImageMime('application/pdf')).toBe(false);
  });
});
