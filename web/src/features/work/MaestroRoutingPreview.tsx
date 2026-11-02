import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, InlineAlert } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type {
  MaestroRoutePreviewResult,
  MaestroRoutingProvider,
  MaestroWorkAssignmentInput,
  Project,
  WorkPlan,
} from '../../types';
import type { FlowDraftModel, FlowStepModel } from './flowModel';
import styles from './MaestroRoutingPreview.module.scss';

type SelectedProvider = MaestroRoutingProvider | '';
type AssignmentDraft = {
  providerId: SelectedProvider;
  model: string;
  sandbox: MaestroWorkAssignmentInput['sandbox'];
  worktreePath: string;
};

const PROVIDERS: MaestroRoutingProvider[] = ['agy', 'opencode', 'codex', 'claude'];
const LABELS: Record<MaestroRoutingProvider, string> = {
  agy: 'Antigravity',
  opencode: 'OpenCode',
  codex: 'Codex',
  claude: 'Claude',
};
const defaultSandbox = (providerId: SelectedProvider): MaestroWorkAssignmentInput['sandbox'] =>
  providerId === 'agy' ? 'enabled' : providerId === 'codex' ? 'workspace-write' : 'none';

function suggested(step: FlowStepModel): AssignmentDraft {
  const provider = PROVIDERS.find((id) => id === step.provider) || '';
  return {
    providerId: provider,
    model: step.profile || '',
    sandbox: defaultSandbox(provider),
    worktreePath: '',
  };
}

function workPackageProposal(flow: FlowDraftModel) {
  return {
    tasks: flow.steps.map((step) => ({
      id: step.id,
      title: step.title,
      objective: step.goal,
      dependsOn: step.dependencies,
      acceptanceCriteria: step.acceptanceCriteria,
      requiredCapabilities: [step.role || 'engineering'],
    })),
  };
}

/**
 * User-controlled preview only. Execution stays disabled until Maestro owns a
 * durable async Run. Do not use Nexus MissionRunner as a hidden fallback.
 */
