import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import './i18n';
import { NexCompactBar, resolveCompactQuotaLabel } from './NexCompactBar';
import { NexInteractionPanel } from './NexInteractionPanel';
import { NexNotificationBubble } from './NexNotificationBubble';
import { NexPresenceApp, deriveNexV2Mood } from './NexPresenceApp';
import { NexSettingsDrawer } from './NexSettingsDrawer';
import {
  buildNexHubHeaders,
  defaultPlatformCapabilities,
  defaultPresenceState,
  normalizeHudState,
} from './nexV2Client';
import {
  alertQualifiesForNotificationLevel,
  resolveDefaultNexV2Tab,
  resolveTopNexBubbleAlert,
} from './nexV2HudHelpers';
import type { NexV2HudState } from './nexV2Types';

const noop = () => {};

describe('Nex 2.0 presence shell and modular surfaces', () => {
  it('mounts the presence shell without mounting the lazy interaction panel initially', () => {
    const html = renderToStaticMarkup(<NexPresenceApp />);
    expect(html).toContain('data-nex-presence="true"');
    expect(html).toContain('shellAvatarOnly');
    expect(html).not.toContain('data-nex-panel="true"');
    expect(html).not.toContain('data-nex-settings="true"');

    const settingsOverPanelHtml = renderToStaticMarkup(
      <NexPresenceApp initialPanelOpen={true} initialSettingsOpen={true} />,
    );
    expect(settingsOverPanelHtml).toContain('data-nex-settings="true"');
    expect(settingsOverPanelHtml).not.toContain('data-nex-panel="true"');
    expect(settingsOverPanelHtml).not.toContain('loadingFallback');
  });

  it('renders strictly 3 main tabs, operational status strip, and attention badge inside NexInteractionPanel', () => {
    const hudWithAlert: NexV2HudState = {
      generated_at: '2026-11-25T10:00:00Z',
      providers: [],
      agents: [{ id: 'a1', project_id: 'p1', name: 'Coder', status: 'WORKING' }],
      missions: [],
      alerts: [
        {
          fingerprint: 'fp-1',
          category: 'quota',
          severity: 'CRITICAL',
          reason: 'Low quota',
          acked: false,
          last_observed_at: '2026-11-25T10:00:00Z',
        },
      ],
      runtime_attention: [],
      primary: { kind: 'ALERT', id: 'fp-1' },
      system: {
        capacity: 'OK',
        resources: 'OK',
        agents: 'OK',
        missions: 'OK',
        cost: 'UNKNOWN',
        cache_ttl_millis: 15_000,
      },
    };
    expect(resolveDefaultNexV2Tab(hudWithAlert)).toBe('attention');
    expect(resolveDefaultNexV2Tab(null)).toBe('resources');

    const html = renderToStaticMarkup(
      <NexInteractionPanel
        activeTab="resources"
        mood="critical"
        hudState={hudWithAlert}
        offline={false}
        onSelectTab={noop}
        onClose={noop}
        onOpenSettings={noop}
        onAckAlert={noop}
        onSnoozeAlert={noop}
      />,
    );
    const tabMatches = html.match(/role="tab"/g) ?? [];
    expect(tabMatches).toHaveLength(3);
    expect(html).toContain('data-nex-tab="resources"');
    expect(html).toContain('data-nex-tab="attention"');
    expect(html).toContain('data-nex-tab="chat"');
    expect(html).toContain('data-nex-tab-badge="1"');
    expect(html).toContain('data-nex-panel-settings="true"');
  });

  it('keeps the compact bar usable without an avatar and reports Sem leitura / UNKNOWN rather than 0% when quota is missing', () => {
    const html = renderToStaticMarkup(
      <NexCompactBar
        mood="idle"
        hudState={null}
        offline={false}
        doNotDisturb={true}
        panelOpen={false}
        onTogglePanel={noop}
        onOpenSettings={noop}
      />,
    );
    expect(html).toContain('data-quota-state="UNKNOWN"');
    expect(html).toContain('Sem leitura');
    expect(html).not.toContain('0%');
    expect(html).toContain('Silencioso');
    expect(html).toContain('data-nex-compact-open="true"');
    expect(html).toContain('data-nex-compact-settings="true"');
    expect(resolveCompactQuotaLabel(null, true)).toBe('UNKNOWN');
  });

  it('suppresses NexNotificationBubble under Do Not Disturb, open panel, or offline state', () => {
    const alert = {
      fingerprint: 'fp-1',
      category: 'quota',
      severity: 'WARNING',
      provider: 'claude',
      reason: 'Quota low',
      acked: false,
      last_observed_at: '2026-11-25T10:00:00Z',
    };

    const visibleHtml = renderToStaticMarkup(
      <NexNotificationBubble
        alert={alert}
        doNotDisturb={false}
        panelOpen={false}
        offline={false}
        onDismiss={noop}
        onOpenAttention={noop}
      />,
    );
    expect(visibleHtml).toContain('Quota low');

    expect(
      renderToStaticMarkup(
        <NexNotificationBubble
          alert={alert}
          doNotDisturb={true}
          panelOpen={false}
          offline={false}
          onDismiss={noop}
          onOpenAttention={noop}
        />,
      ),
    ).toBe('');

    expect(
      renderToStaticMarkup(
        <NexNotificationBubble
          alert={alert}
          doNotDisturb={false}
          panelOpen={true}
          offline={false}
          onDismiss={noop}
          onOpenAttention={noop}
        />,
      ),
    ).toBe('');

    expect(
      renderToStaticMarkup(
        <NexNotificationBubble
          alert={alert}
          doNotDisturb={false}
          panelOpen={false}
          offline={true}
          onDismiss={noop}
          onOpenAttention={noop}
        />,
      ),
    ).toBe('');
  });

  it('renders UNKNOWN and STALE provider states honestly without converting missing quota to zero', () => {
    const hudState: NexV2HudState = {
      generated_at: '2026-11-25T10:00:00Z',
      providers: [
        {
          provider: 'claude',
          profile: 'default',
          health: 'UNKNOWN',
          availability: 'UNKNOWN',
          capacity_state: 'UNKNOWN',
          freshness: 'STALE',
          rate_limited: false,
          windows: [],
        },
      ],
      agents: [],
      missions: [],
      alerts: [],
      runtime_attention: [],
      primary: { kind: 'EMPTY' },
      system: {
        capacity: 'UNKNOWN',
        resources: 'UNKNOWN',
        agents: 'UNKNOWN',
        missions: 'UNKNOWN',
        cost: 'UNKNOWN',
        cache_ttl_millis: 15_000,
      },
    };
    const html = renderToStaticMarkup(
      <NexInteractionPanel
        activeTab="resources"
        hudState={hudState}
        offline={false}
        onSelectTab={noop}
        onClose={noop}
        onOpenSettings={noop}
        onAckAlert={noop}
        onSnoozeAlert={noop}
      />,
    );
    expect(html).toContain('claude');
    expect(html).toContain('data-freshness="STALE"');
    expect(html).toContain('Desatualizado');
    expect(html).toContain('data-quota-state="UNKNOWN"');
    expect(html).toContain('Sem leitura');
    expect(html).not.toContain('0%');
  });

  it('surfaces Wayland capability limits honestly in NexSettingsDrawer and segmented display controls', () => {
    const html = renderToStaticMarkup(
      <NexSettingsDrawer
        open={true}
        canGoBack={true}
        state={defaultPresenceState()}
        capabilities={{
          ...defaultPlatformCapabilities(),
          globalPositioning: false,
          session: 'wayland',
          globalHotkeyReason: 'Restricted on Wayland',
        }}
        onClose={noop}
        onPresetChange={noop}
        onDisplayModeChange={noop}
        onToggleAlwaysOnTop={noop}
        onToggleDoNotDisturb={noop}
        onTogglePaused={noop}
        onToggleReducedMotion={noop}
      />,
    );
    expect(html).toContain('data-nex-settings="true"');
    expect(html).toContain('data-nex-settings-back="true"');
    expect(html).toContain('data-nex-display-mode="avatar"');
    expect(html).toContain('data-nex-display-mode="bar"');
    expect(html).toContain('data-nex-display-mode="both"');
    expect(html).toContain('data-nex-notif-level="important"');
    expect(html).toContain('data-nex-notif-cooldown="30"');
    expect(html).toContain('data-nex-toggle="hud-ondemand"');
    expect(html).toContain('Wayland');

    const normalized = normalizeHudState({
      providers: null as unknown as NexV2HudState['providers'],
      agents: null as unknown as NexV2HudState['agents'],
      missions: null as unknown as NexV2HudState['missions'],
      alerts: null as unknown as NexV2HudState['alerts'],
      runtime_attention: null as unknown as NexV2HudState['runtime_attention'],
    });
    expect(normalized?.providers).toEqual([]);
    expect(normalized?.alerts).toEqual([]);
    expect(deriveNexV2Mood(normalized, true)).toBe('offline');
  });

  it('wires NexPresenceApp reconnect action when Core is offline', () => {
    const html = renderToStaticMarkup(
      <NexInteractionPanel
        activeTab="chat"
        hudState={null}
        offline={true}
        onSelectTab={noop}
        onClose={noop}
        onOpenSettings={noop}
        onAckAlert={noop}
        onSnoozeAlert={noop}
      />,
    );
    expect(html).toContain('data-nex-chat="true"');
    expect(html).toContain('data-nex-quick-prompts="true"');
  });

  it('unifies alerts, runtime attention waits, and blocked missions in resolveTopNexBubbleAlert and renders +N badge', () => {
    const hud: NexV2HudState = {
      generated_at: '2026-11-26T23:00:00Z',
      providers: [],
      agents: [],
      missions: [
        {
          run_id: 'run-1',
          project_id: 'proj-1',
          state: 'BLOCKED_NEEDS_USER',
          current_stage: 'review',
          current_task_title: 'Review API',
          provider: 'codex',
          completed_packages: 1,
          total_packages: 2,
          needs_user: true,
          intervention_reason: 'Confirm migration',
        },
      ],
      alerts: [
        {
          fingerprint: 'warn-1',
          category: 'quota',
          severity: 'WARNING',
          provider: 'claude',
          reason: 'Quota low',
          acked: false,
          last_observed_at: '2026-11-26T23:00:00Z',
        },
        {
          fingerprint: 'crit-1',
          category: 'quota',
          severity: 'CRITICAL',
          provider: 'codex',
          reason: 'Quota exhausted',
          acked: false,
          last_observed_at: '2026-11-26T23:00:00Z',
        },
      ],
      runtime_attention: [
        {
          runtime_id: 'rt-1',
          project_name: 'Nexus',
          provider: 'codex',
          reason: 'QUESTION',
          prompt_kind: 'yn',
          fingerprint: 'fp-rt-1',
          updated_at: '2026-11-26T23:00:00Z',
        },
      ],
      primary: { kind: 'ALERT', id: 'crit-1' },
      system: {
        capacity: 'OK',
        resources: 'OK',
        agents: 'OK',
        missions: 'OK',
        cost: 'UNKNOWN',
        cache_ttl_millis: 8_000,
      },
    };

    const first = resolveTopNexBubbleAlert(hud, {});
    expect(first.totalPending).toBe(4);
    expect(first.alert?.fingerprint).toBe('crit-1');

    const bubbleHtml = renderToStaticMarkup(
      <NexNotificationBubble
        alert={first.alert}
        pendingCount={first.totalPending}
        doNotDisturb={false}
        panelOpen={false}
        offline={false}
        onDismiss={noop}
        onOpenAttention={noop}
      />,
    );
    expect(bubbleHtml).toContain('+3');

    const second = resolveTopNexBubbleAlert(hud, { 'crit-1': true });
    expect(second.totalPending).toBe(3);
    expect(second.alert?.fingerprint).toBe('runtime:rt-1:fp-rt-1');

    const third = resolveTopNexBubbleAlert(hud, {
      'crit-1': true,
      'runtime:rt-1:fp-rt-1': true,
    });
    expect(third.totalPending).toBe(2);
    expect(third.alert?.fingerprint).toBe('mission:run-1:review');

    // Notification intensity filtering preserves totalPending count for the badge
    // while keeping NEX Hub active to avoid OS fallback popup floods.
    const quiet = resolveTopNexBubbleAlert(hud, {}, undefined, 'quiet');
    expect(quiet.alert).toBeNull();
    expect(quiet.totalPending).toBe(4);

    const criticalOnly = resolveTopNexBubbleAlert(hud, { 'crit-1': true }, undefined, 'critical');
    expect(criticalOnly.alert).toBeNull();
    expect(criticalOnly.totalPending).toBe(3);

    const gentleQuotaAlert = {
      fingerprint: 'warn-25',
      category: 'quota',
      severity: 'WARNING',
      remaining_percent: 25,
      acked: false,
      last_observed_at: '2026-11-26T23:00:00Z',
    };
    expect(alertQualifiesForNotificationLevel(gentleQuotaAlert, 'all')).toBe(true);
    expect(alertQualifiesForNotificationLevel(gentleQuotaAlert, 'important')).toBe(false);
    expect(
      alertQualifiesForNotificationLevel(
        { ...gentleQuotaAlert, remaining_percent: 10 },
        'important',
      ),
    ).toBe(true);

    expect(
      buildNexHubHeaders({
        ...defaultPresenceState(),
        notificationLevel: 'quiet',
        hudMemoryMode: 'ondemand',
      }),
    ).toEqual({
      'X-Nexus-Nex-Hub': 'active',
      'X-Nexus-Nex-Dnd': '1',
    });
  });
});
