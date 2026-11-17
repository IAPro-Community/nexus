import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { NexAssistantPanel } from './NexAssistantPanel';

describe('Nex assistant presentation and privacy', () => {
  it('retains the assistant in the HUD without revealing the conversation in privacy mode', () => {
    const html = renderToStaticMarkup(<NexAssistantPanel projectId="p1" privacyMode />);
    expect(html).toContain('Assistente Nex');
    expect(html).toContain('oculto pelo modo de privacidade');
    expect(html).not.toContain('Pergunte ao Nex');
  });

  it('keeps the manual assistant collapsed when the HUD is not private', () => {
    const html = renderToStaticMarkup(<NexAssistantPanel />);
    expect(html).toContain('Assistente Nex');
    expect(html).not.toContain('Permitir enviar fatos estáticos');
  });
});
