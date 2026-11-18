import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { ConsumptionBreakdownPanel } from './ConsumptionBreakdownPanel';

describe('ConsumptionBreakdownPanel', () => {
  it('renders an accessible section with a live loading status before evidence arrives', () => {
    const html = renderToStaticMarkup(React.createElement(ConsumptionBreakdownPanel));

    expect(html).toContain('aria-label="Consumo por missão e agente"');
    expect(html).toContain('role="status"');
    expect(html).toContain('Carregando evidências de consumo…');
    // No drill-down content is invented before the observation list loads.
    expect(html).not.toContain('Observações individuais');
    expect(html).not.toContain('Por missão (run)');
  });

  it('offers the same 7/30/90 period control as the analytics panel', () => {
    const html = renderToStaticMarkup(React.createElement(ConsumptionBreakdownPanel));

    expect(html).toContain('aria-label="Período analisado"');
    expect(html.match(/aria-pressed="true"/g) || []).toHaveLength(1);
    expect(html).toContain('7 dias');
    expect(html).toContain('30 dias');
    expect(html).toContain('90 dias');
  });
});
