import React, { useEffect, useMemo, useState } from 'react';
import { Button, Dialog, Textarea } from '../design-system';
import { useTranslation } from 'react-i18next';
import { nexus } from '../nexus/api';
import type { CatalogSkill, ContextReadiness } from '../types';
import styles from './TaskPreparationDialog.module.scss';

export interface TaskPreparationDialogProps {
  open: boolean;
  agentId: string;
  agentName?: string;
  projectId?: string;
  initialPrompt?: string;
  startIfNeeded: boolean;
  onClose: () => void;
  onSubmit: (
    prompt: string,
    skillIds: string[],
    startIfNeeded: boolean,
    contextFingerprintId?: string,
  ) => Promise<void>;
}

const statusTone = (state?: string) => (state === 'READY' ? 'success' : 'warning');

export const TaskPreparationDialog: React.FC<TaskPreparationDialogProps> = ({
  open,
  agentId,
  agentName,
  projectId,
  initialPrompt = '',
  startIfNeeded,
  onClose,
  onSubmit,
}) => {
  const { t } = useTranslation();
  const [prompt, setPrompt] = useState(initialPrompt);
  const [catalog, setCatalog] = useState<CatalogSkill[]>([]);
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [readiness, setReadiness] = useState<ContextReadiness | null>(null);
  const [busy, setBusy] = useState(false);
  const [contextBusy, setContextBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open) return;
    setPrompt(initialPrompt);
    setError('');
    setSelectedSkills([]);
    void Promise.all([
      nexus
        .getMaestroCatalog()
        .then((value) => setCatalog(value.library || []))
        .catch(() => setCatalog([])),
      projectId
        ? nexus
            .getContextReadiness(projectId)
            .then(setReadiness)
            .catch(() => setReadiness(null))
        : Promise.resolve(),
    ]);
  }, [initialPrompt, open, projectId]);

  const selected = useMemo(
    () =>
      selectedSkills
        .map((id) => catalog.find((skill) => skill.id === id))
        .filter(Boolean) as CatalogSkill[],
    [catalog, selectedSkills],
  );

  const prepareContext = async (createContext: boolean) => {
    if (!projectId || contextBusy) return;
    setContextBusy(true);
    setError('');
    try {
      setReadiness(await nexus.prepareContext(projectId, createContext));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setContextBusy(false);
    }
  };

  const submit = async () => {
    if (!prompt.trim() || busy) return;
    if (projectId && readiness?.state !== 'READY') {
      setError(
        t(
          'taskPreparation.contextRequired',
          'Prepare o contexto durável antes de enviar a tarefa.',
        ),
      );
      return;
    }
    setBusy(true);
    setError('');
    try {
      await onSubmit(
        prompt.trim(),
        selectedSkills,
        startIfNeeded,
        readiness?.current_fingerprint_id,
      );
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const toggleSkill = (id: string) => {
    setSelectedSkills((current) =>
      current.includes(id)
        ? current.filter((skillId) => skillId !== id)
        : current.length < 3
          ? [...current, id]
          : current,
    );
  };

  const preview = JSON.stringify(
    {
      objective: prompt.trim(),
      destination: { agent_id: agentId, agent_name: agentName || agentId },
      context: readiness
        ? {
            project_id: readiness.project_id,
            worktree: readiness.current_fingerprint.canonical_path,
            branch: readiness.current_fingerprint.branch,
            readiness: readiness.state,
            instruction: 'Read AGENTS.md and DEV/ before acting.',
          }
        : { instruction: 'Read AGENTS.md and DEV/ before acting.' },
      skills: selected.map((skill) => ({
        id: skill.id,
        source: skill.source,
        contract: skill.contract || '',
      })),
    },
    null,
    2,
  );

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={`${t('taskPreparation.title', 'Preparar tarefa')}${agentName ? ` · ${agentName}` : ''}`}
      wide
    >
      <div className={styles.dialog}>
        <ol
          aria-label={t('taskPreparation.steps', 'Etapas da preparação')}
          className={styles.steps}
        >
          <li className={styles.step}>
            <strong>{t('taskPreparation.objective', 'Objetivo')}</strong>
            <Textarea
              rows={5}
              value={prompt}
              onChange={setPrompt}
              placeholder={t('taskPreparation.placeholder', 'Descreva a tarefa…')}
              autoFocus
            />
          </li>
          <li className={styles.step}>
            <strong>{t('taskPreparation.context', 'Contexto')}</strong>
            <p className={styles.muted}>
              {t(
                'taskPreparation.contextHint',
                'O Agente deve ler AGENTS.md e DEV/ antes de atuar.',
              )}
            </p>
            {projectId && readiness && (
              <p data-tone={statusTone(readiness.state)} role="status">
                {t('taskPreparation.contextState', 'Estado do contexto')}: {readiness.state} ·{' '}
                {readiness.current_fingerprint.branch || 'worktree'}
              </p>
            )}
            {projectId && readiness?.state !== 'READY' && (
              <div className={styles.actions}>
                <Button disabled={contextBusy} onClick={() => void prepareContext(false)}>
                  {t('taskPreparation.prepareContext', 'Preparar/atualizar contexto')}
                </Button>
                {readiness?.state === 'MISSING' && (
                  <Button disabled={contextBusy} onClick={() => void prepareContext(true)}>
                    {t('taskPreparation.createAgents', 'Criar AGENTS.md')}
                  </Button>
                )}
              </div>
            )}
          </li>
          <li className={styles.step}>
            <strong>{t('taskPreparation.skillsReview', 'Skills e revisão')}</strong>
            <div
              role="group"
              aria-label={t('taskPreparation.skillsLabel', 'Skills selecionáveis')}
              className={styles.skillGroup}
            >
              {catalog.slice(0, 12).map((skill) => (
                <button
                  className={styles.skillButton}
                  key={skill.id}
                  type="button"
                  aria-pressed={selectedSkills.includes(skill.id)}
                  onClick={() => toggleSkill(skill.id)}
                >
                  {selectedSkills.includes(skill.id) ? '✓ ' : ''}
                  {skill.name || skill.id} · {skill.availability}
                </button>
              ))}
            </div>
            <details open>
              <summary>{t('taskPreparation.preview', 'Prévia integral do envelope')}</summary>
              <pre
                className={styles.preview}
                tabIndex={0}
                aria-label={t('taskPreparation.previewContent', 'Conteúdo integral do envelope')}
              >
                {preview}
              </pre>
            </details>
          </li>
        </ol>
        {error && (
          <div className="nx-inline-error" role="alert">
            {error}
          </div>
        )}
        <div className={styles.actions}>
          <Button onClick={onClose}>{t('common.cancel', 'Cancelar')}</Button>
          <Button
            tone="brand"
            disabled={!prompt.trim() || busy || Boolean(projectId && readiness?.state !== 'READY')}
            onClick={() => void submit()}
          >
            {busy
              ? t('taskPreparation.sending', 'Enviando…')
              : startIfNeeded
                ? t('taskPreparation.startAndSend', 'Iniciar e enviar tarefa')
                : t('taskPreparation.send', 'Enviar tarefa')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
