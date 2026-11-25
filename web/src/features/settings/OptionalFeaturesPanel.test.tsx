import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, describe, expect, it } from 'vitest';
import '../../i18n';
import {
  OPTIONAL_FEATURE_KEYS,
  applyOptionalFeatureSnapshot,
  getOptionalFeatures,
} from '../../app/composerFlowFlag';
import { OptionalFeaturesPanel } from './OptionalFeaturesPanel';

describe('OptionalFeaturesPanel', () => {
  afterEach(() => {
    applyOptionalFeatureSnapshot(null);
  });

  it('lists all 11 optional feature flags as inactive by default with accurate env var guidance', () => {
    const html = renderToStaticMarkup(React.createElement(OptionalFeaturesPanel));

    expect(html).toContain('NEXUS_FEATURES_ENABLED=true');
    expect(html).toContain('NEXUS_FEATURE_&lt;NOME&gt;=true');

    for (const key of OPTIONAL_FEATURE_KEYS) {
      expect(html).toContain(`data-testid="feature-flag-${key}"`);
      expect(html).toContain(`NEXUS_FEATURE_${key}=true`);
    }

    expect(html).not.toContain('NEXUS_FEATURES_ENABLED=1');
    expect(html).not.toContain('Ativo');
    expect(html).toContain('Desativado');
  });

  it('reflects partially enabled flags from the server snapshot', () => {
    applyOptionalFeatureSnapshot({
      global: true,
      features: {
        COMPOSER: true,
        PARALLEL_CHALLENGE: true,
        FLOW: false,
      },
    });

    const html = renderToStaticMarkup(
      React.createElement(OptionalFeaturesPanel, { snapshot: getOptionalFeatures() }),
    );

    expect(html).toContain('Opt-in Avançado Ativo');
    expect(html).toContain('Ativo');
    expect(html).toContain('Desativado');
    expect(html).not.toContain('Como ativar os recursos opcionais');
  });
});
