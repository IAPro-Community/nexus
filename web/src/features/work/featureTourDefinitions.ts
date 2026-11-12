import type { TourStep } from '../../app/tour/tour';

/**
 * Tours describe existing controls only. Missing targets are skipped by the
 * shared product-tour component, including empty projects and offline AI.
 */
export type FeatureTourSurface =
  | 'composer-start'
  | 'composer-session'
  | 'composer-offline'
  | 'direct-session'
  | 'flow-library'
  | 'flow-editor'
  | 'flow-run'
  | 'resources'
  | 'agents'
  | 'agent-terminal'
  | 'browser-inspector'
  | 'notifications'
  | 'projects';

export const featureTourSteps: Record<FeatureTourSurface, TourStep[]> = {
  'composer-start': [
    {
      id: 'purpose',
      target: '[data-tour="composer-purpose"]',
      title: 'tour.composerPurposeTitle',
      body: 'tour.composerPurposeBody',
    },
    {
      id: 'goal',
      target: '[data-tour="composer-goal"]',
      title: 'tour.composerGoalTitle',
      body: 'tour.composerGoalBody',
    },
    {
      id: 'mode',
      target: '[data-tour="composer-mode"]',
      title: 'tour.composerModeTitle',
      body: 'tour.composerModeBody',
    },
    {
      id: 'depth',
      target: '[data-tour="composer-depth"]',
      title: 'tour.composerDepthTitle',
      body: 'tour.composerDepthBody',
    },
    {
      id: 'history',
      target: '[data-tour="composer-history"]',
      title: 'tour.composerHistoryTitle',
      body: 'tour.composerHistoryBody',
    },
  ],
  'composer-session': [
    {
      id: 'session',
      target: '[data-tour="composer-session"]',
      title: 'tour.composerSessionTitle',
      body: 'tour.composerSessionBody',
    },
    {
      id: 'conversation',
      target: '[data-tour="composer-conversation"]',
      title: 'tour.composerConversationTitle',
      body: 'tour.composerConversationBody',
    },
    {
      id: 'inspector',
      target: '[data-tour="composer-inspector"]',
      title: 'tour.composerInspectorTitle',
      body: 'tour.composerInspectorBody',
    },
  ],
  'composer-offline': [
    {
      id: 'provider',
      target: '[data-tour="composer-offline-provider"]',
      title: 'tour.composerOfflineTitle',
      body: 'tour.composerOfflineBody',
    },
    {
      id: 'history',
      target: '[data-tour="composer-offline-history"]',
      title: 'tour.composerHistoryTitle',
      body: 'tour.composerHistoryBody',
    },
  ],
  'direct-session': [
    {
      id: 'cancel',
      target: '[data-tour="direct-session-cancel"]',
      title: 'tour.directSessionCancelTitle',
      body: 'tour.directSessionCancelBody',
    },
  ],
  'flow-library': [
    {
      id: 'tracks',
      target: '[data-tour="flow-tracks"]',
      title: 'tour.flowTracksTitle',
      body: 'tour.flowTracksBody',
    },
    {
      id: 'new',
      target: '[data-tour="flow-new"]',
      title: 'tour.flowNewTitle',
      body: 'tour.flowNewBody',
    },
    {
      id: 'drafts',
      target: '[data-tour="flow-drafts"]',
      title: 'tour.flowDraftsTitle',
      body: 'tour.flowDraftsBody',
    },
    {
      id: 'comparison',
      target: '[data-tour="flow-comparison-entry"]',
      title: 'tour.flowComparisonTitle',
      body: 'tour.flowComparisonBody',
    },
  ],
  'flow-run': [
    {
      id: 'status',
      target: '[data-tour="flow-run-status"]',
      title: 'tour.flowRunStatusTitle',
      body: 'tour.flowRunStatusBody',
    },
    {
      id: 'evidence',
      target: '[data-tour="flow-run-steps"]',
      title: 'tour.flowRunStepsTitle',
      body: 'tour.flowRunStepsBody',
    },
    {
      id: 'actions',
      target: '[data-tour="flow-run-actions"]',
      title: 'tour.flowRunActionsTitle',
      body: 'tour.flowRunActionsBody',
    },
  ],
  'flow-editor': [
    {
      id: 'goal',
      target: '[data-tour="flow-goal"]',
      title: 'tour.flowGoalTitle',
      body: 'tour.flowGoalBody',
    },
    {
      id: 'handoff',
      target: '[data-tour="flow-handoff"]',
      title: 'tour.flowHandoffTitle',
      body: 'tour.flowHandoffBody',
    },
    {
      id: 'views',
      target: '[data-tour="flow-views"]',
      title: 'tour.flowViewsTitle',
      body: 'tour.flowViewsBody',
    },
    {
      id: 'canvas',
      target: '[data-tour="flow-canvas"]',
      title: 'tour.flowCanvasTitle',
      body: 'tour.flowCanvasBody',
    },
  ],
  resources: [
    {
      id: 'provider-discovery',
      target: '[data-tour="usage-provider-discovery"]',
      title: 'tour.providerDiscoveryTitle',
      body: 'tour.providerDiscoveryBody',
    },
  ],
  agents: [
    {
      id: 'groups',
      target: '[data-tour="agent-groups"]',
      title: 'tour.agentGroupsTitle',
      body: 'tour.agentGroupsBody',
    },
  ],
  'agent-terminal': [
    {
      id: 'attention-choice',
      target: '[data-tour="attention-structured-choice"]',
      title: 'tour.agentAttentionChoiceTitle',
      body: 'tour.agentAttentionChoiceBody',
    },
    {
      id: 'follow',
      target: '[data-tour="agent-terminal-follow"]',
      title: 'tour.agentTerminalFollowTitle',
      body: 'tour.agentTerminalFollowBody',
    },
  ],
  'browser-inspector': [
    {
      id: 'selected-element',
      target: '[data-tour="browser-inspector-entry"]',
      title: 'tour.browserInspectTitle',
      body: 'tour.browserInspectBody',
    },
    {
      id: 'visual-diff',
      target: '[data-tour="browser-visual-diff-tab"]',
      title: 'tour.browserVisualDiffTitle',
      body: 'tour.browserVisualDiffBody',
    },
  ],
  notifications: [
    {
      id: 'queue',
      target: '[data-tour="focus-queue"]',
      title: 'tour.focusQueueTitle',
      body: 'tour.focusQueueBody',
    },
  ],
  projects: [
    {
      id: 'workspace-sharing',
      target: '[data-tour="workspace-sharing-card"]',
      title: 'tour.workspaceSharingTitle',
      body: 'tour.workspaceSharingBody',
    },
  ],
};

export const featureTourStorageKey = (projectId: string, surface: FeatureTourSurface) =>
  `iapro:nexus:feature-tour:v1:${encodeURIComponent(projectId)}:${surface}`;

export function parseFeatureTourProgress(raw: string | null, max: number): number {
  const parsed = Number(raw);
  return raw !== null && Number.isSafeInteger(parsed) && parsed >= 0
    ? Math.min(parsed, Math.max(0, max - 1))
    : 0;
}