export function MaestroRoutingPreview({
  project,
  plan,
  draft,
  dirty,
  onProviderChange,
}: {
  project: Project;
  plan: WorkPlan;
  draft: FlowDraftModel;
  dirty: boolean;
  onProviderChange: (taskId: string, providerId: SelectedProvider) => void;
}) {
  const { t } = useTranslation();
  const artifactId = plan.structured_facts?.['nexus.source_artifact_id'] || '';
  const [assignments, setAssignments] = useState<Record<string, AssignmentDraft>>({});
  const [preview, setPreview] = useState<MaestroRoutePreviewResult | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const requestVersion = useRef(0);
  const draftFingerprint = JSON.stringify({
    id: draft.id,
    revision: draft.revision,
    steps: draft.steps.map((step) => [
      step.id,
      step.title,
      step.goal,
      step.dependencies,
      step.acceptanceCriteria,
      step.role,
      step.provider,
    ]),
  });

  useEffect(() => {
    requestVersion.current += 1;
    setPreview(null);
    setError('');
  }, [draftFingerprint, plan.current_revision, artifactId]);

  useEffect(() => {
    requestVersion.current += 1;
    setAssignments({});
    setPreview(null);
  }, [project.id, plan.id]);

  const selected = useCallback(
    (step: FlowStepModel) => {
      const existing = assignments[step.id];
      const savedProvider = PROVIDERS.find((id) => id === step.provider) || '';
      if (existing && savedProvider && existing.providerId !== savedProvider) {
        return {
          ...existing,
          providerId: savedProvider,
          sandbox: defaultSandbox(savedProvider),
          model: '',
        };
      }
      return existing || suggested(step);
    },
    [assignments],
  );
  const patchAssignment = (step: FlowStepModel, patch: Partial<AssignmentDraft>) => {
    requestVersion.current += 1;
    setPreview(null);
    setError('');
    setAssignments((old) => ({
      ...old,
      [step.id]: { ...selected(step), ...patch },
    }));
  };

  const hasCriteria = draft.steps.every((step) =>
    step.acceptanceCriteria?.some((item) => item.trim().length >= 3),
  );
  const hasSelections =
    draft.steps.length > 0 &&
    draft.steps.every((step) => {
      const row = selected(step);
      return row.providerId !== '' && row.worktreePath.trim().length > 0;
    });
  const canPreview =
    Boolean(artifactId) &&
    !dirty &&
    hasCriteria &&
    hasSelections &&
    !loading &&
    draft.steps.length <= 32;

  const runPreview = async () => {
    if (!canPreview) return;
    const version = ++requestVersion.current;
    setError('');
    setPreview(null);
    setLoading(true);
    try {
      const choices = draft.steps.map((step): MaestroWorkAssignmentInput => {
        const choice = selected(step);
        // canPreview requires an explicit selection from the operator.
        return {
          taskId: step.id,
          providerId: choice.providerId as MaestroRoutingProvider,
          model: choice.model.trim() || undefined,
          sandbox: choice.sandbox,
          worktreePath: choice.worktreePath.trim(),
        };
      });
      const result = await nexusApi.previewMaestroRouting(
        project.id,
        artifactId,
        workPackageProposal(draft),
        choices,
      );
      if (version === requestVersion.current) setPreview(result);
    } catch (reason) {
      if (version === requestVersion.current) {
        setError(
          reason instanceof Error
            ? reason.message
            : t('flow.maestroPreviewError', 'Maestro indisponível para pré-validação.'),
        );
      }
    } finally {
      // Only one preview can run at a time. Edits invalidate its result, but
      // must never leave the submit control stuck in a loading state.
      setLoading(false);
    }
  };

  return (
    <Card className={styles.panel}>
      <details>
        <summary className={styles.summary}>
          {t('flow.maestroAgentsTitle', 'Agentes por tarefa · Maestro')}
        </summary>
        <p className={styles.description}>
          {t(
            'flow.maestroAgentsDescription',
            'Escolha os agentes para cada etapa. O Maestro valida as atribuições e as worktrees antes da aprovação. Esta tela não inicia execuções.',
          )}
        </p>
        {!artifactId ? (
          <InlineAlert
            tone="warning"
            title={t('flow.maestroNeedsComposer', 'Origem do Composer obrigatória')}
          >
            {t(
              'flow.maestroNeedsComposerText',
              'Use um Flow derivado de um PromptArtifact finalizado no Composer para validar sua origem.',
            )}
          </InlineAlert>
        ) : (
          <>
            {draft.steps.map((step) => {
              const item = selected(step);
              return (
                <fieldset className={styles.step} key={step.id} disabled={loading}>
                  <legend>{step.title}</legend>
                  <div className={styles.fields}>
                    <label>
                      {t('flow.maestroExecutor', 'Agente executor')}
                      <select
                        value={item.providerId}
                        onChange={(event) => {
                          const provider = event.target.value as SelectedProvider;
                          patchAssignment(step, {
                            providerId: provider,
                            sandbox: defaultSandbox(provider),
                            model: '',
                          });
                          onProviderChange(step.id, provider);
                        }}
                      >
                        <option value="">{t('flow.selectProvider', 'Selecione…')}</option>
                        {PROVIDERS.map((provider) => (
                          <option key={provider} value={provider}>
                            {LABELS[provider]}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label>
                      {t('flow.maestroModel', 'Modelo (opcional)')}
                      <input
                        type="text"
                        value={item.model}
                        maxLength={128}
                        placeholder={t('flow.maestroDefaultModel', 'Padrão do agente')}
                        onChange={(event) => patchAssignment(step, { model: event.target.value })}
                      />
                    </label>
                    <label>
                      {t('flow.maestroSandbox', 'Isolamento')}
                      <select
                        value={item.sandbox}
                        onChange={(event) =>
                          patchAssignment(step, {
                            sandbox: event.target.value as AssignmentDraft['sandbox'],
                          })
                        }
                      >
                        <option value="none">
                          {t('flow.maestroSandboxNone', 'Sem sandbox declarado')}
                        </option>
                        {item.providerId === 'agy' && (
                          <option value="enabled">Sandbox do AGY</option>
                        )}
                        {item.providerId === 'codex' && (
                          <>
                            <option value="read-only">Somente leitura</option>
                            <option value="workspace-write">Escrita no workspace</option>
                          </>
                        )}
                      </select>
                    </label>
                    <label className={styles.worktree}>
                      {t('flow.maestroWorktree', 'Caminho da worktree Git')}
                      <input
                        type="text"
                        value={item.worktreePath}
                        placeholder={project.canonical_path}
                        autoComplete="off"
                        onChange={(event) =>
                          patchAssignment(step, { worktreePath: event.target.value })
                        }
                      />
                    </label>
                  </div>
                </fieldset>
              );
            })}
            <p className={styles.hint}>
              {t(
                'flow.maestroIsolationHint',
                'Tarefas independentes precisam de worktrees separadas. Antigravity e OpenCode não são trocados por outro agente automaticamente.',
              )}
            </p>
            {dirty && (
              <InlineAlert tone="warning" title={t('flow.maestroUnsaved', 'Revisão não salva')}>
                Salve o Flow antes da pré-validação.
              </InlineAlert>
            )}
            {!hasCriteria && (
              <InlineAlert
                tone="warning"
                title={t('flow.maestroMissingCriteria', 'Critérios de aceite incompletos')}
              >
                Inclua critérios verificáveis em cada etapa antes de consultar o Maestro.
              </InlineAlert>
            )}
            <Button disabled={!canPreview} onClick={() => void runPreview()}>
              {loading
                ? t('flow.maestroValidating', 'Validando com Maestro…')
                : t('flow.maestroPreview', 'Pré-validar atribuições')}
            </Button>
            {error && (
              <InlineAlert
                tone="danger"
                title={t('flow.maestroPreviewFailed', 'Pré-validação indisponível')}
              >
                {error}
              </InlineAlert>
            )}
            {preview && (
              <div className={styles.result} aria-live="polite">
                <strong>
                  {preview.ready
                    ? t('flow.maestroReviewReady', 'Pronto para revisão — não executado')
                    : t('flow.maestroReviewBlocked', 'Atribuições bloqueadas')}
                </strong>
                <ul>
                  {preview.assignments.map((assignment) => (
                    <li key={assignment.taskId}>
                      {assignment.taskId}: {LABELS[assignment.providerId]} — {assignment.status}
                      {assignment.code ? ` (${assignment.code})` : ''}
                    </li>
                  ))}
                  {preview.blockers.map((blocker, index) => (
                    <li key={`${blocker.code}-${blocker.taskId || index}`}>
                      {blocker.taskId ? `${blocker.taskId}: ` : ''}
                      {blocker.code}
                      {blocker.message ? ` — ${blocker.message}` : ''}
                    </li>
                  ))}
                </ul>
                <p className={styles.hint}>
                  {t(
                    'flow.maestroPreviewOnly',
                    'Este resultado não autoriza execução. O despacho durável permanece no Maestro; não há fallback local do Nexus.',
                  )}
                </p>
              </div>
            )}
          </>
        )}
      </details>
    </Card>
  );
}
