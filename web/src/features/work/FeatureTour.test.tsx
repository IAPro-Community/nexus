import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { FeatureTour } from './FeatureTour';

describe('FeatureTour compact mode', () => {
  it('renders full button label by default', () => {
    const html = renderToStaticMarkup(
      React.createElement(FeatureTour, {
        projectId: 'proj-1',
        surface: 'agent-terminal',
      }),
    );

    expect(html).toContain('data-compact="false"');
    expect(html).toContain('Conhecer esta área');
  });

  it('omits visible button text and inline notice pill in compact mode while preserving aria-label', () => {
    const html = renderToStaticMarkup(
      React.createElement(FeatureTour, {
        projectId: 'proj-1',
        surface: 'agent-terminal',
        compact: true,
      }),
    );

    expect(html).toContain('data-compact="true"');
    expect(html).toContain('aria-label="Conhecer esta área"');
    expect(html).not.toContain('role="status"');
    expect(html).not.toContain('>Conhecer esta área<');
  });
});
