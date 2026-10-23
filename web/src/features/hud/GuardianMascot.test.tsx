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

describe('NEX official 32-bit pixel mascot', () => {
  it('renders a standalone vector sprite with a working mood', () => {
    const html = renderMascot('working');
    expect(html).toContain('data-mood="working"');
    expect(html).toContain('viewBox="0 0 128 128"');
    expect(html).toContain('data-nex-art="32bit-official"');
    expect(html).toContain('data-nex-facing="front"');
    expect(html).toContain('data-nex-part="helmet"');
    expect(html).toContain('data-nex-part="face"');
    expect(html).toContain('data-nex-part="fins"');
    expect(html).toContain('data-nex-part="reactor"');
    expect(html).toContain('data-nex-part="orbit"');
    expect(html).toContain('data-nex-part="orbitals"');
    expect((html.match(/data-nex-module="/g) || []).length).toBe(4);
    expect(html).toContain('shape-rendering="crispEdges"');
    expect(html).not.toContain('Atenção!');
  });

  it('only shows a readable speech bubble for observed attention', () => {
    const html = renderMascot('warning', true);
    expect(html).toContain('Atenção!');
    expect(html).toContain('data-mood="warning"');
    expect(html).toContain('#FF8A45');
  });

  it('keeps distinct focused and offline faces without claiming successful work', () => {
    const working = renderMascot('working');
    const offline = renderMascot('offline');
    expect(working).toContain('data-nex-part="face"');
    expect(working).toContain('#31D5FF');
    expect(offline).toContain('#8195AF');
    expect(offline).not.toContain('Atenção!');
  });

  it('avoids duplicate SVG gradients when the mascot is rendered more than once', () => {
    const html = renderToStaticMarkup(
      React.createElement('div', null, [
        React.createElement(GuardianMascot, { mood: 'idle', attention: false, key: 'left' }),
        React.createElement(GuardianMascot, { mood: 'working', attention: false, key: 'right' }),
      ]),
    );
    const gradientIds = [...html.matchAll(/<linearGradient id="([^"]+)"/g)].map((x) => x[1]);
    expect(gradientIds).toHaveLength(4);
    expect(new Set(gradientIds).size).toBe(4);
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
