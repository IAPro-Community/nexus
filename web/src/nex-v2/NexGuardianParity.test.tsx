import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import './i18n';
import { NexAttentionTab } from './NexAttentionTab';
import { NexResourcesTab } from './NexResourcesTab';
import { computeReconnectBackoffMs } from './nexV2Client';
import type {
  NexV2EventEnvelope,
  NexV2HudState,
  NexV2ModelCatalogWatchSnapshot,
} from './nexV2Types';

const noop = () => {};

const SAMPLE_HUD: NexV2HudState = {
  generated_at: '2026-11-25T12:00:00Z',
  project_id: 'prj-secret-alpha',
  providers: [
    {
      provider: 'claude',
      profile: 'secret-work-profile',
      health: 'WARNING',
      availability: 'AVAILABLE',
      capacity_state: 'LIMITED',
      freshness: 'STALE',
      rate_limited: true,
      windows: [
        {
          kind: '5h',
          group: 'claude-sonnet-4',
          label: 'Janela 5h',
          remaining_percent: 42,
          reset_description: 'em 2h',
          status: 'OK',
        },
        {
          kind: 'weekly',
          group: 'claude-opus-4',
          label: 'Semanal',
          status: 'UNKNOWN',
        },
      ],
    },
  ],
  agents: [{ id: 'a1', project_id: 'prj-secret-alpha', name: 'Coder', status: 'WORKING' }],
  missions: [
    {
      run_id: 'run-secret-9',
      project_id: 'prj-secret-alpha',
      state: 'BLOCKED_NEEDS_USER',
      current_stage: 'review',
      current_task_title: 'Approve migration',
      completed_packages: 1,
      total_packages: 3,
      needs_user: true,
      intervention_reason: 'Need user confirmation',
    },
  ],
  alerts: [
    {
      fingerprint: 'alert-fp-1',
      category: 'quota',
      severity: 'WARNING',
      provider: 'claude',
      profile: 'secret-work-profile',
      reason: 'Window 5h below 50%',
      acked: false,
      last_observed_at: '2026-11-25T12:00:00Z',
    },
  ],
  runtime_attention: [
    {
      runtime_id: 'rt-1',
      project_id: 'prj-secret-alpha',
      project_name: 'Secret Project Name',
      provider: 'codex',
      reason: 'Confirm command',
      prompt_kind: 'yn',
      fingerprint: 'rt-att-1',
      updated_at: '2026-11-25T12:00:00Z',
    },
  ],
  cost: {
    evidence: 'PARTIAL',
    token_evidence: 'PARTIAL',
    known_microusd: 4_250_000,
    unknown_observation_count: 1,
    budget_level: 'OK',
  },
  primary: { kind: 'ALERT', id: 'alert-fp-1' },
  system: {
    capacity: 'OK',
    resources: 'OK',
    agents: 'OK',
    missions: 'OK',
    cost: 'PARTIAL',
    cache_ttl_millis: 15_000,
  },
};

const SAMPLE_WATCH: NexV2ModelCatalogWatchSnapshot = {
  freshness: 'FRESH',
  entries: [
    {
      provider: 'claude',
      profile: 'secret-work-profile',
      model_id: 'claude-sonnet-4',
      is_default: true,
      deprecated: false,
      source: 'provider_snapshot',
      observed_at: '2026-11-25T12:00:00Z',
    },
  ],
  changes: [
    {
      fingerprint: 'mc-added-1',
      kind: 'ADDED',
      provider: 'claude',
      profile: 'secret-work-profile',
      model_id: 'claude-sonnet-4',
      source: 'provider_snapshot',
      observed_at: '2026-11-25T12:00:00Z',
    },
    {
      fingerprint: 'mc-dep-1',
      kind: 'DEPRECATED',
      provider: 'claude',
      profile: 'secret-work-profile',
      model_id: 'claude-opus-3',
      source: 'provider_snapshot',
      observed_at: '2026-11-25T12:00:00Z',
    },
  ],
};

