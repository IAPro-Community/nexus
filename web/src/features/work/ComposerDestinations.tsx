import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ClipboardCopy, Layers, Send } from 'lucide-react';
import { Button, Select } from '../../design-system';
import type { Agent, FlowSuitabilityAssessment, PromptArtifact } from '../../types';
import { asStringArray } from '../../lib/safeArray';
import { TaskPreparationDialog } from '../../components/TaskPreparationDialog';
import { askActionForStatus } from '../agents/askAgentModel';
import type { ComposerGate } from './composerModel';
import { ComposerPromptReview } from './ComposerPromptReview';
import styles from './ComposerSurface.module.scss';

/**
 * Destinations dock: copy, send-to-agent and turn-into-flow actions with the
 * flow-suitability hint, prompt review and the task preparation dialog wiring.
 */
export const ComposerDestinations: React.FC<{
  artifact: PromptArtifact | null;
  agents: Agent[];
  selectedAgentId: string;
  onSelectAgent: (id: string) => void;
  busy: boolean;
  canMaterialize: boolean;
  canExecute: boolean;
  destinationGate?: ComposerGate;
  flowSuitability?: FlowSuitabilityAssessment;
  promptReviewMissing: string[];
  promptReviewContradictions: string[];
  projectId: string;
  onCopy: () => void;
  onTransformFlow: () => void;
  onSendToAgent: (
    prompt: string,
    skills: string[],
    startIfNeeded: boolean,
    contextFingerprintId: string,
  ) => Promise<void>;
}> = ({
  artifact,
  agents,
  selectedAgentId,
  onSelectAgent,
  busy,
  canMaterialize,
  canExecute,
  destinationGate,
  flowSuitability,
  promptReviewMissing,
  promptReviewContradictions,
  projectId,
  onCopy,
  onTransformFlow,
  onSendToAgent,
}) => {
  const { t } = useTranslation();
  const [preparationOpen, setPreparationOpen] = useState(false);
  const selectedAgent = agents.find((agent) => agent.id === selectedAgentId);
  const startAgentIfNeeded = !askActionForStatus(selectedAgent?.status || 'STOPPED').startIfNeeded;

  return (
    <div className={styles.destinationDock}>
      <div className={styles.destinationDockInner}>
        <small className={styles.destinationLabel}>
          {t('work.composer.destinations', 'Destinations')}
        </small>
        {flowSuitability?.result && (
          <div className={styles.suitabilityHint}>
            <small className={styles.suitabilityTitle}>
              {t('work.composer.flowSuitability', 'Flow suitability')}:{' '}
              {t(`work.composer.flowResult.${flowSuitability.result}`, flowSuitability.result)}
            </small>
            {asStringArray(flowSuitability.reasons)[0] && (
              <p className={styles.suitabilityReason}>
                {asStringArray(flowSuitability.reasons)[0]}
              </p>
            )}
          </div>
        )}
        <ComposerPromptReview
          missing={promptReviewMissing}
          contradictions={promptReviewContradictions}
        />
        <div className={styles.destinationActions}>
          <Button onClick={onCopy} disabled={!artifact}>
            <ClipboardCopy size={14} /> {t('work.composer.copy', 'Copy')}
          </Button>
          {agents.length > 0 && (
            <>
              <Select
                value={selectedAgentId}
                onChange={onSelectAgent}
                options={agents.map((agent) => ({ value: agent.id, label: agent.name }))}
                selectClassName={styles.agentSelect}
              />
              <Button
                tone="brand"
                disabled={busy || !selectedAgentId || !canExecute || !artifact}
                onClick={() => setPreparationOpen(true)}
              >
                <Send size={14} /> {t('work.composer.sendToAgent', 'Send to agent')}
              </Button>
            </>
          )}
          <Button
            tone="brand"
            disabled={!canMaterialize || !artifact}
            title={
              !canMaterialize && destinationGate
                ? t(destinationGate.reasonKey, destinationGate.reason)
                : undefined
            }
            onClick={onTransformFlow}
          >
            <Layers size={14} /> {t('work.composer.transformFlow', 'Turn into Flow')}
          </Button>
        </div>
        {destinationGate && artifact && (!canMaterialize || !canExecute) && (
          <small className={styles.mutedCopy}>
            {t('work.copyAlwaysAvailable', 'Copy remains available.')}{' '}
            {t(destinationGate.reasonKey, destinationGate.reason)}
          </small>
        )}
      </div>

      <TaskPreparationDialog
        open={preparationOpen}
        agentId={selectedAgentId}
        agentName={selectedAgent?.name}
        projectId={projectId}
        initialPrompt={artifact?.content || ''}
        startIfNeeded={startAgentIfNeeded}
        onClose={() => setPreparationOpen(false)}
        onSubmit={async (prompt, skills, startIfNeeded, contextFingerprintId) => {
          await onSendToAgent(prompt, skills, startIfNeeded, contextFingerprintId || '');
        }}
      />
    </div>
  );
};
