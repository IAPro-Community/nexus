import React from 'react';
import { useTranslation } from 'react-i18next';
import { Play, Plus, Route, Sparkles, Trash2 } from 'lucide-react';
import { Badge, Button, Card, ConfirmDialog, Input, Select, Textarea } from '../../design-system';
import type { Agent, FlowLeaderPolicy } from '../../types';
import {
  executionPreviewDetails,
  localizeExecutionPreviewMessage,
} from './executionPreviewDetails';
import { GENERATE_BUDGET_SEC, type FlowDraftEditorController } from './useFlowDraftEditor';
import styles from './FlowDraftEditor.module.scss';

/**
 * Compact goal bar: request input / active draft header, preflight checklist,
 * recent-request chips, generation status and the per-Flow leader policy card.
 */
export const FlowGoalBar: React.FC<{ controller: FlowDraftEditorController; agents: Agent[] }> = ({
  controller,
  agents,
}) => {
  const { t } = useTranslation();
  const [deleteOpen, setDeleteOpen] = React.useState(false);
  const {
    selectedPlan,
    deletingPlan,
    deleteSelectedFlow,
    autoGoal,
    setAutoGoal,
    generating,
    generateStage,
    generateProvider,
    generateElapsedSec,
    generateHeartbeat,
    runBusy,
    flowDraft,
    flowDirty,
    flowErrors,
    goalChips,
    leader,
    setLeader,
    preflightReport,
    preflightBusy,
    executionPreview,
    executionPreviewBusy,
    costConfirmationPending,
    cancelGenerate,
    generatePlan,
    createManualPlan,
    saveLeader,
    launchRun,
    runExecutionPreview,
    runPreflight,
  } = controller;

  return (
    <Card className="nx-composer-goal-bar">
      <div className="nx-composer-goal-bar__label">
        <Sparkles size={16} />
        <span>
          {flowDraft && selectedPlan
            ? t('flow.activeDraft', 'Flow Rascunho Ativo')
            : t('flow.yourRequest', 'Sua solicitação')}
        </span>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        title={t('flow.deleteDraft', 'Excluir Flow')}
        description={t(
          'flow.deleteDraftConfirm',
          'Excluir "{{title}}" e todas as suas revisões? Esta ação não pode ser desfeita. Se houver execuções, evidências ou agendamentos vinculados, o Nexus impedirá a exclusão.',
          { title: selectedPlan?.title || 'Flow sem nome' },
        )}
        confirmLabel={t('flow.deleteDraft', 'Excluir Flow')}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => {
          setDeleteOpen(false);
          void deleteSelectedFlow();
        }}
      />

      {!flowDraft && (
        <div className="nx-composer-goal-bar__row">
          <div className={styles.promptInputGroup} data-tour="flow-prompt">
            <label htmlFor="flow-prompt-input" className={styles.promptInputLabel}>
              {t('flow.promptLabel', 'Prompt ou objetivo a decompor')}
            </label>
            <Textarea
              id="flow-prompt-input"
              value={autoGoal}
              onChange={setAutoGoal}
              rows={5}
              placeholder={t(
                'flow.promptPlaceholder',
                'Cole o prompt completo ou descreva a entrega. O Flow vai separar as atividades, dependências e critérios de aceite.',
              )}
              className={styles.goalInput}
            />
          </div>
          <Button
            tone="brand"
            disabled={!autoGoal.trim() || generating}
            onClick={() => void generatePlan()}
          >
            <Plus size={14} />{' '}
            {generating
              ? t('flow.generatingActivities', 'Planejando…')
              : t('flow.generateActivities', 'Gerar atividades')}
          </Button>
          <Button tone="default" disabled={generating} onClick={() => void createManualPlan()}>
            <Plus size={14} /> {t('flow.createManual', 'Criar manual')}
          </Button>
          {generating && (
            <Button tone="default" onClick={cancelGenerate}>
              {t('common.cancel', 'Cancelar')}
            </Button>
          )}
        </div>
      )}

      {flowDraft && selectedPlan && (
        <div className={`nx-composer-goal-bar__row ${styles.goalBarRow}`}>
          <div className={styles.goalBarTitleWrap}>
            <strong className={styles.goalBarTitle}>
              {selectedPlan.title || autoGoal || t('flow.goalFallback', 'Objetivo do Flow')}
            </strong>
            {selectedPlan.description && (
              <p className={`nx-muted-copy ${styles.goalBarSubtitle}`}>
                {selectedPlan.description}
              </p>
            )}
          </div>
          <div className={styles.goalBarActions}>
            <Button
              size="sm"
              tone="danger"
              disabled={deletingPlan || runBusy || generating}
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 size={13} />
              {deletingPlan
                ? t('flow.deleting', 'Excluindo…')
                : t('flow.deleteDraft', 'Excluir Flow')}
            </Button>
            <Badge tone={flowDirty ? 'warning' : 'success'}>
              {flowDirty
                ? t('flow.unsavedChanges', 'Alterações não salvas')
                : t('flow.revisionN', 'Revisão {{n}}', { n: selectedPlan.current_revision })}
            </Badge>
            <Button
              disabled={!selectedPlan || executionPreviewBusy}
              onClick={() => void runExecutionPreview()}
            >
              <Sparkles size={14} />{' '}
              {executionPreviewBusy
                ? t('flow.executionPreviewLoading', 'Avaliando execução…')
                : t('flow.executionPreview', 'Plano de execução')}
            </Button>
            <Button disabled={!selectedPlan || preflightBusy} onClick={() => void runPreflight()}>
              <Route size={14} />{' '}
              {preflightBusy
                ? t('flow.checking', 'Verificando…')
                : t('flow.preflight', 'Preflight')}
            </Button>
            <Button
              tone="brand"
              disabled={!selectedPlan || !flowDraft || runBusy || flowErrors.length > 0}
              onClick={() => void launchRun()}
            >
              <Play size={14} />{' '}
              {runBusy
                ? t('flow.starting', 'Iniciando…')
                : costConfirmationPending
                  ? t('flow.confirmCostAndRun', 'Confirmar custo e executar')
                  : t('flow.approveAndRun', 'Approve & Run')}
            </Button>
          </div>
        </div>
      )}

      {executionPreview && (
        <div className={styles.preflightCard}>
          <div className={styles.preflightHeader}>
            <div>
              <strong>{t('flow.executionPreviewTitle', 'Plano de execução')}</strong>
              <small className={styles.preflightCheckSub}>
                {t(
                  'flow.executionPreviewRevision',
                  'Revisão {{n}} · sem iniciar Agents ou runtimes',
                  {
                    n: executionPreview.revision,
                  },
                )}
              </small>
            </div>
            <Badge tone={executionPreview.ready ? 'success' : 'danger'}>
              {executionPreview.ready
                ? t('flow.executionPreviewReadyBadge', 'Executável')
                : t('flow.blocked', 'Bloqueado')}
            </Badge>
          </div>
          <div className={styles.preflightList}>
            <div className={styles.preflightItem}>
              <div>
                <span className={styles.preflightCheckText}>
                  {t('flow.costPolicy', 'Política de custo')} · {executionPreview.cost_budget.level}
                </span>
                <small className={styles.preflightCheckSub}>
                  {localizeExecutionPreviewMessage(
                    executionPreview.cost_budget.message,
                    t(
                      'flow.executionPreviewNoLocalCostBudget',
                      'No local cost budget is configured.',
                    ),
                  )}
                </small>
                {executionPreview.cost_anomaly.triggered ? (
                  <small className={styles.preflightCheckSub}>
                    {executionPreview.cost_anomaly.message}
                  </small>
                ) : null}
              </div>
              <Badge
                tone={
                  executionPreview.cost_confirmation_required
                    ? 'warning'
                    : executionPreview.cost_budget.level === 'OK'
                      ? 'success'
                      : 'default'
                }
              >
                {executionPreview.cost_confirmation_required
                  ? t('flow.costConfirmationBadge', 'Confirmação')
                  : executionPreview.cost_budget.level}
              </Badge>
            </div>
            {(executionPreview.warnings || []).map((warning) => (
              <div key={warning} className={styles.preflightItem}>
                <div>
                  <span className={styles.preflightCheckText}>
                    {t('flow.executionWarning', 'Aviso')}
                  </span>
                  <small className={styles.preflightCheckSub}>{warning}</small>
                </div>
                <Badge tone="warning">{t('flow.warning', 'Aviso')}</Badge>
              </div>
            ))}
            {(executionPreview.steps || []).map((step) => {
              const agentLabel = step.agent.will_create
                ? t('flow.executionPreviewCreateAgent', 'Criar especialista: {{role}}', {
                    role: step.agent.role || step.role || 'Agent',
                  })
                : step.agent.name ||
                  t('flow.executionPreviewAgentUnknown', 'Agent ainda não definido');
              const resourceLabel = step.resource.provider
                ? `${step.resource.provider}${
                    step.resource.profile ? ` / ${step.resource.profile}` : ''
                  }`
                : t('flow.executionPreviewResourceUnknown', 'recurso indisponível');
              const modelLabel = step.model.model
                ? ` · ${step.model.model}`
                : step.model.evidence === 'UNKNOWN'
                  ? ` · ${t('flow.executionPreviewModelAuto', 'modelo por evidência em runtime')}`
                  : '';
              const details = executionPreviewDetails(
                {
                  agentReason: step.agent.reason,
                  resourceReason: step.resource.reason,
                  blockers: step.blockers,
                },
                t('flow.executionPreviewNoProviderAccounts', 'No provider accounts are available.'),
              );
              return (
                <div key={step.package_id} className={styles.preflightItem}>
                  <div>
                    <span className={styles.preflightCheckText}>{step.title}</span>
                    <small className={styles.preflightCheckSub}>
                      {agentLabel} → {resourceLabel}
                      {modelLabel}
                    </small>
                    <small className={styles.preflightCheckSub}>{details.join(' · ')}</small>
                  </div>
                  <Badge tone={step.ready ? 'success' : 'danger'}>
                    {step.ready
                      ? t('flow.executionPreviewStepReady', 'Pronto')
                      : t('flow.blocked', 'Bloqueado')}
                  </Badge>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {preflightReport && (
        <div className={styles.preflightCard}>
          <div className={styles.preflightHeader}>
            <strong>{t('flow.preflightChecklist', 'Checklist de Preflight')}</strong>
            <Badge tone={preflightReport.ready ? 'success' : 'danger'}>
              {preflightReport.ready
                ? t('flow.readyForExecution', 'Pronto para Execução')
                : t('flow.blocked', 'Bloqueado')}
            </Badge>
          </div>
          <div className={styles.preflightList}>
            {(preflightReport.checks || []).map((ch) => (
              <div key={ch.key} className={styles.preflightItem}>
                <div>
                  <span className={styles.preflightCheckText}>{ch.label}</span>
                  <small className={styles.preflightCheckSub}>{ch.summary}</small>
                </div>
                <Badge
                  tone={
                    ch.status === 'PASS' ? 'success' : ch.status === 'WARN' ? 'warning' : 'danger'
                  }
                >
                  {ch.status}
                </Badge>
              </div>
            ))}
          </div>
        </div>
      )}

      {goalChips.length > 0 && !selectedPlan && (
        <div
          className="nx-composer-goal-chips"
          aria-label={t('flow.recentRequests', 'Solicitações recentes')}
        >
          <small className={styles.chipsLabel}>
            {t('flow.continueRecent', 'Continuar uma solicitação recente:')}
          </small>
          {goalChips.slice(0, 5).map((goal) => (
            <button
              key={goal}
              type="button"
              className="nx-composer-goal-chip"
              onClick={() => setAutoGoal(goal)}
              title={goal}
            >
              {goal.length > 72 ? `${goal.slice(0, 69)}…` : goal}
            </button>
          ))}
        </div>
      )}

      {generating && (
        <div className="nx-composer-generate-status" role="status" aria-live="polite">
          <strong>{generateStage || t('flow.preparing', 'Preparando…')}</strong>
          <div className="nx-composer-generate-status__meta">
            {generateProvider && (
              <span>
                {t('flow.provider', 'Provedor: {{provider}}', { provider: generateProvider })}
              </span>
            )}
            <span>
              {generateElapsedSec}s / ~{GENERATE_BUDGET_SEC}s
            </span>
          </div>
          {generateHeartbeat && (
            <span>
              {t(
                'flow.stillWaiting',
                'Ainda aguardando o modelo (não travou). Pode levar até ~{{budget}}s.',
                { budget: GENERATE_BUDGET_SEC },
              )}
            </span>
          )}
        </div>
      )}

      {flowDraft && selectedPlan && (
        <Card className={styles.leaderCard}>
          <strong>{t('flow.suggestedLeader', 'Agent líder sugerido')}</strong>
          <p className={`nx-muted-copy ${styles.leaderSubtitle}`}>
            {t(
              'flow.leaderHint',
              'Coordena as ondas do DAG e recebe os recibos. A sugestão é exclusiva deste Flow.',
            )}
          </p>
          <div className={`nx-composer-goal-bar__row ${styles.leaderRow}`}>
            <Input
              value={leader.role}
              onChange={(role) => setLeader((current) => ({ ...current, role }))}
              placeholder={t('flow.leaderRolePlaceholder', 'Especialidade do líder')}
              className={styles.leaderInput}
            />
            <Select
              value={leader.strategy}
              onChange={(val) =>
                setLeader((current) => ({
                  ...current,
                  strategy: val as FlowLeaderPolicy['strategy'],
                  preferred_agent_id: val === 'EXISTING' ? current.preferred_agent_id : undefined,
                }))
              }
              options={[
                { value: 'AUTO', label: t('flow.leaderAuto', 'Sugerir automaticamente') },
                { value: 'EXISTING', label: t('flow.leaderExisting', 'Agent cadastrado') },
                { value: 'NONE', label: t('work.noLeader', 'Sem líder') },
              ]}
              selectStyle={{ minWidth: 170 }}
            />
            {leader.strategy === 'EXISTING' && (
              <Select
                value={leader.preferred_agent_id || ''}
                onChange={(val) =>
                  setLeader((current) => ({ ...current, preferred_agent_id: val }))
                }
                placeholder={t('flow.leaderChoose', 'Escolha o líder')}
                options={agents.map((agent) => ({
                  value: agent.id,
                  label: `${agent.name} · ${agent.role}`,
                }))}
                selectStyle={{ minWidth: 160 }}
              />
            )}
            <Button size="sm" onClick={() => void saveLeader()}>
              {t('flow.saveLeader', 'Salvar líder')}
            </Button>
          </div>
        </Card>
      )}
    </Card>
  );
};
