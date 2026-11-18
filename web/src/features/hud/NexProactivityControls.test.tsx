import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { NexProactivityControls } from './NexProactivityControls';

describe('Nex proactive privacy and opt-in', () => {
  it('keeps all proactive controls out of the HUD when privacy is enabled', () => {
    const html = renderToStaticMarkup(<NexProactivityControls active privacyMode projectId="p1" />);
    expect(html).not.toContain('Iniciativa do Nex');
  });

  it('keeps the proactivity panel hidden while the HUD is collapsed', () => {
    const html = renderToStaticMarkup(
      <NexProactivityControls active={false} privacyMode={false} />,
    );
    expect(html).toBe('');
  });

  it('shows no synthetic suggestions before reading the canonical HUD', () => {
    const html = renderToStaticMarkup(<NexProactivityControls active privacyMode={false} />);
    expect(html).toContain('Iniciativa do Nex');
    expect(html).not.toContain('Missão aguardando decisão');
  });
});
