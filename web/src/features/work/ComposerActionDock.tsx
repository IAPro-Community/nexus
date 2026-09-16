import React from 'react';
import { useTranslation } from 'react-i18next';
import { Sparkles } from 'lucide-react';
import { Button } from '../../design-system';
import type { Agent, FlowSuitabilityAssessment, PromptArtifact } from '../../types';
import type { ComposerActionPhase, ComposerGate } from './composerModel';
import { ComposerDestinations } from './ComposerDestinations';
import { ComposerPendingDecision } from './ComposerPendingDecision';
import styles from './ComposerInspector.module.scss';

export const ComposerActionDock: React.FC<{
  phase: ComposerActionPhase;
  pendingCount: number;
  busy: boolean;
  confirmGaps: boolean;
  artifact: PromptArtifact | null;
  agents: Agent[];
  selectedAgentId: string;
  onSelectAgent: (id: string) => void;
  canMaterialize: boolean;
  canExecute: boolean;
  destinationGate?: ComposerGate;
  flowSuitability?: FlowSuitabilityAssessment;
  promptReviewMissing: string[];
  promptReviewContradictions: string[];
  projectId: string;
  contextRefreshing?: boolean;
  onRefreshContext?: () => void;
  onCopy: () => void;
  onTransformFlow: () => void;
  onSendToAgent: (
    prompt: string,
    skills: string[],
    startIfNeeded: boolean,
    contextFingerprintId: string,
  ) => Promise<void>;
  onReviewFinalize: () => void;
  onFinalizeWithGaps: () => void;
  onSeeGaps: () => void;
  onFocusPending?: () => void;
}> = ({
  phase,
  pendingCount,
  busy,
  confirmGaps,
  artifact,
  agents,
  selectedAgentId,
  onSelectAgent,
  canMaterialize,
  canExecute,
  destinationGate,
  flowSuitability,
  promptReviewMissing,
  promptReviewContradictions,
  projectId,
  contextRefreshing,
  onRefreshContext,
  onCopy,
  onTransformFlow,
  onSendToAgent,
  onReviewFinalize,
  onFinalizeWithGaps,
  onSeeGaps,
  onFocusPending,
}) => {
  const { t } = useTranslation();
  const ready = phase === 'ready_review' || phase === 'finalized';

  return (
    <div className={styles.dock}>
      <ComposerPendingDecision count={pendingCount} onFocus={onFocusPending} />
      {phase === 'elaborating' ? (
        <div className={styles.dockActions}>
          {pendingCount > 0 ? (
            <Button size="sm" onClick={onSeeGaps}>
              {t('work.composer.seeGaps', 'See gaps')}
            </Button>
          ) : null}
          <Button size="sm" tone="brand" disabled={busy} onClick={onReviewFinalize}>
            <Sparkles size={14} />
            {busy
              ? t('work.composer.finalizing', 'Finalizing…')
              : t('work.composer.reviewAndFinish', 'Review and finalize')}
          </Button>
          {confirmGaps ? (
            <Button size="sm" tone="warning" disabled={busy} onClick={onFinalizeWithGaps}>
              {t('work.composer.finalizeWithGaps', 'Finish with confirmed gaps')}
            </Button>
          ) : null}
        </div>
      ) : null}
      {ready ? (
        <>
          {phase === 'ready_review' && !artifact ? (
            <div className={styles.dockActions}>
              <p className={styles.readyHint}>
                {t('work.composer.readyForReview', 'Ready for review')}
              </p>
              <Button size="sm" tone="brand" disabled={busy} onClick={onReviewFinalize}>
                <Sparkles size={14} /> {t('work.composer.reviewAndFinish', 'Review and finalize')}
              </Button>
            </div>
          ) : null}
          <ComposerDestinations
            artifact={artifact}
            agents={agents}
            selectedAgentId={selectedAgentId}
            onSelectAgent={onSelectAgent}
            busy={busy}
            canMaterialize={canMaterialize}
            canExecute={canExecute}
            destinationGate={destinationGate}
            flowSuitability={flowSuitability}
            promptReviewMissing={promptReviewMissing}
            promptReviewContradictions={promptReviewContradictions}
            projectId={projectId}
            contextRefreshing={contextRefreshing}
            onRefreshContext={onRefreshContext}
            onCopy={onCopy}
            onTransformFlow={onTransformFlow}
            onSendToAgent={onSendToAgent}
          />
        </>
      ) : null}
    </div>
  );
};
