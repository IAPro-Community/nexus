import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import type { MascotAttention } from './mascotModel';
import { GuardianMascot } from './GuardianMascot';

const attention: MascotAttention = {
  id: 'mission:project-1:run-1',
  titleKey: 'hud.mascot.missionTitle',
  detailKey: 'hud.mascot.missionDetail',
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

    expect(html).toContain('data-nex-art="hud-32bit-atlas"');
    expect(html.match(/data-nex-art=/g) || []).toHaveLength(1);
    // The default fixture projects a critical attention item, so the sprite
    // carries the warning treatment.
    expect(html).toContain('data-nex-mood="warning"');
    expect(html).not.toContain('data-companion-sprite="atlas"');
    expect(html).not.toContain('/companion/');
    expect(html).toContain('data-hud-mascot="true"');
    expect(html).toContain('data-blink="false"');
    expect(html).not.toContain('data-companion-bucket');
    expect(html).toContain('data-hud-hit-area="true"');
    expect(html).toContain('2');
  });

  it('keeps the operational mood rendered on the locked atlas sprite', () => {
    const html = renderGuardian({ mood: 'working' });

    expect(html).toContain('data-nex-mood="working"');
    expect(html).toContain('data-mood="working"');
    // Static SSR has no IntersectionObserver. Motion activates only after
    // client visibility is verified; the operational mood still renders.
    expect(html).toContain('data-motion="off"');
  });

  it('keeps the Companion still during SSR until visibility is established', () => {
    const html = renderGuardian({ mood: 'idle', motion: true });
    expect(html).toContain('data-nex-behavior="still"');
    expect(html).toContain('data-motion="off"');
    expect(html).not.toContain('data-idle-action=');
  });

  it('disables all decorative motion when offline or explicitly disabled', () => {
    for (const overrides of [{ mood: 'offline' as const }, { motion: false }]) {
      const html = renderGuardian(overrides);
      expect(html).toContain('data-motion="off"');
    }
  });

  it('shows actionable attention copy only for a real projected item', () => {
    const html = renderGuardian({ bubble: attention });

    expect(html).toContain('data-mascot-bubble');
    expect(html).toContain('Uma missão precisa da sua atenção');
    expect(html).toContain('Abrir');
    expect(html).toContain('Adiar');
  });

  it('keeps the actionable attention bubble visible in the native HUD', () => {
    const html = renderGuardian({ bubble: attention, native: true });

    expect(html).toContain('data-mascot-bubble');
    expect(html).toContain('data-hud-hit-area="true"');
    expect(html).toContain('Abrir');
    expect(html).toContain('Adiar');
  });

  it('keeps the attention bubble hidden when there is no actionable item', () => {
    expect(renderGuardian()).not.toContain('data-mascot-bubble');
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
    expect(critical).toContain('data-nex-mood="critical"');
  });
});
