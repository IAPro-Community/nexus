import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { en, es, ptBR } from '../../i18n/resources';
import { featureTourSteps } from './featureTourDefinitions';

// Prevent documentation-only tours: all targets must correspond to actual
// UI anchors in the same release source. Browser tests still verify visibility.
const folder = dirname(fileURLToPath(import.meta.url));
const sourceFiles = [
  'ComposerStart.tsx',
  'ComposerWorkspaceHeader.tsx',
  'ComposerSurface.tsx',
  'ComposerInspector.tsx',
  'DirectSessionLauncher.tsx',
  'ComposerReadOnlyHistory.tsx',
  'FlowRunsHistorySurface.tsx',
  'FlowDraftEditor.tsx',
  'FlowRunSurface.tsx',
  'ComparisonLabPanel.tsx',
  '../usage/UsageSurface.tsx',
  '../../nexus/AgentTerminal.tsx',
  '../../components/AttentionNotificationCard.tsx',
  '../browser/BrowserSurface.tsx',
  '../agents/AgentsSurface.tsx',
  '../../notifications/FocusQueueSection.tsx',
  '../projects/ProjectManagerSurface.tsx',
  '../settings/RemoteAccessTab.tsx',
  '../settings/NotificationCategorySettings.tsx',
  '../../workspace/WorkspaceSnapshotsControl.tsx',
];
const source = sourceFiles.map((file) => readFileSync(join(folder, file), 'utf8')).join('\n');

describe('Feature tour implementation parity', () => {
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
    const tourBlocks: Record<string, string>[] = [en.tour, es.tour, ptBR.tour];
    expect(tourBlocks).toHaveLength(3);
    for (const steps of Object.values(featureTourSteps)) {
      for (const step of steps) {
        for (const name of [step.title, step.body]) {
          const key = name.replace(/^tour\./, '');
          for (const block of tourBlocks) {
            expect(typeof block[key], name).toBe('string');
            expect(block[key].trim().length, name).toBeGreaterThan(0);
          }
        }
      }
    }
  });
});
