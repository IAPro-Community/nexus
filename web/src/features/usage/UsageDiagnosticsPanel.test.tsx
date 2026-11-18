import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { UsageDiagnosticsPanel } from './UsageDiagnosticsPanel';

const baseProps = {
  accounts: [],
  monitorByAccount: {},
  unconfiguredProviders: [],
  onRefreshAccount: () => undefined,
  onRequestLogin: () => undefined,
  onRegisterProvider: () => undefined,
};

describe('UsageDiagnosticsPanel', () => {
  it('renders an accessible read-only section with a polite live summary', () => {
    const html = renderToStaticMarkup(React.createElement(UsageDiagnosticsPanel, baseProps));

    expect(html).toContain('aria-label="Diagnósticos de coleta"');
    expect(html).toContain('aria-live="polite"');
    expect(html).toContain('Sem evidências: nenhuma conta registrada ainda.');
  });

  it('never invents evidence when there is no data', () => {
    const html = renderToStaticMarkup(React.createElement(UsageDiagnosticsPanel, baseProps));

    // No observation rows, no deduced counts, no healthy claims.
    expect(html).not.toContain('Problemas deduzidos:');
    expect(html).not.toContain('Nenhum problema deduzido');
    expect(html).not.toContain('<time');
    expect(html).toContain('Sem contas para inspecionar.');
  });

  it('reports the Maestro status as still being checked, not as available', () => {
    const html = renderToStaticMarkup(React.createElement(UsageDiagnosticsPanel, baseProps));

    expect(html).toContain('Integração Maestro');
    expect(html).toContain('role="status"');
    expect(html).toContain('Verificando status do Maestro…');
    expect(html).not.toContain('Maestro disponível');
  });

  it('exposes section headings for keyboard browsing', () => {
    const html = renderToStaticMarkup(React.createElement(UsageDiagnosticsPanel, baseProps));

    expect(html).toContain('id="usage-diagnostics-collection"');
    expect(html).toContain('id="usage-diagnostics-states"');
    expect(html).toContain('id="usage-diagnostics-maestro"');
    expect(html).toContain('id="usage-diagnostics-problems"');
  });
});
