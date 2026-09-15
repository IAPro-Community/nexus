import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { PanelRight, Sparkles } from 'lucide-react';
import { Button } from '../../design-system';
import type { PromptArtifact, Project } from '../../types';
import { nexus } from '../../nexus/api';
import { composerReferencedFiles } from './composerModel';
import type { ComposerGate } from './composerModel';
import { asStringArray } from '../../lib/safeArray';
import { ComposerStart } from './ComposerStart';
import { ComposerSessionHeader } from './ComposerSessionHeader';
import { ComposerBriefRail } from './ComposerBriefRail';
import { ComposerSkills } from './ComposerSkills';
import { ComposerPromptPreview } from './ComposerPromptPreview';
import { ComposerDestinations } from './ComposerDestinations';
import { ClarificationPrompt } from './ClarificationPrompt';
import { ComposerContextChips } from './ComposerContextChips';
import { useComposerSession } from './useComposerSession';
import {
  AgentActivity,
  ConversationSurface,
  useNexusAttachments,
  useNexusConversationRuntime,
  type NexusConversationEvent,
} from '../conversation';
import styles from './ComposerSurface.module.scss';

/**
 * Composer orchestrator — conversation is the center; domain rails live in the inspector.
 */
export const ComposerSurface: React.FC<{
  project: Project;
  onTransformFlow: (artifact: PromptArtifact) => void;
  destinationGate?: ComposerGate;
}> = ({ project, onTransformFlow, destinationGate }) => {
  const { t } = useTranslation();
  const composer = useComposerSession(project);
  const [inspectorOpen, setInspectorOpen] = useState(true);
  const {
    sessions,
    view,
    artifact,
    error,
    busy,
    confirmGaps,
    refineText,
    showRefineInput,
    agents,
    selectedAgentId,
    briefItems,
    nextUnknown,
    actionableGaps,
    sessionOptions,
    conversationEvents,
    bootstrapActivity,
  } = composer;

  const attachments = useNexusAttachments({
    sessionId: view?.session.id || 'pending',
    onInvalid: () => undefined,
  });

  const eventsWithBootstrap: NexusConversationEvent[] = useMemo(() => {
    if (!bootstrapActivity || view) return conversationEvents;
    return [
      {
        id: 'bootstrap-activity',
        sessionId: 'bootstrap',
        sequence: 0,
        kind: 'activity',
        payload: bootstrapActivity as unknown as Record<string, unknown>,
        createdAt: new Date().toISOString(),
        status: 'running',
      },
    ];
  }, [bootstrapActivity, conversationEvents, view]);

  const runtime = useNexusConversationRuntime({
    events: view ? conversationEvents : eventsWithBootstrap,
    isRunning: busy,
    isDisabled: view?.session.state === 'FINALIZED',
    onSend: async ({ text }) => {
      const files = attachments.takeFiles();
      const uploaded: Array<{
        id: string;
        name?: string;
        mimeType?: string;
        url?: string;
        size?: number;
      }> = [];
      if (view && files.length) {
        for (const file of files) {
          const created = await nexus.uploadComposerAttachment(view.session.id, file);
          uploaded.push({
            id: created.id,
            name: created.name,
            mimeType: created.mime_type,
            url: `/api/v1/attachments/${created.id}`,
            size: created.size,
          });
        }
      }
      await composer.sendText(text, uploaded.length ? uploaded : undefined);
      attachments.clear();
    },
    onCancel: composer.cancelCreate,
  });

  const canMaterialize = destinationGate?.canMaterialize ?? true;
  const canExecute = destinationGate?.canExecute ?? true;

  if (!view) {
    return (
      <div className={styles.deliberative}>
        <ComposerStart
          sessions={sessions}
          busy={busy}
          error={error}
          onCreate={(args) => void composer.create(args)}
          onResumeSession={(id) => void composer.loadSession(id)}
        />
        {bootstrapActivity ? (
          <div className={styles.bootstrapActivity}>
            <AgentActivity activity={bootstrapActivity} />
            <Button size="sm" onClick={composer.cancelCreate}>
              {t('conversation.stop', 'Stop')}
            </Button>
          </div>
        ) : null}
      </div>
    );
  }

  const turns = view.turns || [];
  const readinessState = view.brief.readiness?.state || 'UNKNOWN';
  const readyEnough =
    readinessState !== 'BLOCKED' &&
    (readinessState === 'READY' || readinessState === 'READY_WITH_ASSUMPTIONS');
  const showDestinationDock =
    turns.length > 0 || Boolean(artifact) || view.session.state === 'FINALIZED' || readyEnough;
  const isFinalized = view.session.state === 'FINALIZED';

  const inspector = (
    <>
      <ComposerBriefRail
        view={view}
        briefItems={briefItems}
        nextQuestion={nextUnknown?.question || ''}
        actionableGaps={actionableGaps}
      />
      <ComposerSkills
        skills={view.skills || []}
        busy={busy}
        onUpdateSkill={(skillId, state) => void composer.updateSkill(skillId, state)}
      />
      <ComposerPromptPreview
        preview={view.brief.preview}
        artifact={artifact}
        busy={busy}
        showRefineInput={showRefineInput}
        refineText={refineText}
        onToggleRefine={() => composer.setShowRefineInput(!showRefineInput)}
        onRefineTextChange={composer.setRefineText}
        onCancelRefine={() => composer.setShowRefineInput(false)}
        onRefine={() => void composer.refine()}
      />
      {showDestinationDock ? (
        <ComposerDestinations
          artifact={artifact}
          agents={agents}
          selectedAgentId={selectedAgentId}
          onSelectAgent={composer.setSelectedAgentId}
          busy={busy}
          canMaterialize={canMaterialize}
          canExecute={canExecute}
          destinationGate={destinationGate}
          flowSuitability={view.flow_suitability}
          promptReviewMissing={asStringArray(view.prompt_review?.missing)}
          promptReviewContradictions={asStringArray(view.prompt_review?.contradictions)}
          projectId={project.id}
          onCopy={() => void composer.copy()}
          onTransformFlow={() => {
            if (artifact) onTransformFlow(artifact);
          }}
          onSendToAgent={composer.sendToAgent}
        />
      ) : null}
    </>
  );

  return (
    <div className={styles.conversationShell}>
      <ComposerSessionHeader
        view={view}
        sessionOptions={sessionOptions}
        showSwitcher={sessions.length > 1}
        busy={busy}
        onSelectSession={(id) => void composer.loadSession(id)}
        onNewSession={composer.resetSession}
      />
      <ConversationSurface
        runtime={runtime}
        inspector={inspector}
        inspectorOpen={inspectorOpen}
        onToggleInspector={() => setInspectorOpen((v) => !v)}
        pendingAttachments={attachments.pending}
        onAddFiles={attachments.addFiles}
        onRemoveAttachment={attachments.remove}
        error={error}
        emptyLabel={t(
          'work.composer.emptyTurns',
          'Describe context, desired outcome and constraints. Composer keeps this elaboration.',
        )}
        header={
          <div className={styles.conversationChrome}>
            <Button size="sm" onClick={() => setInspectorOpen((v) => !v)}>
              <PanelRight size={14} />{' '}
              {inspectorOpen
                ? t('conversation.hideDetails', 'Hide details')
                : t('conversation.showDetails', 'Show details')}
            </Button>
            {!artifact && !isFinalized ? (
              <Button
                tone="brand"
                size="sm"
                disabled={busy}
                onClick={() => void composer.finalize()}
              >
                <Sparkles size={14} />{' '}
                {busy
                  ? t('work.composer.finalizing', 'Finalizing…')
                  : t('work.composer.finalize', 'Finish elaboration')}
              </Button>
            ) : null}
            {confirmGaps ? (
              <Button
                tone="warning"
                size="sm"
                disabled={busy}
                onClick={() => void composer.finalize(true)}
              >
                {t('work.composer.finalizeWithGaps', 'Finish with confirmed gaps')}
              </Button>
            ) : null}
          </div>
        }
        footer={
          <div className={styles.conversationFooter}>
            {nextUnknown && !isFinalized ? (
              <ClarificationPrompt
                item={{
                  id: nextUnknown.id,
                  question: nextUnknown.question,
                  rationale: nextUnknown.rationale,
                  severity: nextUnknown.severity,
                  suggested_options: nextUnknown.suggested_options,
                  default_choice: nextUnknown.default_choice,
                }}
                busy={busy}
                allowDismiss={nextUnknown.severity !== 'BLOCKING'}
                showRemember
                onAnswer={(answer) =>
                  void composer.resolveUnknown(nextUnknown.id, 'ANSWERED', answer)
                }
                onDismiss={() => void composer.resolveUnknown(nextUnknown.id, 'DISMISSED', '')}
                onRemember={(answer) => void composer.rememberUnknown(nextUnknown, answer)}
              />
            ) : null}
            {!isFinalized ? (
              <ComposerContextChips
                projectPath={project.canonical_path}
                files={composerReferencedFiles(view)}
                busy={busy}
                onAttach={(path) => void composer.attachContext(path)}
                onDetach={(path) => void composer.detachContext(path)}
              />
            ) : null}
          </div>
        }
      />
    </div>
  );
};
