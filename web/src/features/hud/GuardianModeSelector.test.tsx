import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { GuardianModeSelector } from './GuardianModeSelector';

describe('GuardianModeSelector', () => {
  it('offers one mascot presentation without a separate 2D companion mode', () => {
    const html = renderToStaticMarkup(
      React.createElement(GuardianModeSelector, {
        appearance: 'MASCOT',
        pending: false,
        onSelect: () => undefined,
      }),
    );

    expect(html).toContain('Mascote');
    expect(html).toContain('Barra');
    expect(html).toContain('Bandeja');
    expect(html.match(/<button\b/g) || []).toHaveLength(3);
    expect(html.match(/aria-pressed="true"/g) || []).toHaveLength(1);
    expect(html).not.toContain('Companion 2D');
    expect(html).not.toContain('2D');
  });
});
