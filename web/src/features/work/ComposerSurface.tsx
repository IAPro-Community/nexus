import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  ClipboardCopy,
  FileText,
  Layers,
  Lightbulb,
  MessageCircle,
  Plus,
  RefreshCw,
  Send,
  Sparkles,
} from 'lucide-react';
import { Badge, Button, Card, Input, Select } from '../../design-system';
import { nexus } from '../../nexus/api';
import type {
  ComposerSession,
  ComposerSessionView,
  PromptArtifact,
  PromptReadinessCheck,
  Agent,
  Project,
} from '../../types';

type BriefUnknown = NonNullable<ComposerSessionView['brief']['unknowns']>[number];
import { selectResumableComposerSession } from './composerSessionModel';
import {
  composerNeedsGapConfirmation,
  formatComposerSessionTitle,
  composerSessionStateKey,
} from './composerModel';
import type { ComposerGate } from './composerModel';
import { asArray, asStringArray } from '../../lib/safeArray';
import { TaskPreparationDialog } from '../../components/TaskPreparationDialog';
import { askActionForStatus } from '../agents/askAgentModel';
import { ClarificationPrompt } from './ClarificationPrompt';
import styles from './ComposerSurface.module.scss';

const ARCHETYPE_LABELS: Record<string, string> = {
  SOFTWARE_FEATURE: 'Feature',
  BUG_FIX: 'Correção',
  ARCHITECTURE: 'Arquitetura',
  DEVOPS: 'DevOps',
  RESEARCH: 'Pesquisa',
  SECURITY: 'Segurança',
  GENERIC: 'Genérico',
};

function isUnknownResolved(status: string): boolean {
  return status === 'ANSWERED' || status === 'CONFIRMED' || status === 'DISMISSED';
}

