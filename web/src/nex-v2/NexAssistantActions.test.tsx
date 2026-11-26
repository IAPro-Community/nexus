import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import './i18n';
import { NexChatTab } from './NexChatTab';
import { NexReceiptsPanel } from './NexReceiptsPanel';
import type { NexV2ActionReceipt, NexV2ChatTurn, NexV2PersonalMemoryEntry } from './nexV2Types';

const noop = () => {};

const SAMPLE_MEMORIES: NexV2PersonalMemoryEntry[] = [
  {
    id: 'mem-1',
    key: 'idioma',
    value: 'pt-BR-secret-pref',
    scope: '@personal',
    provenance: 'user_explicit',
    created_at: '2026-11-25T12:00:00Z',
    updated_at: '2026-11-25T12:00:00Z',
  },
];

const SAMPLE_TURNS: NexV2ChatTurn[] = [
  {
    sequence: 1,
    role: 'user',
    content: 'abrir agentes',
  },
  {
    sequence: 2,
    role: 'assistant',
    content: 'Posso abrir essa área do Nexus.',
    source: 'nexus_snapshot',
    proposal: {
      id: 'nexus.open.agents',
      title: 'Abrir agentes',
      description: 'Navegar para a lista de agentes.',
      target: 'agents',
      risk: 'navigation',
      requires_confirmation: true,
    },
  },
];

const SAMPLE_RECEIPTS: NexV2ActionReceipt[] = [
  {
    receipt_id: 'rcpt-1',
    action_id: 'open_nexus_surface',
    risk: 'navigation',
    owner: 'nexus.ui',
    status: 'EXECUTED',
    summary: 'navigation approved: agents',
    created_at: '2026-11-25T12:00:00Z',
    completed_at: '2026-11-25T12:00:01Z',
  },
  {
    receipt_id: 'rcpt-2',
    action_id: 'ack_capacity_alert',
    risk: 'mutate',
    owner: 'nexus.capacity',
    status: 'DENIED',
    summary: 'denied by user',
    created_at: '2026-11-25T12:01:00Z',
    completed_at: '2026-11-25T12:01:01Z',
  },
  {
    receipt_id: 'rcpt-3',
    action_id: 'delegate_agent_task',
    risk: 'delegate',
    owner: 'nexus.agents',
    status: 'UNAVAILABLE',
    summary: 'delegation must be dispatched from the Agents surface',
    created_at: '2026-11-25T12:02:00Z',
    completed_at: '2026-11-25T12:02:01Z',
  },
];

describe('Nex 2.0 P4/P5 Assistant, Personal Memory & Audited Receipts', () => {
  it('renders memory count, receipts count, proposal controls, and open receipts panel', () => {
    const html = renderToStaticMarkup(
      <NexChatTab
        projectId="prj-demo"
        privacyMode={false}
        initialHistory={SAMPLE_TURNS}
        initialMemories={SAMPLE_MEMORIES}
        initialReceipts={SAMPLE_RECEIPTS}
        initialReceiptsOpen={true}
      />,
    );

    expect(html).toContain('data-nex-memory-toggle="true"');
    expect(html).toContain('Memória (1)');
    expect(html).toContain('data-nex-receipts-toggle="true"');
    expect(html).toContain('Atividade (3)');
    expect(html).toContain('data-nex-proposal="nexus.open.agents"');
    expect(html).toContain('data-nex-proposal-confirm="true"');
    expect(html).toContain('data-nex-proposal-dismiss="true"');
    expect(html).toContain('data-nex-receipts-panel="true"');
    expect(html).toContain('navigation approved: agents');
  });

  it('renders NexReceiptsPanel empty state and newest-first receipt ordering', () => {
    const emptyHtml = renderToStaticMarkup(
      <NexReceiptsPanel
        receipts={[]}
        privacyMode={false}
        runningAction={false}
        onRunReadHud={noop}
      />,
    );
    expect(emptyHtml).toContain('Nenhuma ação registrada.');
    expect(emptyHtml).toContain('data-nex-action-read-hud="true"');

    const populatedHtml = renderToStaticMarkup(
      <NexReceiptsPanel
        receipts={SAMPLE_RECEIPTS}
        privacyMode={false}
        runningAction={false}
        onRunReadHud={noop}
      />,
    );
    expect(populatedHtml).toContain('data-receipt-status="EXECUTED"');
    expect(populatedHtml).toContain('data-receipt-status="DENIED"');
    expect(populatedHtml).toContain('data-receipt-status="UNAVAILABLE"');

    const idx3 = populatedHtml.indexOf('data-nex-receipt="rcpt-3"');
    const idx2 = populatedHtml.indexOf('data-nex-receipt="rcpt-2"');
    const idx1 = populatedHtml.indexOf('data-nex-receipt="rcpt-1"');
    expect(idx3).toBeGreaterThan(-1);
    expect(idx3).toBeLessThan(idx2);
    expect(idx2).toBeLessThan(idx1);
  });

  it('masks receipt summaries under privacyMode without hiding status', () => {
    const html = renderToStaticMarkup(
      <NexReceiptsPanel
        receipts={SAMPLE_RECEIPTS}
        privacyMode={true}
        runningAction={false}
        onRunReadHud={noop}
      />,
    );
    expect(html).not.toContain('navigation approved: agents');
    expect(html).not.toContain('delegation must be dispatched');
    expect(html).toContain('•••');
    expect(html).toContain('data-receipt-status="EXECUTED"');
    expect(html).toContain('data-receipt-status="UNAVAILABLE"');
  });
});
