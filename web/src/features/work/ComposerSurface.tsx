import React from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Sparkles } from 'lucide-react';
import { Badge, Button, Card, Select } from '../../design-system';
import type { ContextFileRef, Project, PromptArtifact } from '../../types';
import { composerSessionStateKey, formatComposerSessionTitle } from './composerModel';
import type { ComposerGate } from './composerModel';
import { asArray, asStringArray } from '../../lib/safeArray';
import { ComposerStart } from './ComposerStart';
import { ComposerConversation } from './ComposerConversation';
import { ComposerContextChips } from './ComposerContextChips';
import { ComposerBriefRail } from './ComposerBriefRail';
import { ComposerPromptPreview } from './ComposerPromptPreview';
import { ComposerDestinations } from './ComposerDestinations';
import { useComposerSession } from './useComposerSession';
import styles from './ComposerSurface.module.scss';

/**
 * Thin orchestrator for the prompt Composer. State, API orchestration and the
 * derived view models live in useComposerSession; this component only wires the
 * decomposed subcomponents together.
 */
export const ComposerSurface: React.FC<{
  project: Project;
  onTransformFlow: (artifact: PromptArtifact) => void;
  destinationGate?: ComposerGate;
}> = ({ project, onTransformFlow, destinationGate }) => {
  const { t } = useTranslation();
  const c = useComposerSession({ project, destinationGate });

  if (!c.view) {
    return (
      <ComposerStart
        sessions={c.sessions}
        busy={c.busy}
        error={c.error}
        onCreate={(args) => void c.create(args)}
        onResumeSession={(id) => void c.loadSession(id)}
      />
    );
  }

  const view = c.view;
  const archetype = view.brief.intent?.archetype;
  const archetypeLabel = archetype ? t(`work.composer.archetypes.${archetype}`, archetype) : null;
  const turns = view.turns || [];
  const readinessState = view.brief.readiness?.state || 'UNKNOWN';
  const readyEnough =
    readinessState !== 'BLOCKED' &&
    (readinessState === 'READY' || readinessState === 'READY_WITH_ASSUMPTIONS');
  const showDestinationDock =
    turns.length > 0 || Boolean(c.artifact) || view.session.state === 'FINALIZED' || readyEnough;
  const promptReviewMissing = asStringArray(view.prompt_review?.missing);
  const promptReviewContradictions = asStringArray(view.prompt_review?.contradictions);
  const referencedFiles =
    view.brief.context &&
    typeof view.brief.context === 'object' &&
    !Array.isArray(view.brief.context)
      ? asArray<ContextFileRef>(
          (view.brief.context as { referenced_files?: ContextFileRef[] }).referenced_files,
        )
      : [];

  return (
    <div className={styles.deliberative}>
      <div className={styles.deliberativeHeader}>
        <div>
          <h2 className={styles.deliberativeTitle}>
            {formatComposerSessionTitle(
              view.session.title,
              view.session.id,
              view.session.created_at,
            )}
          </h2>
        </div>
        <div className={styles.flexRowGap8Center}>
          {archetypeLabel && (
            <Badge tone="default">
              {t('work.composer.archetype', 'Archetype')}: {archetypeLabel}
            </Badge>
          )}
          {c.sessions.length > 1 && (
            <Select
              value={view.session.id}
              onChange={(id) => {
                if (id && id !== view.session.id) void c.loadSession(id);
              }}
              options={c.sessionOptions}
              selectClassName={styles.sessionSelect}
            />
          )}
          <Badge tone={view.session.state === 'FINALIZED' ? 'success' : 'brand'}>
            {t(composerSessionStateKey(view.session.state), view.session.state)}
          </Badge>
          <Button size="sm" disabled={c.busy} onClick={c.reset}>
            <Plus size={13} /> {t('work.composer.newSession', 'New')}
          </Button>
        </div>
      </div>

      <div className={styles.deliberativeGrid}>
        <ComposerConversation
          view={view}
          nextUnknown={c.nextUnknown}
          busy={c.busy}
          message={c.message}
          onMessageChange={c.setMessage}
          onSend={() => void c.send()}
          onResolveUnknown={(unknownId, status, answer) =>
            void c.resolveUnknown(unknownId, status, answer)
          }
          onRemember={(unknown, answer) => void c.rememberUnknown(unknown, answer)}
          onDismiss={(unknownId) => void c.resolveUnknown(unknownId, 'DISMISSED', '')}
        />

        <ComposerBriefRail
          view={view}
          briefItems={c.briefItems}
          nextQuestion={c.nextQuestion}
          actionableGaps={c.actionableGaps}
        />
      </div>

      <ComposerContextChips
        projectPath={project.canonical_path}
        files={referencedFiles}
        busy={c.busy}
        onAttach={(path) => void c.attachContext(path)}
        onDetach={(path) => void c.detachContext(path)}
      />

      {(view.skills || []).length > 0 && (
        <Card className={styles.artifactCard}>
          <strong>{t('work.composer.maestroSkills', 'Maestro skills')}</strong>
          {(view.skills || []).map((skill) => (
            <div key={skill.skill_id} className={styles.skillRow}>
              <div className={styles.skillInfo}>
                <strong>{skill.skill_id}</strong>
                <small className={styles.skillReason}>{skill.reason || skill.applicability}</small>
              </div>
              <Badge tone={skill.state === 'UNAVAILABLE' ? 'danger' : 'default'}>
                {t(`work.composer.skillState.${skill.state}`, skill.state)}
              </Badge>
              {skill.state !== 'UNAVAILABLE' && skill.state !== 'APPLIED' && (
                <>
                  <Button
                    size="sm"
                    tone="brand"
                    disabled={c.busy}
                    onClick={() => void c.updateSkill(skill.skill_id, 'ACCEPTED')}
                  >
                    {t('work.composer.accept', 'Accept')}
                  </Button>
                  <Button
                    size="sm"
                    disabled={c.busy}
                    onClick={() => void c.updateSkill(skill.skill_id, 'REJECTED')}
                  >
                    {t('work.composer.dismiss', 'Dismiss')}
                  </Button>
                </>
              )}
            </div>
          ))}
        </Card>
      )}

      {c.error && <div className={styles.inlineError}>{c.error}</div>}

      {!c.artifact && view.session.state !== 'FINALIZED' && (
        <div className={styles.headerActions}>
          <Button tone="brand" disabled={c.busy} onClick={() => void c.finalize()}>
            <Sparkles size={14} />{' '}
            {c.busy
              ? t('work.composer.finalizing', 'Finalizing…')
              : t('work.composer.finalize', 'Finish elaboration')}
          </Button>
          {c.confirmGaps && (
            <Button tone="warning" disabled={c.busy} onClick={() => void c.finalize(true)}>
              {t('work.composer.finalizeWithGaps', 'Finish with confirmed gaps')}
            </Button>
          )}
        </div>
      )}

      <ComposerPromptPreview
        preview={view.brief.preview}
        artifact={c.artifact}
        busy={c.busy}
        showRefineInput={c.showRefineInput}
        refineText={c.refineText}
        onToggleRefine={() => c.setShowRefineInput((open) => !open)}
        onRefineTextChange={c.setRefineText}
        onCancelRefine={() => c.setShowRefineInput(false)}
        onRefine={() => void c.refine()}
      />

      {showDestinationDock && (
        <ComposerDestinations
          artifact={c.artifact}
          agents={c.agents}
          selectedAgentId={c.selectedAgentId}
          onSelectAgent={c.setSelectedAgentId}
          busy={c.busy}
          canMaterialize={c.canMaterialize}
          canExecute={c.canExecute}
          destinationGate={destinationGate}
          flowSuitability={view.flow_suitability}
          promptReviewMissing={promptReviewMissing}
          promptReviewContradictions={promptReviewContradictions}
          projectId={project.id}
          onCopy={() => void c.copy()}
          onTransformFlow={() => {
            if (c.artifact) onTransformFlow(c.artifact);
          }}
          onSendToAgent={c.sendToAgent}
        />
      )}
    </div>
  );
};
