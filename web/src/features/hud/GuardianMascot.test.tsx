import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { GuardianMascot, GuardianModeSelector } from './GuardianMascot';
import type { GuardianMood, HudAppearance } from './guardianModel';

const renderMascot = (mood: GuardianMood, attention = false) =>
  renderToStaticMarkup(React.createElement(GuardianMascot, { mood, attention }));

const renderSelector = (appearance: HudAppearance, onSelect: (next: HudAppearance) => void) => {
  const props = { appearance, pending: false, onSelect };
  return renderToStaticMarkup(React.createElement(GuardianModeSelector, props));
};

describe('Nexus Guardian pixel mascot', () => {
  it('renders a standalone vector sprite with a working mood', () => {
    const html = renderMascot('working');
    expect(html).toContain('data-mood="working"');
    expect(html).toContain('viewBox="0 0 64 64"');
    expect(html).toContain('shape-rendering="crispEdges"');
    expect(html).not.toContain('Atenção!');
  });

  it('only shows a readable speech bubble for observed attention', () => {
    const html = renderMascot('warning', true);
    expect(html).toContain('Atenção!');
    expect(html).toContain('data-mood="warning"');
  });

  it('provides three accessible native presentation controls', () => {
    const selected: string[] = [];
    const html = renderSelector('MASCOT', (mode) => {
      selected.push(mode);
    });
    expect(html).toContain('aria-label="Apresentação nativa do Guardian"');
    expect(html).toContain('Mascote');
    expect(html).toContain('Bandeja');
    expect(html).toContain('aria-pressed="true"');
    expect(selected).toEqual([]);
  });

  it('keeps mascot selectable without an independent popup', () => {
    const html = renderSelector('BAR', () => undefined);
    expect(html).toContain('Exibir mascote flutuante');
    expect(html).not.toContain('disabled=""');
  });
});