describe('Nex 2.0 P3 Guardian operational parity', () => {
  it('renders multi-window quotas, resets, UNKNOWN window without 0%, cost evidence, and model watch', () => {
    const html = renderToStaticMarkup(
      <NexResourcesTab
        hudState={SAMPLE_HUD}
        modelWatch={SAMPLE_WATCH}
        offline={false}
        privacyMode={false}
      />,
    );
    expect(html).toContain('secret-work-profile');
    expect(html).toContain('42%');
    expect(html).toContain('em 2h');
    expect(html).toContain('data-quota-state="UNKNOWN"');
    expect(html).toContain('Sem leitura');
    expect(html).not.toContain('0%');
    expect(html).toContain('data-freshness="STALE"');
    expect(html).toContain('Desatualizado');
    expect(html).toContain('data-evidence="PARTIAL"');
    expect(html).toContain('Parcial');
    expect(html).toContain('4.25');
    expect(html).toContain('data-change-kind="ADDED"');
    expect(html).toContain('Novo modelo');
    expect(html).toContain('data-change-kind="DEPRECATED"');
    expect(html).toContain('Descontinuado');
  });

  it('masks profiles, project names, and spend amounts under privacyMode', () => {
    const resHtml = renderToStaticMarkup(
      <NexResourcesTab
        hudState={SAMPLE_HUD}
        modelWatch={SAMPLE_WATCH}
        offline={false}
        privacyMode={true}
      />,
    );
    expect(resHtml).not.toContain('secret-work-profile');
    expect(resHtml).not.toContain('4.25');
    expect(resHtml).toContain('•••');

    const attHtml = renderToStaticMarkup(
      <NexAttentionTab
        hudState={SAMPLE_HUD}
        events={[]}
        privacyMode={true}
        onAckAlert={noop}
        onSnoozeAlert={noop}
      />,
    );
    expect(attHtml).not.toContain('Secret Project Name');
    expect(attHtml).not.toContain('Approve migration');
    expect(attHtml).toContain('•••');
  });

  it('deduplicates repeated alerts/events in Attention tab and bounds reconnect backoff', () => {
    const events: NexV2EventEnvelope[] = [
      {
        revision: 1,
        event_id: 'e1',
        fingerprint: 'alert:alert-fp-1:WARNING',
        kind: 'CAPACITY_ALERT',
        severity: 'WARNING',
        source: 'capacity_monitor',
        title: 'claude',
        created_at: '2026-11-25T12:00:00Z',
      },
      {
        revision: 2,
        event_id: 'e2',
        fingerprint: 'model:mc-added-1',
        kind: 'MODEL_CATALOG_ADDED',
        severity: 'INFO',
        source: 'model_catalog_watch',
        title: 'claude · claude-sonnet-4',
        created_at: '2026-11-25T12:00:00Z',
      },
      {
        revision: 3,
        event_id: 'e3',
        fingerprint: 'model:mc-added-1',
        kind: 'MODEL_CATALOG_ADDED',
        severity: 'INFO',
        source: 'model_catalog_watch',
        title: 'claude · claude-sonnet-4',
        created_at: '2026-11-25T12:00:00Z',
      },
    ];
    const html = renderToStaticMarkup(
      <NexAttentionTab
        hudState={SAMPLE_HUD}
        events={events}
        privacyMode={false}
        onAckAlert={noop}
        onSnoozeAlert={noop}
      />,
    );
    expect(html.match(/model_catalog_watch|claude-sonnet-4/g)).toHaveLength(1);
    expect(computeReconnectBackoffMs(0)).toBe(0);
    expect(computeReconnectBackoffMs(1)).toBe(2000);
    expect(computeReconnectBackoffMs(2)).toBe(4000);
    expect(computeReconnectBackoffMs(3)).toBe(8000);
    expect(computeReconnectBackoffMs(10)).toBe(15_000);
  });
});
