import React from 'react';
import { useTranslation } from 'react-i18next';
import { Sparkles } from 'lucide-react';
import { Button } from '../../design-system';
import type { PromptArtifact, Project } from '../../types';
import { composerReferencedFiles } from './composerModel';
import type { ComposerGate } from './composerModel';
import { asStringArray } from '../../lib/safeArray';
import { ComposerStart } from './ComposerStart';
import { ComposerSessionHeader } from './ComposerSessionHeader';
import { ComposerConversation } from './ComposerConversation';
import { ComposerBriefRail } from './ComposerBriefRail';
import { ComposerSkills } from './ComposerSkills';
import { ComposerPromptPreview } from './ComposerPromptPreview';
import { ComposerDestinations } from './ComposerDestinations';
import { useComposerSession } from './useComposerSession';
import styles from './ComposerSurface.module.scss';

/**
 * Composer orchestrator. All session/API state and handlers live in
 * useComposerSession; this component only wires those into the extracted
 * presentational surfaces (Start / Header / Conversation / BriefRail / Skills /
 * PromptPreview / Destinations).
 */
export const ComposerSurface: React.FC<{
  project: Project;
  onTransformFlow: (artifact: PromptArtifact) => void;
  destinationGate?: ComposerGate;
}> = ({ project, onTransformFlow, destinationGate }) => {
  const { t } = useTranslation();
  const composer = useComposerSession(project);
  const {
    sessions,
    view,
    artifact,
    error,
    busy,
    confirmGaps,
    message,
    refineText,
    showRefineInput,
    agents,
    selectedAgentId,
    briefItems,
    nextUnknown,
    actionableGaps,
    sessionOptions,
  } = composer;

  const canMaterialize = destinationGate?.canMaterialize ?? true;
  const canExecute = destinationGate?.canExecute ?? true;

  if (!view) {
    return (
      <ComposerStart
        sessions={sessions}
        busy={busy}
        error={error}
        onCreate={(args) => void composer.create(args)}
        onResumeSession={(id) => void composer.loadSession(id)}
      />
    );
  }

  const turns = view.turns || [];
  const readinessState = view.brief.readiness?.state || 'UNKNOWN';
  const readyEnough =
    readinessState !== 'BLOCKED' &&
    (readinessState === 'READY' || readinessState === 'READY_WITH_ASSUMPTIONS');
  const showDestinationDock =
    turns.length > 0 || Boolean(artifact) || view.session.state === 'FINALIZED' || readyEnough;

  return (
    <div className={styles.deliberative}>
      <ComposerSessionHeader
        view={view}
        sessionOptions={sessionOptions}
        showSwitcher={sessions.length > 1}
        busy={busy}
        onSelectSession={(id) => void composer.loadSession(id)}
        onNewSession={composer.resetSession}
      />

      <div className={styles.deliberativeGrid}>
        <ComposerConversation
          view={view}
          nextUnknown={nextUnknown}
          busy={busy}
          message={message}
          onMessageChange={composer.setMessage}
          onSend={() => void composer.send()}
          onResolveUnknown={(unknownId, status, answer) =>
            void composer.resolveUnknown(unknownId, status, answer)
          }
          onRemember={(unknown, answer) => void composer.rememberUnknown(unknown, answer)}
          onDismiss={(unknownId) => void composer.resolveUnknown(unknownId, 'DISMISSED', '')}
          projectPath={project.canonical_path}
          contextFiles={composerReferencedFiles(view)}
          onAttachContext={(path) => void composer.attachContext(path)}
          onDetachContext={(path) => void composer.detachContext(path)}
        />
        <ComposerBriefRail
          view={view}
          briefItems={briefItems}
          nextQuestion={nextUnknown?.question || ''}
          actionableGaps={actionableGaps}
        />
      </div>

      <ComposerSkills
        skills={view.skills || []}
        busy={busy}
        onUpdateSkill={(skillId, state) => void composer.updateSkill(skillId, state)}
      />

      {error && <div className={styles.inlineError}>{error}</div>}

      {!artifact && view.session.state !== 'FINALIZED' && (
        <div className={styles.headerActions}>
          <Button tone="brand" disabled={busy} onClick={() => void composer.finalize()}>
            <Sparkles size={14} />{' '}
            {busy
              ? t('work.composer.finalizing', 'Finalizing…')
              : t('work.composer.finalize', 'Finish elaboration')}
          </Button>
          {confirmGaps && (
            <Button tone="warning" disabled={busy} onClick={() => void composer.finalize(true)}>
              {t('work.composer.finalizeWithGaps', 'Finish with confirmed gaps')}
            </Button>
          )}
        </div>
      )}

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

      {showDestinationDock && (
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
      )}
    </div>
  );
};