export const ComposerSurface: React.FC<{
  project: Project;
  onTransformFlow: (artifact: PromptArtifact) => void;
  destinationGate?: ComposerGate;
}> = ({ project, onTransformFlow, destinationGate }) => {
  const { t } = useTranslation();
  const [sessions, setSessions] = useState<ComposerSession[]>([]);
  const [view, setView] = useState<ComposerSessionView | null>(null);
  const [inputMode, setInputMode] = useState<'IDEA' | 'EXISTING_PROMPT'>('IDEA');
  const [draft, setDraft] = useState('');
  const [sourcePrompt, setSourcePrompt] = useState('');
  const [message, setMessage] = useState('');
  const [artifact, setArtifact] = useState<PromptArtifact | null>(null);
  const [error, setError] = useState('');
  const [confirmGaps, setConfirmGaps] = useState(false);
  const [busy, setBusy] = useState(false);
  const [refineText, setRefineText] = useState('');
  const [showRefineInput, setShowRefineInput] = useState(false);
  const [unknownAnswers, setUnknownAnswers] = useState<Record<string, string>>({});
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState('');
  const [preparationOpen, setPreparationOpen] = useState(false);
  const canMaterialize = destinationGate?.canMaterialize ?? true;
  const canExecute = destinationGate?.canExecute ?? true;

  const selectedSkillIds = useMemo(
    () =>
      (view?.skills || [])
        .filter((skill) => skill.state === 'ACCEPTED' || skill.state === 'APPLIED')
        .map((skill) => skill.skill_id),
    [view?.skills],
  );

  const refreshSessions = useCallback(async () => {
    const next = await nexus.listComposerSessions(project.id);
    setSessions(next || []);
    return next || [];
  }, [project.id]);
  useEffect(() => {
    void (async () => {
      try {
        const next = await refreshSessions();
        const id = selectResumableComposerSession(next);
        if (id) setView(await nexus.getComposerSession(id));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [project.id, refreshSessions]);

  useEffect(() => {
    void nexus
      .listAgents(project.id)
      .then((items) => {
        const next = items || [];
        setAgents(next);
        setSelectedAgentId((current) => current || next[0]?.id || '');
      })
      .catch(() => undefined);
  }, [project.id]);

  const briefItems = useMemo(() => {
    if (!view) return [];
    return [
      [t('work.composer.understood', 'Understood'), view.brief.goal],
      [
        t('work.composer.context', 'Context'),
        [
          ...asStringArray(view.brief.context),
          ...(view.brief.context &&
          typeof view.brief.context === 'object' &&
          !Array.isArray(view.brief.context)
            ? asStringArray((view.brief.context as { existing_state?: unknown }).existing_state)
            : []),
        ].join(' · '),
      ],
      [t('work.composer.decisions', 'Decisions'), asStringArray(view.brief.decisions).join(' · ')],
      [
        t('work.composer.criteria', 'Criteria'),
        asStringArray(view.brief.success_criteria).join(' · '),
      ],
      [
        t('work.composer.openDoubts', 'Open doubts'),
        asStringArray(view.brief.open_questions).join(' · '),
      ],
    ].filter(([, value]) => value);
  }, [t, view]);

  const nextUnknown = useMemo(() => {
    if (!view) return null;
    type UnknownItem = NonNullable<ComposerSessionView['brief']['unknowns']>[number];
    const unknowns = asArray<UnknownItem>(view.brief.unknowns);
    const unresolved = unknowns.filter((u) => !isUnknownResolved(u.status));
    const intelligenceBlocking = unresolved.find(
      (u) => u.severity === 'BLOCKING' && String(u.source || '').toUpperCase() === 'INTELLIGENCE',
    );
    if (intelligenceBlocking) return intelligenceBlocking;
    const blocking = unresolved.find((u) => u.severity === 'BLOCKING');
    const withOptions = unresolved.find((u) => asArray(u.suggested_options).length > 0);
    return blocking || withOptions || unresolved[0] || null;
  }, [view]);

  const nextQuestion = nextUnknown?.question || '';

  const actionableGaps = useMemo(() => {
    if (!view) return [];
    type UnknownItem = NonNullable<ComposerSessionView['brief']['unknowns']>[number];
    return asArray<UnknownItem>(view.brief.unknowns).filter((u) => !isUnknownResolved(u.status));
  }, [view]);

  const promptReadyEnough = useMemo(() => {
    if (!view) return false;
    const blockingOpen = actionableGaps.some((u) => u.severity === 'BLOCKING');
    return !blockingOpen && (Boolean(artifact) || turnsReady(view));
  }, [view, actionableGaps, artifact]);

  function turnsReady(sessionView: ComposerSessionView): boolean {
    const goal = String(sessionView.brief.goal || '').trim();
    if (goal.split(/\s+/).length >= 8) return true;
    const criteria = asStringArray(sessionView.brief.success_criteria);
    const decisions = asStringArray(sessionView.brief.decisions);
    return criteria.length > 0 || decisions.length > 0 || (sessionView.turns || []).length >= 2;
  }

  const sessionOptions = useMemo(
    () =>
      sessions.map((s) => ({
        value: s.id,
        label: `${formatComposerSessionTitle(s.title, s.id, s.created_at)} · ${t(
          composerSessionStateKey(s.state),
          s.state,
        )}`,
      })),
    [sessions, t],
  );

  const create = async () => {
    if (!draft.trim()) return;
    setBusy(true);
    setError('');
    try {
      const next = await nexus.createComposerSessionWithMode(
        project.id,
        draft.trim(),
        inputMode,
        inputMode === 'EXISTING_PROMPT' ? sourcePrompt.trim() : undefined,
      );
      setView(next);
      setDraft('');
      setSourcePrompt('');
      setArtifact(null);
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const send = async () => {
    if (!view || !message.trim() || busy) return;
    const textToSend = message.trim();
    setBusy(true);
    setError('');
    try {
      setView(await nexus.addComposerTurn(view.session.id, textToSend, view.session.revision));
      setMessage('');
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const finalize = async (confirmed = false) => {
    if (!view) return;
    if (!confirmed && composerNeedsGapConfirmation(view.brief)) {
      setConfirmGaps(true);
      setError(t('work.confirmGaps'));
      return;
    }
    setBusy(true);
    setError('');
    try {
      const res = await nexus.finalizeComposerSession(view.session.id, selectedSkillIds, confirmed);
      setArtifact(res);
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      setError(detail);
      setConfirmGaps(detail.includes('open questions') || detail.includes('gaps'));
    } finally {
      setBusy(false);
    }
  };

  const refine = async () => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const newArtifact = await nexus.refineComposerArtifact(view.session.id, refineText.trim());
      setArtifact(newArtifact);
      setShowRefineInput(false);
      setRefineText('');
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const resolveUnknown = async (unknownId: string, status: string, answerOverride?: string) => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const answer = answerOverride ?? unknownAnswers[unknownId] ?? '';
      const updated = await nexus.resolveComposerUnknown(
        view.session.id,
        unknownId,
        answer,
        status,
        view.session.revision,
      );
      setView(updated);
      setUnknownAnswers((prev) => {
        const next = { ...prev };
        delete next[unknownId];
        return next;
      });
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const rememberUnknown = async (unknown: { id: string; field?: string }, answer: string) => {
    if (!view) return;
    try {
      await nexus.upsertIntelligenceMemory({
        scope: 'PROJECT',
        project_id: project.id,
        key: unknown.field || unknown.id,
        value: answer,
        source: 'explicit',
        force: true,
      });
    } catch (err) {
      // Non-promotable keys can't become memories, but the answer below must
      // still resolve the unknown instead of stranding it.
      setError(err instanceof Error ? err.message : String(err));
    }
    await resolveUnknown(unknown.id, 'ANSWERED', answer);
  };

  const updateSkill = async (skillId: string, state: 'ACCEPTED' | 'REJECTED') => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      setView(await nexus.updateComposerSkillState(view.session.id, skillId, state));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    if (!artifact) return;
    try {
      await navigator.clipboard.writeText(artifact.content);
    } catch {
      setError(t('work.composer.copyFailed', 'Could not copy the prompt in this browser.'));
    }
  };

  const selectedAgent = agents.find((agent) => agent.id === selectedAgentId);
  const startAgentIfNeeded = !askActionForStatus(selectedAgent?.status || 'STOPPED').startIfNeeded;

  if (!view) {
    return (
      <Card className="nx-composer-goal-bar">
        <div className={`nx-composer-goal-bar__label ${styles.flexRowBetween}`}>
          <div className={styles.flexRowGap6}>
            <Sparkles size={16} />
            <span>{t('work.composer.startTitle', 'Start an elaboration')}</span>
          </div>
          <div className={styles.flexRowGap4}>
            <Button
              size="sm"
              tone={inputMode === 'IDEA' ? 'brand' : 'default'}
              onClick={() => setInputMode('IDEA')}
              disabled={busy}
            >
              <Lightbulb size={13} /> {t('work.composer.ideaMode', 'Idea / Explore')}
            </Button>
            <Button
              size="sm"
              tone={inputMode === 'EXISTING_PROMPT' ? 'brand' : 'default'}
              onClick={() => setInputMode('EXISTING_PROMPT')}
              disabled={busy}
            >
              <FileText size={13} /> {t('work.composer.existingPromptMode', 'Existing prompt')}
            </Button>
          </div>
        </div>

        <div className={styles.flexColGap8}>
          <Input
            value={draft}
            onChange={setDraft}
            placeholder={
              inputMode === 'IDEA'
                ? t(
                    'work.composer.ideaPlaceholder',
                    'Which idea or goal do you want to turn into a strong prompt?',
                  )
                : t(
                    'work.composer.existingGoalPlaceholder',
                    'What is the main goal of this existing prompt?',
                  )
            }
            style={{ flex: 1 }}
            disabled={busy}
          />
          {inputMode === 'EXISTING_PROMPT' && (
            <textarea
              className={styles.sourceTextarea}
              value={sourcePrompt}
              onChange={(e) => setSourcePrompt(e.target.value)}
              placeholder={t(
                'work.composer.sourcePlaceholder',
                'Paste the original prompt for gap analysis, completeness and structure…',
              )}
              rows={4}
              disabled={busy}
            />
          )}
          <div className={styles.flexRowEnd}>
            <Button
              tone="brand"
              disabled={
                !draft.trim() || (inputMode === 'EXISTING_PROMPT' && !sourcePrompt.trim()) || busy
              }
              onClick={() => void create()}
            >
              <MessageCircle size={14} />{' '}
              {busy
                ? t('work.composer.starting', 'Starting…')
                : inputMode === 'EXISTING_PROMPT'
                  ? t('work.composer.analyzePrompt', 'Analyze prompt')
                  : t('work.composer.converse', 'Converse')}
            </Button>
          </div>
        </div>

        {sessions.length > 0 && (
          <div className={styles.flexRowWrapGap8}>
            <small className={styles.mutedLabel}>
              {t('work.composer.previousSessions', 'Previous elaborations:')}
            </small>
            <Select
              placeholder={t('work.composer.resumeSession', 'Resume a saved session…')}
              value=""
              onChange={async (id) => {
                if (!id) return;
                setBusy(true);
                setError('');
                try {
                  setView(await nexus.getComposerSession(id));
                } catch (err) {
                  setError(err instanceof Error ? err.message : String(err));
                } finally {
                  setBusy(false);
                }
              }}
              options={sessionOptions}
              selectStyle={{ fontSize: '0.857rem', height: 28 }}
            />
          </div>
        )}
        {error && <div className="nx-inline-error">{error}</div>}
      </Card>
    );
  }

  const archetype = view.brief.intent?.archetype;
  const archetypeLabel = archetype ? ARCHETYPE_LABELS[archetype] || archetype : null;
  const turns = view.turns || [];
  const showDestinationDock =
    turns.length > 0 ||
    Boolean(artifact) ||
    view.session.state === 'FINALIZED' ||
    promptReadyEnough;
  const readinessScore = view.brief.readiness?.score ?? 0;
  const readinessState = view.brief.readiness?.state || 'UNKNOWN';
  const inferredPremises = asArray<BriefUnknown>(view.brief.unknowns).filter(
    (u) => u.status === 'INFERRED' && u.inferred_value,
  );

  return (
    <div className="nx-composer-deliberative">
      <div className="nx-composer-deliberative__header">
        <div>
          <h2>
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
          {sessions.length > 1 && (
            <Select
              value={view.session.id}
              onChange={async (id) => {
                if (!id || id === view.session.id) return;
                setBusy(true);
                setError('');
                try {
                  setView(await nexus.getComposerSession(id));
                  setArtifact(null);
                } catch (err) {
                  setError(err instanceof Error ? err.message : String(err));
                } finally {
                  setBusy(false);
                }
              }}
              options={sessionOptions}
              selectStyle={{ fontSize: '0.857rem', height: 28 }}
            />
          )}
          <Badge tone={view.session.state === 'FINALIZED' ? 'success' : 'brand'}>
            {t(composerSessionStateKey(view.session.state), view.session.state)}
          </Badge>
          <Button
            size="sm"
            disabled={busy}
            onClick={() => {
              setView(null);
              setArtifact(null);
            }}
          >
            <Plus size={13} /> {t('work.composer.newSession', 'New')}
          </Button>
        </div>
      </div>

      <div className="nx-composer-deliberative__grid">
        <Card>
          <strong>{t('work.composer.conversation', 'Elaboration conversation')}</strong>
          <div className="nx-composer-turns">
            {turns.length === 0 ? (
              <p className="nx-muted-copy">
                {t(
                  'work.composer.emptyTurns',
                  'Describe context, desired outcome and constraints. Composer keeps this elaboration.',
                )}
              </p>
            ) : (
              turns.map((turn) => (
                <div
                  key={turn.id}
                  className={`nx-composer-turn nx-composer-turn--${turn.role.toLowerCase()}`}
                >
                  <small>
                    {turn.role === 'USER'
                      ? t('work.composer.you', 'You')
                      : t('work.composer.composerRole', 'Composer')}
                  </small>
                  <p>{turn.content}</p>
                </div>
              ))
            )}
            {nextUnknown && view.session.state !== 'FINALIZED' ? (
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
                onAnswer={(answer) => void resolveUnknown(nextUnknown.id, 'ANSWERED', answer)}
                onDismiss={() => void resolveUnknown(nextUnknown.id, 'DISMISSED', '')}
                onRemember={(answer) => void rememberUnknown(nextUnknown, answer)}
              />
            ) : null}
          </div>
          {view.session.state !== 'FINALIZED' && (
            <div className={`nx-composer-goal-bar__row ${styles.flexRowGap8mt}`}>
              <textarea
                className={`nx-textarea ${styles.composerTextarea}`}
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                    e.preventDefault();
                    void send();
                  }
                }}
                placeholder={t(
                  'work.composer.messagePlaceholder',
                  'Add a requirement, decision or gap answer… (Enter sends, Shift+Enter new line)',
                )}
                rows={2}
                disabled={busy}
              />
              <Button
                tone="brand"
                disabled={!message.trim() || busy}
                onClick={() => void send()}
                className={styles.sendButton}
              >
                <Send size={14} />{' '}
                {busy ? t('work.composer.sending', 'Sending…') : t('work.composer.send', 'Send')}
              </Button>
            </div>
          )}
        </Card>

        <aside className={`nx-composer-deliberative__rail ${styles.rail}`}>
          <Card className={styles.railCard}>
            <strong>{t('work.composer.liveBrief', 'Live briefing')}</strong>
            <div className={styles.flexColGap10}>
              {briefItems.length > 0 ? (
                briefItems.map(([label, value]) => (
                  <div key={String(label)}>
                    <small className={styles.mutedLabel}>{label}</small>
                    <p className={styles.briefValue}>{value}</p>
                  </div>
                ))
              ) : (
                <p className={`nx-muted-copy ${styles.briefEmptyHint}`}>
                  {t(
                    'work.composer.briefEmpty',
                    'The live briefing synthesizes goal, context, criteria and decisions as the conversation advances.',
                  )}
                </p>
              )}
            </div>
          </Card>

          {nextQuestion ? (
            <Card className={styles.railCard}>
              <strong>{t('work.composer.nextQuestion', 'Next question')}</strong>
              <p className={styles.nextQuestionText}>{nextQuestion}</p>
              <small className={styles.mutedLabel}>
                {t(
                  'work.composer.answerInThread',
                  'Answer in the conversation thread with the options shown there.',
                )}
              </small>
            </Card>
          ) : null}

          {inferredPremises.length > 0 ? (
            <Card className={styles.railCard}>
              <strong>{t('work.assumedPremise', 'Nexus assumed — edit')}</strong>
              <div className={styles.flexColGap6}>
                {inferredPremises.slice(0, 4).map((item) => (
                  <p key={item.id} className={styles.briefValue}>
                    {item.inferred_value}
                  </p>
                ))}
              </div>
            </Card>
          ) : null}

          <Card className={styles.railCard}>
            <div className={styles.readinessCompact}>
              <strong>{t('work.composer.readiness', 'Readiness')}</strong>
              <Badge tone={readinessState === 'READY' || promptReadyEnough ? 'success' : 'warning'}>
                {readinessScore}% · {readinessState}
              </Badge>
            </div>
            <p className={styles.readinessSummary}>
              {view.brief.readiness?.summary ||
                t('work.composer.readinessEmpty', 'No assessment yet.')}
            </p>
            {asArray<PromptReadinessCheck>(view.brief.readiness?.checks).length > 0 && (
              <div className={`${styles.flexColGap6} ${styles.readinessSection}`}>
                {asArray<PromptReadinessCheck>(view.brief.readiness?.checks).map((check) => (
                  <div key={check.key} className={styles.readinessCheckItem}>
                    <div className={styles.readinessCheckContent}>
                      <span className={styles.readinessCheckLabel}>{check.label || check.key}</span>
                    </div>
                    <span
                      className={
                        check.score >= 80
                          ? styles.readinessCheckScoreHigh
                          : styles.readinessCheckScore
                      }
                    >
                      {check.score}%
                    </span>
                  </div>
                ))}
              </div>
            )}
          </Card>

          <Card className={styles.railCard}>
            <strong>{t('work.composer.gaps', 'Open refinements')}</strong>
            {actionableGaps.length === 0 ? (
              <p className="nx-muted-copy">{t('work.composer.noGaps', 'No open gaps.')}</p>
            ) : (
              <ul className={styles.flexColGap6}>
                {actionableGaps.map((unknown) => (
                  <li key={unknown.id} className={styles.unknownQuestion}>
                    <Badge tone={unknown.severity === 'BLOCKING' ? 'danger' : 'warning'}>
                      {unknown.severity}
                    </Badge>{' '}
                    {unknown.question}
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </aside>
      </div>

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
                {skill.state}
              </Badge>
              {skill.state !== 'UNAVAILABLE' && skill.state !== 'APPLIED' && (
                <>
                  <Button
                    size="sm"
                    tone="brand"
                    disabled={busy}
                    onClick={() => void updateSkill(skill.skill_id, 'ACCEPTED')}
                  >
                    {t('work.composer.accept', 'Accept')}
                  </Button>
                  <Button
                    size="sm"
                    disabled={busy}
                    onClick={() => void updateSkill(skill.skill_id, 'REJECTED')}
                  >
                    {t('work.composer.dismiss', 'Dismiss')}
                  </Button>
                </>
              )}
            </div>
          ))}
        </Card>
      )}

      {error && <div className="nx-inline-error">{error}</div>}

      {!artifact && view.session.state !== 'FINALIZED' && (
        <div className="nx-composer-header-actions">
          <Button tone="brand" disabled={busy} onClick={() => void finalize()}>
            <Sparkles size={14} />{' '}
            {busy
              ? t('work.composer.finalizing', 'Finalizing…')
              : t('work.composer.finalize', 'Finish elaboration')}
          </Button>
          {confirmGaps && (
            <Button tone="warning" disabled={busy} onClick={() => void finalize(true)}>
              {t('work.composer.finalizeWithGaps', 'Finish with confirmed gaps')}
            </Button>
          )}
        </div>
      )}

      {artifact && (
        <Card className={styles.artifactCard}>
          <div className={styles.artifactHeader}>
            <strong>
              {t('work.composer.canonicalPrompt', 'Canonical prompt')} · v{artifact.version}
            </strong>
            <Badge tone="success">
              {t('work.composer.immutableVersion', 'Immutable version')} #{artifact.version}
            </Badge>
          </div>
          <pre className={`nx-flow-step-compare ${styles.preWrap}`}>{artifact.content}</pre>
          <div className={`nx-composer-header-actions ${styles.artifactActions}`}>
            <Button disabled={busy} onClick={() => setShowRefineInput(!showRefineInput)}>
              <RefreshCw size={14} /> {t('work.refine', 'Refine')} (v{artifact.version + 1})
            </Button>
          </div>

          {showRefineInput && (
            <div className={styles.refinementPanel}>
              <small className={styles.refinementLabel}>
                {t('work.composer.refineHint', 'Optional refinement instruction:')}
              </small>
              <Input
                value={refineText}
                onChange={setRefineText}
                placeholder={t(
                  'work.composer.refinePlaceholder',
                  "E.g. 'Add PostgreSQL support', 'Focus only on the REST API'…",
                )}
                disabled={busy}
              />
              <div className={styles.refinementActions}>
                <Button size="sm" disabled={busy} onClick={() => setShowRefineInput(false)}>
                  {t('common.cancel', 'Cancel')}
                </Button>
                <Button size="sm" tone="brand" disabled={busy} onClick={() => void refine()}>
                  {busy
                    ? t('work.composer.generatingVersion', 'Generating v{{version}}…', {
                        version: artifact.version + 1,
                      })
                    : t('work.composer.generateRevision', 'Generate new revision (v{{version}})', {
                        version: artifact.version + 1,
                      })}
                </Button>
              </div>
            </div>
          )}
        </Card>
      )}

      {showDestinationDock && (
        <div className={`nx-composer-destination-dock ${styles.destinationDock}`}>
          <div className={styles.destinationDockInner}>
            <small className={styles.destinationLabel}>
              {t('work.composer.destinations', 'Destinations')}
            </small>
            <div className={styles.destinationActions}>
              <Button onClick={() => void copy()} disabled={!artifact}>
                <ClipboardCopy size={14} /> {t('work.composer.copy', 'Copy')}
              </Button>
              {agents.length > 0 && (
                <>
                  <Select
                    value={selectedAgentId}
                    onChange={setSelectedAgentId}
                    options={agents.map((agent) => ({ value: agent.id, label: agent.name }))}
                    selectStyle={{ minWidth: 150 }}
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
                onClick={() => {
                  if (artifact) onTransformFlow(artifact);
                }}
              >
                <Layers size={14} /> {t('work.composer.transformFlow', 'Turn into Flow')}
              </Button>
            </div>
            {destinationGate && artifact && (!canMaterialize || !canExecute) && (
              <small className="nx-muted-copy">
                {t('work.copyAlwaysAvailable', 'Copy remains available.')}{' '}
                {t(destinationGate.reasonKey, destinationGate.reason)}
              </small>
            )}
          </div>
        </div>
      )}

      <TaskPreparationDialog
        open={preparationOpen}
        agentId={selectedAgentId}
        agentName={selectedAgent?.name}
        projectId={project.id}
        initialPrompt={artifact?.content || ''}
        startIfNeeded={startAgentIfNeeded}
        onClose={() => setPreparationOpen(false)}
        onSubmit={async (prompt, skills, startIfNeeded, contextFingerprintId) => {
          await nexus.askAgent(selectedAgentId, prompt, startIfNeeded, skills, {
            projectId: project.id,
            contextFingerprintId: contextFingerprintId || '',
          });
          setMessage(t('work.composer.sentToAgent', 'Task sent to Agent.'));
        }}
      />
    </div>
  );
};
