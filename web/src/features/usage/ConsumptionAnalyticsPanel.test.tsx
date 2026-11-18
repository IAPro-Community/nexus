import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { ConsumptionAnalyticsPanel } from './ConsumptionAnalyticsPanel';

describe('ConsumptionAnalyticsPanel', () => {
  it('keeps the CSV export available but disabled until analytics data arrives', () => {
    const html = renderToStaticMarkup(React.createElement(ConsumptionAnalyticsPanel));

    expect(html).toContain('aria-label="Análise de consumo"');
    expect(html).toContain('Exportar CSV');
    expect(html).toContain('disabled');
    // The daily metric radio group only exists once the report is loaded.
    expect(html).not.toContain('radiogroup');
  });

  it('never renders a fabricated daily chart before evidence arrives', () => {
    const html = renderToStaticMarkup(React.createElement(ConsumptionAnalyticsPanel));

    expect(html).toContain('role="status"');
    expect(html).not.toContain('barTrack');
    expect(html).not.toContain('Invocações');
  });
});
