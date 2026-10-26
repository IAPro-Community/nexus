import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import type { CompanionAttention } from './companionModel';
import { GuardianMascot } from './GuardianMascot';

const attention: CompanionAttention = {
  id: 'mission:project-1:run-1',
  titleKey: 'hud.companion.missionTitle',
  detailKey: 'hud.companion.missionDetail',
  severity: 'critical',
  target: 'missions',
  needsUser: true,
  projectId: 'project-1',
  recordId: 'run-1',
};

const renderGuardian = (overrides: Partial<React.ComponentProps<typeof GuardianMascot>> = {}) =>
  renderToStaticMarkup(
    React.createElement(GuardianMascot, {
      mood: 'attention',
      count: 2,
      expanded: false,
      motion: true,
      onExpand: () => undefined,
      onOpenAttention: () => undefined,
      onSnooze: () => undefined,
      ...overrides,
    }),
  );

describe('NEX single Guardian mascot', () => {
  it('renders the official NEX mascot as the only visual, with accessible pending controls', () => {
    const html = renderGuardian();

    expect(html).toContain('data-nex-art="32bit-official"');
    expect(html.match(/data-nex-art=/g) || []).toHaveLength(1);
    expect(html).toContain('viewBox="0 0 128 128"');
    expect(html).toContain('data-nex-part="helmet"');
    expect(html).toContain('data-nex-part="face"');
    expect(html).toContain('data-nex-part="reactor"');
    expect((html.match(/data-nex-module=/g) || []).length).toBe(4);
    expect(html).not.toContain('data-companion-sprite="atlas"');
    expect(html).not.toContain('/companion/');
    expect(html).toContain('data-hud-mascot="true"');
    expect(html).not.toContain('data-companion-bucket');
    expect(html).toContain('data-hud-hit-area="true"');
    expect(html).toContain('2');
  });

  it('keeps articulated fins and four independently anchored orbital modules', () => {
    const html = renderGuardian({ mood: 'working' });

    expect(html).toContain('data-nex-part="fins"');
    expect(html).toContain('data-nex-fin="left"');
    expect(html).toContain('data-nex-fin="right"');
    expect(html).toContain('data-nex-part="orbit-front"');
    expect((html.match(/data-nex-module=/g) || [])).toHaveLength(4);
    expect(html).toContain('transform="translate(14 96)"');
    expect(html).toContain('transform="translate(114 96)"');
    expect(html).toContain('transform="translate(30 115)"');
    expect(html).toContain('transform="translate(98 115)"');
    expect(html).toContain('data-motion="on"');
  });

  it('disables all decorative motion when offline or explicitly disabled', () => {
    for (const overrides of [{ mood: 'offline' as const }, { motion: false }]) {
      const html = renderGuardian(overrides);
      expect(html).toContain('data-motion="off"');
    }
  });

  it('shows actionable attention copy only for a real projected item', () => {
    const html = renderGuardian({ bubble: attention });

    expect(html).toContain('data-companion-bubble');
    expect(html).toContain('Uma missão precisa da sua atenção');
    expect(html).toContain('Abrir');
    expect(html).toContain('Adiar');
  });

  it('keeps the actionable attention bubble visible in the native HUD', () => {
    const html = renderGuardian({ bubble: attention, native: true });

    expect(html).toContain('data-companion-bubble');
    expect(html).toContain('data-hud-hit-area="true"');
    expect(html).toContain('Abrir');
    expect(html).toContain('Adiar');
  });

  it('keeps the attention bubble hidden when there is no actionable item', () => {
    expect(renderGuardian()).not.toContain('data-companion-bubble');
  });

  it('turns off sprite motion when disabled and exposes native drag by keyboard', () => {
    const html = renderGuardian({ motion: false, native: true });

    expect(html).toContain('data-motion="off"');
    expect(html).toContain('data-hud-drag-handle="true"');
    expect(html).toContain('aria-label="Arrastar HUD. Use as setas para mover."');
  });

  it('shows the critical state for priority attention', () => {
    const critical = renderGuardian({ mood: 'critical' });

    expect(critical).toContain('data-mood="critical"');
    expect(critical).toContain('#FF496C');
  });
});
