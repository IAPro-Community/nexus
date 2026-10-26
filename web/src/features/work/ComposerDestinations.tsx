import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ClipboardCopy, Layers, RefreshCw, Send } from 'lucide-react';
import { Button, Select } from '../../design-system';
import type { Agent, FlowSuitabilityAssessment, PromptArtifact } from '../../types';
import { asStringArray } from '../../lib/safeArray';
import { TaskPreparationDialog } from '../../components/TaskPreparationDialog';
import { askActionForStatus } from '../agents/askAgentModel';
import type { ComposerGate } from './composerModel';
import { ComposerPromptReview } from './ComposerPromptReview';
import styles from './ComposerInspector.module.scss';

/**
 * Destinations dock: copy, send-to-agent and turn-into-flow.
 * Layout is a narrow column — actions stack; the agent picker sits above Send.
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
  contextRefreshing = false,
  onRefreshContext,
  onCopy,
  onTransformFlow,
  onSendToAgent,
}) => {
  const { t } = useTranslation();
  const [preparationOpen, setPreparationOpen] = useState(false);
  // Recommend an execution path but never force it: both actions remain
  // available. Runtime admission, rather than this heuristic, authorizes GO.
  const flowPreferred = flowSuitability?.result === 'FLOW_BENEFICIAL' ||
    flowSuitability?.result === 'FLOW_STRONGLY_RECOMMENDED';
  const routeReason = asStringArray(flowSuitability?.reasons).slice(0, 2).join(' · ');
  const selectedAgent = agents.find((agent) => agent.id === selectedAgentId);
  const startAgentIfNeeded = !askActionForStatus(selectedAgent?.status || 'STOPPED').startIfNeeded;
  const blocked = Boolean(artifact && (!canMaterialize || !canExecute));
  const showRefresh =
    Boolean(onRefreshContext) &&
    destinationGate &&
    (destinationGate.action === 'REFRESH' ||
      destinationGate.action === 'PREPARE' ||
      destinationGate.action === 'RETRY');

  return (
    <div className={styles.destinations}>
      <small className={styles.destinationLabel}>
        {t('work.composer.destinations', 'Destinations')}
      </small>
      {flowSuitability?.result ? (
        <p className={styles.destinationHint} role="status">
          <strong>
            {flowPreferred
              ? t('work.composer.recommendFlow', 'Caminho recomendado: Flow coordenado')
              : t('work.composer.recommendAgent', 'Caminho recomendado: Agent direto')}
          </strong>
          {routeReason ? ` · ${routeReason}` : ''}
          {' · '}
          {t('work.composer.routeIsAdvice', 'Você pode escolher outro caminho. A execução depende de aprovação e validação.')}
        </p>
      ) : null}
      <ComposerPromptReview
        missing={promptReviewMissing}
        contradictions={promptReviewContradictions}
      />
      <div className={styles.destinationActions}>
        <Button onClick={onCopy} disabled={!artifact}>
          <ClipboardCopy size={14} /> {t('work.composer.copy', 'Copy')}
        </Button>
        {agents.length > 0 ? (
          <>
            <Select
              value={selectedAgentId}
              onChange={onSelectAgent}
              options={agents.map((agent) => ({ value: agent.id, label: agent.name }))}
              aria-label={t('work.composer.sendToAgent', 'Send to agent')}
            />
            <Button
              tone={flowPreferred ? 'default' : 'brand'}
              disabled={busy || !selectedAgentId || !canExecute || !artifact}
              onClick={() => setPreparationOpen(true)}
            >
              <Send size={14} /> {t('work.composer.sendToAgent', 'Send to agent')}
            </Button>
          </>
        ) : null}
        <Button
          tone={flowPreferred ? 'brand' : 'default'}
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
      {showRefresh ? (
        <Button size="sm" disabled={contextRefreshing} onClick={() => onRefreshContext?.()}>
          <RefreshCw size={13} />{' '}
          {contextRefreshing
            ? t('work.checking', 'Checking…')
            : destinationGate?.action === 'PREPARE'
              ? t('work.createBaseContext', 'Create base context')
              : t('work.refreshContext', 'Refresh context')}
        </Button>
      ) : null}
      {destinationGate && destinationGate.action !== 'NONE' ? (
        <p className={styles.destinationHint}>
          {blocked ? `${t('work.copyAlwaysAvailable', 'Copy remains available.')} ` : ''}
          {t(destinationGate.reasonKey, destinationGate.reason)}
        </p>
      ) : null}

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
