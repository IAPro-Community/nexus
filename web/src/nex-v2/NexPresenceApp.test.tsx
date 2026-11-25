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
  defaultPlatformCapabilities,
  defaultPresenceState,
  normalizeHudState,
} from './nexV2Client';
import type { NexV2HudState } from './nexV2Types';

const noop = () => {};

describe('Nex 2.0 presence shell and modular surfaces', () => {
  it('mounts the presence shell without mounting the lazy interaction panel initially', () => {
    const html = renderToStaticMarkup(<NexPresenceApp />);
    expect(html).toContain('data-nex-presence="true"');
    expect(html).not.toContain('data-nex-panel="true"');
    expect(html).not.toContain('data-nex-settings="true"');
  });

  it('renders strictly 3 main tabs and a separate settings trigger inside NexInteractionPanel', () => {
    const html = renderToStaticMarkup(
      <NexInteractionPanel
        activeTab="chat"
        hudState={null}
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
    expect(html).toContain('data-nex-tab="chat"');
    expect(html).toContain('data-nex-tab="resources"');
    expect(html).toContain('data-nex-tab="attention"');
    expect(html).toContain('data-nex-panel-settings="true"');
  });

  it('keeps the compact bar usable without an avatar and reports UNKNOWN rather than 0% when quota is missing', () => {
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
    expect(html).toContain('UNKNOWN');
    expect(html).not.toContain('0%');
    expect(html).toContain('DND');
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
    expect(html).toContain('STALE');
    expect(html).toContain('UNKNOWN');
    expect(html).not.toContain('0%');
  });

  it('surfaces Wayland capability limits honestly in NexSettingsDrawer and normalizes nullable Go slices', () => {
    const html = renderToStaticMarkup(
      <NexSettingsDrawer
        open={true}
        state={defaultPresenceState()}
        capabilities={{
          ...defaultPlatformCapabilities(),
          globalPositioning: false,
          session: 'wayland',
          globalHotkeyReason: 'Restricted on Wayland',
        }}
        onClose={noop}
        onPresetChange={noop}
        onToggleShowAvatar={noop}
        onToggleCompactBar={noop}
        onToggleAlwaysOnTop={noop}
        onToggleDoNotDisturb={noop}
        onTogglePaused={noop}
        onToggleReducedMotion={noop}
      />,
    );
    expect(html).toContain('data-nex-settings="true"');
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
});
