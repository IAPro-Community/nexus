import React, { useCallback, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { PanelRight } from 'lucide-react';
import { Button } from '../../design-system';
import type { PromptArtifact, Project, ContextReadinessState } from '../../types';
import { nexus } from '../../nexus/api';
import { composerActionPhase, composerReferencedFiles } from './composerModel';
import type { ComposerGate } from './composerModel';
import { ComposerStart } from './ComposerStart';
import { ComposerWorkspaceHeader } from './ComposerWorkspaceHeader';
import { ComposerInspector } from './ComposerInspector';
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
  contextState?: ContextReadinessState;
  intelligenceReady?: boolean;
  intelligenceLabel?: string;
  memoryCount?: number;
  contextRefreshing?: boolean;
  onRefreshContext?: () => void;
}> = ({
  project,
  onTransformFlow,
  destinationGate,
  contextState = 'MISSING',
  intelligenceReady = false,
  intelligenceLabel,
  memoryCount = 0,
  contextRefreshing = false,
  onRefreshContext,
}) => {
  const { t } = useTranslation();
  const composer = useComposerSession(project);
  const [inspectorOpen, setInspectorOpen] = useState(
    () => typeof window === 'undefined' || window.matchMedia('(min-width: 768px)').matches,
  );
  const [threadEndFocusRequest, setThreadEndFocusRequest] = useState(0);
  const [attachmentTransferRunning, setAttachmentTransferRunning] = useState(false);
  const attachmentAbort = useRef<AbortController | null>(null);
  const {
    sessions,
    view,
    artifact,
    error,
    busy,
    conversationRunning,
    confirmGaps,
    refineText,
    showRefineInput,
    agents,
    skillCatalog,
    selectedAgentId,
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

  const cancelActiveConversation = useCallback(() => {
    attachmentAbort.current?.abort();
    composer.cancelConversation();
  }, [composer]);

  const onSend = useCallback(
    async ({ text }: { text: string }) => {
      attachmentAbort.current?.abort();
      const controller = new AbortController();
      attachmentAbort.current = controller;
      setAttachmentTransferRunning(true);
      try {
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
            const created = await nexus.uploadComposerAttachment(
              view.session.id,
              file,
              controller.signal,
            );
            uploaded.push({
              id: created.id,
              name: created.name,
              mimeType: created.mime_type,
              url: `/api/v1/attachments/${created.id}`,
              size: created.size,
            });
          }
        }
        if (controller.signal.aborted) return;
        await composer.sendText(text, uploaded.length ? uploaded : undefined);
        if (!controller.signal.aborted) attachments.clear();
      } finally {
        if (attachmentAbort.current === controller) attachmentAbort.current = null;
        setAttachmentTransferRunning(false);
      }
    },
    [attachments, composer, view],
  );

  const runtime = useNexusConversationRuntime({
    events: view ? conversationEvents : eventsWithBootstrap,
    isRunning: conversationRunning || attachmentTransferRunning,
    isDisabled: view?.session.state === 'FINALIZED',
    onSend,
    onCancel: cancelActiveConversation,
  });

  const canMaterialize = destinationGate?.canMaterialize ?? true;
  const canExecute = destinationGate?.canExecute ?? true;
  const intelLabel =
    intelligenceLabel ||
    (intelligenceReady ? t('work.intelligenceReady', 'READY') : t('work.intelligenceOff', 'OFF'));

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
            <Button size="sm" onClick={composer.cancelConversation}>
              {t('conversation.stop', 'Stop')}
            </Button>
          </div>
        ) : null}
      </div>
    );
  }

  const isFinalized = view.session.state === 'FINALIZED';
  const phase = composerActionPhase(view, Boolean(artifact));
  const pendingCount = actionableGaps.length;

  const inspector = (
    <ComposerInspector
      view={view}
      project={project}
      artifact={artifact}
      busy={busy}
      confirmGaps={confirmGaps}
      phase={phase}
      pendingCount={pendingCount}
      actionableGaps={actionableGaps}
      memoryCount={memoryCount}
      intelligenceLabel={intelLabel}
      agents={agents}
      selectedAgentId={selectedAgentId}
      onSelectAgent={composer.setSelectedAgentId}
      canMaterialize={canMaterialize}
      canExecute={canExecute}
      destinationGate={destinationGate}
      showRefineInput={showRefineInput}
      refineText={refineText}
      onToggleRefine={() => composer.setShowRefineInput(!showRefineInput)}
      onRefineTextChange={composer.setRefineText}
      onCancelRefine={() => composer.setShowRefineInput(false)}
      onRefine={() => void composer.refine()}
      skillCatalog={skillCatalog}
      onUpdateSkill={(skillId, state) => void composer.updateSkill(skillId, state)}
      onSearchSkillCatalog={composer.searchSkillCatalog}
      onSelectCatalogSkill={(skillId) => void composer.selectCatalogSkill(skillId)}
      onAttach={(path) => void composer.attachContext(path)}
      onDetach={(path) => void composer.detachContext(path)}
      onCopy={() => void composer.copy()}
      onTransformFlow={() => {
        if (artifact) onTransformFlow(artifact);
      }}
      onSendToAgent={composer.sendToAgent}
      onReviewFinalize={() => void composer.finalize()}
      onFinalizeWithGaps={() => void composer.finalize(true)}
      onFocusThread={() => {
        setInspectorOpen(false);
        setThreadEndFocusRequest((value) => value + 1);
      }}
      contextRefreshing={contextRefreshing}
      onRefreshContext={onRefreshContext}
    />
  );

  return (
    <div className={styles.conversationShell}>
      <ComposerWorkspaceHeader
        view={view}
        sessionOptions={sessionOptions}
        showSwitcher={sessions.length > 1}
        busy={busy}
        onSelectSession={(id) => void composer.loadSession(id)}
        onNewSession={composer.resetSession}
        contextState={contextState}
        intelligenceLabel={intelLabel}
        intelligenceReady={intelligenceReady}
        memoryCount={memoryCount}
      />
      <ConversationSurface
        runtime={runtime}
        inspector={inspector}
        inspectorOpen={inspectorOpen}
        onToggleInspector={() => setInspectorOpen((v) => !v)}
        hideInspectorChrome
        inspectorTitle={t('work.composer.inspector', 'Briefing')}
        pendingAttachments={attachments.pending}
        onAddFiles={attachments.addFiles}
        onRemoveAttachment={attachments.remove}
        error={error}
        emptyLabel={t(
          'work.composer.emptyTurns',
          'Describe context, desired outcome and constraints. Composer keeps this elaboration.',
        )}
        header={
          <div
            className={styles.conversationChrome}
            data-inspector={inspectorOpen ? 'open' : 'closed'}
          >
            <Button size="sm" onClick={() => setInspectorOpen((v) => !v)}>
              <PanelRight size={14} />{' '}
              {inspectorOpen
                ? t('conversation.hideDetails', 'Hide details')
                : t('conversation.showDetails', 'Show details')}
            </Button>
          </div>
        }
        threadEndFocusRequest={threadEndFocusRequest}
        threadEnd={
          nextUnknown && !isFinalized ? (
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
          ) : null
        }
        composerLeading={
          isFinalized ? null : (
            <ComposerContextChips
              variant="icon"
              projectPath={project.canonical_path}
              files={composerReferencedFiles(view)}
              busy={busy}
              onAttach={(path) => void composer.attachContext(path)}
              onDetach={(path) => void composer.detachContext(path)}
            />
          )
        }
      />
    </div>
  );
};
