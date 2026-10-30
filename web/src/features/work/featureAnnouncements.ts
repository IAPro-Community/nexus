import type { FeatureTourSurface } from './featureTourDefinitions';

/**
 * Product discovery catalog, bundled with the actual application version.
 *
 * A feature is announced only in a surface where its target exists. Entries
 * must be added in the same PR as the feature; there is no remote tracking
 * pixel, runtime release service or inference from git commit titles.
 *
 * Increment revision when a material change should be announced again.
 */
export interface FeatureAnnouncement {
  id: string;
  revision: number;
  surfaces: readonly FeatureTourSurface[];
  /** Route kind accepted by the shared Nexus shell. */
  openSurface: string;
  target: string;
  title: string;
  summary: string;
  /** Optional tour step id, within the current surface. */
  tourStep?: string;
}

export const featureAnnouncements: readonly FeatureAnnouncement[] = [
  {
    id: 'composer-idea-workspace',
    revision: 1,
    surfaces: ['composer-start'],
    openSurface: 'work',
    target: '[data-tour="composer-goal"]',
    title: 'discovery.composerWorkspaceTitle',
    summary: 'discovery.composerWorkspaceBody',
    tourStep: 'goal',
  },
  {
    id: 'composer-decision-inspector',
    revision: 1,
    surfaces: ['composer-session'],
    openSurface: 'work',
    target: '[data-tour="composer-inspector"]',
    title: 'discovery.composerInspectorTitle',
    summary: 'discovery.composerInspectorBody',
    tourStep: 'inspector',
  },
  {
    id: 'composer-offline-recovery',
    revision: 1,
    surfaces: ['composer-offline'],
    openSurface: 'work',
    target: '[data-tour="composer-offline-provider"]',
    title: 'discovery.composerOfflineTitle',
    summary: 'discovery.composerOfflineBody',
    tourStep: 'provider',
  },
  {
    id: 'flow-review-journey',
    revision: 1,
    surfaces: ['flow-library'],
    openSurface: 'missions',
    target: '[data-tour="flow-tracks"]',
    title: 'discovery.flowJourneyTitle',
    summary: 'discovery.flowJourneyBody',
    tourStep: 'tracks',
  },
  {
    id: 'flow-semantic-planner',
    revision: 1,
    surfaces: ['flow-editor'],
    openSurface: 'missions',
    target: '[data-tour="flow-goal"]',
    title: 'discovery.flowPromptTitle',
    summary: 'discovery.flowPromptBody',
    tourStep: 'goal',
  },
  {
    id: 'flow-pipeline-graph',
    revision: 1,
    surfaces: ['flow-editor'],
    openSurface: 'missions',
    target: '[data-tour="flow-canvas"]',
    title: 'discovery.flowGraphTitle',
    summary: 'discovery.flowGraphBody',
    tourStep: 'canvas',
  },
  {
    id: 'flow-mission-evidence',
    revision: 1,
    surfaces: ['flow-run'],
    openSurface: 'missions',
    target: '[data-tour="flow-run-status"]',
    title: 'discovery.flowEvidenceTitle',
    summary: 'discovery.flowEvidenceBody',
    tourStep: 'status',
  },
] as const;

export const announcementStorageKey = (id: string) => `iapro:nexus:feature-announcement:v1:${id}`;

export function announcementsForSurface(
  surface: FeatureTourSurface,
  isPresent: (selector: string) => boolean = () => true,
): FeatureAnnouncement[] {
  return featureAnnouncements.filter(
    (item) => item.surfaces.includes(surface) && isPresent(item.target),
  );
}

export function isAnnouncementUnread(
  item: FeatureAnnouncement,
  storedRevision: string | null,
): boolean {
  const previous = Number(storedRevision);
  return storedRevision === null || !Number.isSafeInteger(previous) || previous < item.revision;
}

export function announcementTourIndex(
  item: FeatureAnnouncement,
  stepIds: readonly string[],
): number {
  const result = stepIds.indexOf(item.tourStep || '');
  return result < 0 ? 0 : result;
}
