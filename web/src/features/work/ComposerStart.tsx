import React, { useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FileText, Lightbulb, MessageCircle, RefreshCw, Settings2, Sparkles, Trash2 } from 'lucide-react';
import { Button, Card, ConfirmDialog, Input, Select } from '../../design-system';
import type { ComposerSession } from '../../types';
import { nexus } from '../../nexus/api';
import { composerSessionStateKey, formatComposerSessionTitle } from './composerModel';
import styles from './ComposerSurface.module.scss';

export type ComposerStartCreateArgs = {
  mode: 'IDEA' | 'EXISTING_PROMPT';
  interviewMode: 'QUICK' | 'STANDARD' | 'DEEP';
  draft: string;
  sourcePrompt: string;
};

/**
 * Initial elaboration screen shown while no session is loaded: idea vs.
 * existing-prompt modes, the draft inputs, previous-session picker and the
 * create action. Draft/mode state is local; the orchestrator keeps API + view.
 */
export const ComposerStart: React.FC<{
  projectId: string;
  onOpenIntelligenceSettings?: () => void;
  sessions: ComposerSession[];
  busy: boolean;
  error?: string;
  onCreate: (args: ComposerStartCreateArgs) => void;
  onResumeSession: (id: string) => void;
  onDeleteSession: (id: string, expectedRevision: number) => void;
}> = ({
  projectId,
  onOpenIntelligenceSettings,
  sessions,
  busy,
  error,
  onCreate,
  onResumeSession,
  onDeleteSession,
}) => {
  const { t } = useTranslation();
  const [inputMode, setInputMode] = useState<'IDEA' | 'EXISTING_PROMPT'>('IDEA');
  const [interviewMode, setInterviewMode] = useState<'QUICK' | 'STANDARD' | 'DEEP'>('STANDARD');
  const interviewDepthId = useId();
  const [draft, setDraft] = useState('');
  const [sourcePrompt, setSourcePrompt] = useState('');
  const [sessionForDeletion, setSessionForDeletion] = useState<ComposerSession | null>(null);
  const [selectedSavedSession, setSelectedSavedSession] = useState('');
  const [probeBusy, setProbeBusy] = useState(false);
  const [probeMessage, setProbeMessage] = useState('');
  const [probeSuccess, setProbeSuccess] = useState(false);

  const testConnection = async () => {
    if (probeBusy) return;
    setProbeBusy(true);
    setProbeMessage('');
    setProbeSuccess(false);
    try {
      await nexus.probeIntelligence(projectId);
      setProbeSuccess(true);
      setProbeMessage(
        t('work.composer.probeSuccess', 'A IA respondeu ao teste de conexão. Você pode tentar novamente.'),
      );
    } catch (reason) {
      setProbeMessage(
        reason instanceof Error
          ? reason.message
          : t('work.composer.probeFailed', 'Não foi possível testar a IA. Verifique Nexus Intelligence.'),
      );
    } finally {
      setProbeBusy(false);
    }
  };

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

  const disabled =
    !draft.trim() || (inputMode === 'EXISTING_PROMPT' && !sourcePrompt.trim()) || busy;

  return (
    <Card className={styles.goalBar}>
      <div className={`${styles.goalBarLabel} ${styles.flexRowBetween}`}>
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
          className={styles.flexGrow}
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
        <div className={styles.flexRowWrapGap8}>
          <label htmlFor={interviewDepthId} className={styles.mutedLabel}>
            {t('work.composer.interviewDepth', 'Interview depth')}
          </label>
          <Select
            id={interviewDepthId}
            value={interviewMode}
            onChange={(value) => setInterviewMode(value as 'QUICK' | 'STANDARD' | 'DEEP')}
            options={[
              {
                value: 'QUICK',
                label: t('work.composer.interviewQuick', 'Quick — fewer questions'),
              },
              {
                value: 'STANDARD',
                label: t('work.composer.interviewStandard', 'Standard — adaptive'),
              },
              {
                value: 'DEEP',
                label: t('work.composer.interviewDeep', 'Deep — challenge assumptions'),
              },
            ]}
            disabled={busy}
          />
        </div>
        <div className={styles.flexRowEnd}>
          <Button
            tone="brand"
            disabled={disabled}
            onClick={() =>
              onCreate({
                mode: inputMode,
                interviewMode,
                draft: draft.trim(),
                sourcePrompt: sourcePrompt.trim(),
              })
            }
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
            placeholder={t('work.composer.resumeSession', 'Escolha uma elaboração salva…')}
            value={selectedSavedSession}
            onChange={setSelectedSavedSession}
            options={sessionOptions}
            selectClassName={styles.sessionSelect}
          />
          <Button
            size="sm"
            disabled={!selectedSavedSession || busy}
            onClick={() => onResumeSession(selectedSavedSession)}
          >
            {t('work.composer.openSession', 'Abrir')}
          </Button>
          <Button
            size="sm"
            tone="danger"
            disabled={!selectedSavedSession || busy}
            onClick={() =>
              setSessionForDeletion(
                sessions.find((item) => item.id === selectedSavedSession) || null,
              )
            }
          >
            <Trash2 size={13} /> {t('common.delete', 'Excluir')}
          </Button>
        </div>
      )}
      <ConfirmDialog
        open={Boolean(sessionForDeletion)}
        title={t('work.composer.deleteElaboration', 'Excluir elaboração')}
        description={t(
          'work.composer.deleteElaborationConfirm',
          'Excluir "{{title}}" e suas mensagens e artefatos? A ação não pode ser desfeita. Elaborações vinculadas a Flows ou entregas ficam protegidas.',
          { title: sessionForDeletion?.title || '' },
        )}
        confirmLabel={t('work.composer.deleteElaboration', 'Excluir elaboração')}
        onCancel={() => setSessionForDeletion(null)}
        onConfirm={() => {
          if (sessionForDeletion) {
            onDeleteSession(sessionForDeletion.id, sessionForDeletion.revision);
            setSelectedSavedSession('');
          }
          setSessionForDeletion(null);
        }}
      />
      {error && (
        <div role="alert" className={styles.inlineError}>
          {error}
          <div className={styles.flexRowWrapGap8}>
            {onOpenIntelligenceSettings && (
              <Button size="sm" onClick={onOpenIntelligenceSettings}>
                <Settings2 size={13} /> {t('work.composer.openIntelligence', 'Configurar Nexus Intelligence')}
              </Button>
            )}
            <Button size="sm" disabled={probeBusy || busy} onClick={() => void testConnection()}>
              <RefreshCw size={13} />
              {probeBusy
                ? t('work.composer.probing', 'Testando conexão…')
                : t('work.composer.testConnection', 'Testar conexão da IA')}
            </Button>
          </div>
          {probeMessage && (
            <p role={probeSuccess ? 'status' : 'alert'}>
              {probeMessage}
            </p>
          )}
        </div>
      )}
    </Card>
  );
};
