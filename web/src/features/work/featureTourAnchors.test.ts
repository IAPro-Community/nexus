import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { featureTourSteps } from './featureTourDefinitions';

// Prevent documentation-only tours: all targets must correspond to actual
// UI anchors in the same release source. Browser tests still verify visibility.
const folder = dirname(fileURLToPath(import.meta.url));
const sourceFiles = [
  'ComposerStart.tsx',
  'ComposerWorkspaceHeader.tsx',
  'ComposerSurface.tsx',
  'ComposerInspector.tsx',
  'ComposerReadOnlyHistory.tsx',
  'FlowRunsHistorySurface.tsx',
  'FlowDraftEditor.tsx',
  'FlowRunSurface.tsx',
];
const source = sourceFiles.map((file) => readFileSync(join(folder, file), 'utf8')).join('\n');

describe('Composer/Flow tour implementation parity', () => {
  it('only references anchors actually rendered by the feature surfaces', () => {
    for (const [surface, steps] of Object.entries(featureTourSteps)) {
      for (const step of steps) {
        const selector = step.target.match(/^\[data-tour="([^"]+)"\]$/)?.[1];
        expect(selector, `${surface}/${step.id}`).toBeTruthy();
        expect(
          source.includes(`data-tour="${selector}"`),
          `${surface}/${step.id} is not connected to a DOM element`,
        ).toBe(true);
      }
    }
  });
  it('provides translations for all scoped tour keys in all three locales', () => {
    const resources = readFileSync(join(folder, '..', '..', 'i18n', 'resources.ts'), 'utf8');
    const tourBlocks = resources
      .split('  tour: {')
      .slice(1)
      .map((block) => block.split('  app: {')[0]);
    expect(tourBlocks).toHaveLength(3);
    for (const steps of Object.values(featureTourSteps)) {
      for (const step of steps) {
        for (const name of [step.title, step.body]) {
          const key = name.replace(/^tour\./, '');
          for (const block of tourBlocks) {
            expect(new RegExp(`\\b${key}:\\s*['"]`).test(block), name).toBe(true);
          }
        }
      }
    }
  });
});
