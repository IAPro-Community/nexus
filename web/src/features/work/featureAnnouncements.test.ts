import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  announcementsForSurface, announcementStorageKey, announcementTourIndex,
  featureAnnouncements, isAnnouncementUnread,
} from './featureAnnouncements';
import { featureTourSteps } from './featureTour';

const folder = dirname(fileURLToPath(import.meta.url));
const sourcePaths = [
  'ComposerStart.tsx', 'ComposerSurface.tsx', 'ComposerInspector.tsx',
  'ComposerReadOnlyHistory.tsx', 'FlowRunsHistorySurface.tsx',
  'FlowDraftEditor.tsx', 'FlowRunSurface.tsx',
];
const uiSources = sourcePaths.map((path) => readFileSync(join(folder, path), 'utf8')).join('\n');
const localeSource = readFileSync(join(folder, '..', '..', 'i18n', 'resources.ts'), 'utf8');

describe('release-aware contextual feature discovery', () => {
  it('has stable and unique revisioned feature identities', () => {
    expect(featureAnnouncements.length).toBeGreaterThan(0);
    expect(new Set(featureAnnouncements.map((item) => item.id)).size).toBe(featureAnnouncements.length);
    for (const item of featureAnnouncements) {
      expect(item.id).toMatch(/^[a-z0-9-]+$/);
      expect(Number.isSafeInteger(item.revision) && item.revision > 0).toBe(true);
      expect(item.surfaces.length).toBeGreaterThan(0);
      expect(item.title).toMatch(/^discovery\./);
      expect(item.summary).toMatch(/^discovery\./);
      expect(item.target).toMatch(/^\[data-tour="[a-z0-9-]+"\]$/);
    }
  });

  it('does not announce controls absent from a surface', () => {
    const items = announcementsForSurface('flow-editor', (selector) =>
      selector === '[data-tour="flow-canvas"]');
    expect(items.every((item) => item.target === '[data-tour="flow-canvas"]')).toBe(true);
    expect(announcementsForSurface('composer-start', () => false)).toHaveLength(0);
  });

  it('only targets existing controls and matching guided steps', () => {
    for (const item of featureAnnouncements) {
      const selector = item.target.match(/data-tour="([^"]+)"/)?.[1] || '';
      expect(uiSources.includes(`data-tour="${selector}"`), item.id).toBe(true);
      for (const surface of item.surfaces) {
        expect(featureTourSteps[surface].some((step) =>
          step.target === item.target && step.id === item.tourStep), item.id).toBe(true);
      }
    }
  });

  it('localizes every notice in the three supported languages', () => {
    const translations = localeSource.split('  discovery: {').slice(1)
      .map((block) => block.split('  tour: {')[0]);
    expect(translations).toHaveLength(3);
    for (const item of featureAnnouncements) {
      for (const key of [item.title, item.summary]) {
        const token = key.replace('discovery.', '');
        for (const translation of translations) {
          expect(translation.includes(`    ${token}: '`), key).toBe(true);
        }
      }
    }
  });

  it('does not reannounce an acknowledged revision, but reannounces a newer one', () => {
    const item = featureAnnouncements[0];
    expect(isAnnouncementUnread(item, null)).toBe(true);
    expect(isAnnouncementUnread(item, String(item.revision))).toBe(false);
    expect(isAnnouncementUnread(item, String(item.revision + 1))).toBe(false);
    expect(isAnnouncementUnread({ ...item, revision: item.revision + 1 }, String(item.revision))).toBe(true);
    expect(isAnnouncementUnread(item, 'invalid')).toBe(true);
  });

  it('stores acknowledgements by feature identity, not by project', () => {
    const key = announcementStorageKey(featureAnnouncements[0].id);
    expect(key).not.toContain('project:');
    expect(key).not.toBe(announcementStorageKey(featureAnnouncements[1].id));
  });

  it('opens the relevant step rather than always starting at the beginning', () => {
    const item = featureAnnouncements.find((entry) => entry.id === 'composer-idea-workspace');
    expect(item).toBeDefined();
    const ids = featureTourSteps['composer-start'].map((step) => step.id);
    expect(announcementTourIndex(item!, ids)).toBe(1);
    expect(announcementTourIndex(item!, ['purpose'])).toBe(0);
  });
});
