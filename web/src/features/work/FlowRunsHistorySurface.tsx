import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { History, Layers, Play, RefreshCw, Sparkles, Trash2 } from 'lucide-react';
import { Badge, Button, Card, ConfirmDialog, EmptyState, Spinner } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { MissionRun, Project, WorkPlan, Agent } from '../../types';
import { asArray } from '../../lib/safeArray';
import { flowRunStateFromMission } from './flowRunModel';
import { FlowDraftEditor } from './FlowDraftEditor';
import { FLOW_DRAFT_KEY } from './WorkSurface';
import styles from './FlowRunsHistorySurface.module.scss';

type Track = 'drafts' | 'runs';
type RunFilter = 'ALL' | 'ACTIVE' | 'COMPLETED' | 'FAILED';

const SELECTED_PLAN_KEY = (projectId: string) => `nexus:flow:selected-plan:${projectId}`;

const readSelectedPlanId = (projectId: string): string => {
  try {
    return window.sessionStorage.getItem(SELECTED_PLAN_KEY(projectId)) || '';
  } catch {
    return '';
  }
};

const writeSelectedPlanId = (projectId: string, planId: string): void => {
  try {
    if (planId) window.sessionStorage.setItem(SELECTED_PLAN_KEY(projectId), planId);
    else window.sessionStorage.removeItem(SELECTED_PLAN_KEY(projectId));
  } catch {
    /* ignore */
  }
};

export const FlowRunsHistorySurface: React.FC<{
  project: Project;
  agents?: Agent[];
  onOpenRun: (run: MissionRun) => void;
  onOpenComposer?: () => void;
}> = ({ project, agents = [], onOpenRun, onOpenComposer }) => {
  const { t } = useTranslation();
  const [runs, setRuns] = useState<MissionRun[]>([]);
  const [plans, setPlans] = useState<WorkPlan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [track, setTrack] = useState<Track>('drafts');
  const [filterState, setFilterState] = useState<RunFilter>('ALL');
  const [selectedPlan, setSelectedPlan] = useState<WorkPlan | null>(null);
  const [newGoal, setNewGoal] = useState('');
  const [creating, setCreating] = useState(false);
  const [planToDelete, setPlanToDelete] = useState<WorkPlan | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);

  const loadSeqRef = React.useRef(0);
  const load = useCallback(async () => {
    const seq = (loadSeqRef.current += 1);
    const isLatest = () => loadSeqRef.current === seq;
    setLoading(true);
    setError('');
    try {
      const [runList, planList] = await Promise.all([
        nexusApi.getRuns().catch(() => []),
        nexusApi.getPlans(project.id).catch(() => []),
      ]);
      if (!isLatest()) return;
      const nextRuns = asArray(runList).filter((run): run is MissionRun => {
        const item = run as MissionRun;
        return (
          Boolean(item && typeof item === 'object' && item.id) &&
          (!item.project_id || item.project_id === project.id)
        );
      });
      const nextPlans = asArray(planList).filter((plan): plan is WorkPlan => {
        const item = plan as WorkPlan;
        return Boolean(item && typeof item === 'object' && item.id);
      });
      setRuns(nextRuns);
      setPlans(nextPlans);

      let pendingId = '';
      try {
        pendingId =
          window.sessionStorage.getItem(FLOW_DRAFT_KEY(project.id)) ||
          readSelectedPlanId(project.id);
      } catch {
        pendingId = '';
      }
      if (pendingId) {
        const match = nextPlans.find((plan) => plan.id === pendingId);
        if (match) {
          setTrack('drafts');
          setSelectedPlan(match);
        }
        try {
          window.sessionStorage.removeItem(FLOW_DRAFT_KEY(project.id));
        } catch {
          /* ignore */
        }
      }
    } catch (err) {
      if (!isLatest()) return;
      setError(err instanceof Error ? err.message : String(err));
      setRuns([]);
      setPlans([]);
    } finally {
      if (isLatest()) setLoading(false);
    }
  }, [project.id]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    writeSelectedPlanId(project.id, selectedPlan?.id || '');
  }, [project.id, selectedPlan?.id]);

  const deleteFlow = async () => {
    if (!planToDelete || deleteBusy) return;
    const plan = planToDelete;
    setPlanToDelete(null);
    setDeleteBusy(true);
    setError('');
    try {
      await nexusApi.deletePlan(plan.id, plan.current_revision);
      writeSelectedPlanId(project.id, '');
      try {
        if (sessionStorage.getItem(FLOW_DRAFT_KEY(project.id)) === plan.id) {
          sessionStorage.removeItem(FLOW_DRAFT_KEY(project.id));
        }
        sessionStorage.removeItem(`nexus:flow:draft-edits:${project.id}:${plan.id}`);
      } catch { /* optional local cache */ }
      if (selectedPlan?.id === plan.id) setSelectedPlan(null);
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setDeleteBusy(false);
    }
  };

  const draftPlans = useMemo(() => {
    const runPlanIds = new Set(runs.map((run) => run.plan_id).filter(Boolean));
    return plans.filter((plan) => plan.status === 'DRAFT' || !runPlanIds.has(plan.id));
  }, [plans, runs]);

  const filtered = useMemo(() => {
    return runs.filter((run) => {
      const state = flowRunStateFromMission(run.state || '');
      if (filterState === 'ALL') return true;
      if (filterState === 'ACTIVE')
        return (
          state === 'QUEUED' ||
          state === 'READY' ||
          state === 'RUNNING' ||
          state === 'VERIFYING' ||
          state === 'BLOCKED'
        );
      if (filterState === 'COMPLETED') return state === 'COMPLETED';
      if (filterState === 'FAILED') return state === 'FAILED' || state === 'CANCELED';
      return true;
    });
  }, [runs, filterState]);

  const sorted = useMemo(
    () =>
      [...filtered].sort((a, b) =>
        String(b.started_at || b.id).localeCompare(String(a.started_at || a.id)),
      ),
    [filtered],
  );

  if (selectedPlan) {
    return (
      <div className={`nx-surface-scroll ${styles.canvasHost}`}>
        <div className={styles.canvasBar}>
          <Button size="sm" onClick={() => setSelectedPlan(null)}>
            ← {t('flow.backToList', 'Voltar à lista')}
          </Button>
          <strong>{selectedPlan.title || selectedPlan.id}</strong>
        </div>
        <FlowDraftEditor
          project={project}
          agents={agents}
          onRunCreated={(run) => {
            setSelectedPlan(null);
            onOpenRun(run);
          }}
          initialPlan={selectedPlan}
          onPlanDeleted={() => {
            writeSelectedPlanId(project.id, '');
            setSelectedPlan(null);
            void load();
          }}
        />
      </div>
    );
  }

  return (
    <div className={`nx-surface-scroll ${styles.root}`}>
      <div className={styles.header}>
        <div className={styles.title}>
          <History size={16} />
          <h1 className="nx-sr-only">{t('flow.studioTitle', 'Flow Studio')}</h1>
          <strong aria-hidden="true">{t('flow.studioTitle', 'Flow Studio')}</strong>
          <Badge tone="default">{track === 'drafts' ? draftPlans.length : runs.length}</Badge>
        </div>
        <div className={styles.actions}>
          <div
            className={styles.trackToggle}
            role="tablist"
            aria-label={t('flow.tracks', 'Trilhas')}
          >
            <button
              type="button"
              role="tab"
              aria-selected={track === 'drafts'}
              data-active={track === 'drafts' ? 'true' : undefined}
              onClick={() => setTrack('drafts')}
            >
              <Layers size={12} /> {t('flow.tabPlan', 'Este plano')}
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={track === 'runs'}
              data-active={track === 'runs' ? 'true' : undefined}
              onClick={() => setTrack('runs')}
            >
              <Play size={12} /> {t('flow.tabHistory', 'Histórico')}
            </button>
          </div>
          {track === 'runs' && (
            <div className={styles.filterGroup}>
              {(['ALL', 'ACTIVE', 'COMPLETED', 'FAILED'] as const).map((s) => (
                <button
                  key={s}
                  type="button"
                  className={styles.filterChip}
                  data-active={filterState === s ? 'true' : undefined}
                  onClick={() => setFilterState(s)}
                >
                  {s === 'ALL'
                    ? t('flow.filterAll', 'Todos')
                    : s === 'ACTIVE'
                      ? t('flow.filterActive', 'Em andamento')
                      : s === 'COMPLETED'
                        ? t('flow.filterCompleted', 'Concluídos')
                        : t('flow.filterFailed', 'Falhas')}
                </button>
              ))}
            </div>
          )}
          <Button size="sm" onClick={() => void load()} disabled={loading}>
            <RefreshCw size={13} className={loading ? 'nx-spin' : ''} />
            {t('common.refresh')}
          </Button>
          {onOpenComposer && (
            <Button size="sm" tone="brand" onClick={onOpenComposer}>
              <Sparkles size={13} /> {t('flow.elaborateInComposer', 'Elaborar no Composer')}
            </Button>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={Boolean(planToDelete)}
        title={t('flow.deleteDraft', 'Excluir Flow')}
        description={t(
          'flow.deleteDraftConfirm',
          'Excluir "{{title}}" e suas revisões? Esta ação não pode ser desfeita. Flows com execuções ou agendamentos são protegidos.',
          { title: planToDelete?.title || 'Flow sem nome' },
        )}
        confirmLabel={t('flow.deleteDraft', 'Excluir Flow')}
        onCancel={() => setPlanToDelete(null)}
        onConfirm={() => void deleteFlow()}
      />
      {loading && (
        <div className={styles.loadingContainer}>
          <Spinner />
        </div>
      )}
      {error && <div className="nx-inline-error">{error}</div>}

      {!loading && !error && track === 'drafts' && (
        <Card className={styles.newFlowCard}>
          <strong>{t('flow.newFlow', 'New Flow')}</strong>
          <div className={styles.newFlowRow}>
            <input
              className="nx-input"
              value={newGoal}
              onChange={(e) => setNewGoal(e.target.value)}
              placeholder={t('flow.goalPlaceholder', 'What do you want to deliver?')}
              disabled={creating}
            />
            <Button
              tone="brand"
              disabled={!newGoal.trim() || creating}
              onClick={() => {
                setCreating(true);
                void nexusApi
                  .createPlan(project.id, {
                    goal: newGoal.trim(),
                    auto_plan: false,
                    title: newGoal.trim(),
                  })
                  .then((plan) => {
                    setSelectedPlan(plan);
                    setNewGoal('');
                    return load();
                  })
                  .catch((err) => setError(err instanceof Error ? err.message : String(err)))
                  .finally(() => setCreating(false));
              }}
            >
              <Sparkles size={13} />{' '}
              {creating ? t('common.loading', '…') : t('flow.newFlow', 'New Flow')}
            </Button>
            {onOpenComposer ? (
              <Button onClick={onOpenComposer}>
                {t('flow.elaborateInComposer', 'Elaborate in Composer')}
              </Button>
            ) : null}
          </div>
        </Card>
      )}

      {!loading && !error && track === 'drafts' && draftPlans.length === 0 && (
        <EmptyState
          title={t('flow.noDrafts', 'No plan yet')}
          hint={t(
            'flow.noDraftsHint',
            'Describe an outcome and generate a Flow, or deepen it in Composer first.',
          )}
          action={
            onOpenComposer ? (
              <Button tone="brand" onClick={onOpenComposer}>
                {t('flow.openComposer', 'Abrir Composer')}
              </Button>
            ) : undefined
          }
        />
      )}

      {!loading && !error && track === 'runs' && sorted.length === 0 && (
        <EmptyState
          title={t('flow.noRuns', 'Nenhuma execução ainda')}
          hint={t('flow.noRunsHint', 'Aprove um rascunho de Flow para ver o histórico aqui.')}
          action={
            <Button tone="brand" onClick={() => setTrack('drafts')}>
              {t('flow.seeDrafts', 'Ver rascunhos')}
            </Button>
          }
        />
      )}

      {track === 'drafts' && (
        <div className={styles.runsList}>
          {draftPlans.map((plan) => (
            <Card key={plan.id} className={`nx-flow-run-history-card ${styles.planRow}`}>
              <button
                type="button"
                onClick={() => setSelectedPlan(plan)}
                className={styles.runButton}
              >
                <div className={styles.runInfo}>
                  <strong className={styles.runTitle}>
                    <Layers size={14} />
                    {plan.title ||
                      plan.description ||
                      t('flow.untitledDraft', 'Rascunho sem título')}
                  </strong>
                  <small className={styles.runMeta}>
                    {plan.id.slice(-8)} · {plan.status || 'DRAFT'}
                  </small>
                </div>
                <Badge tone="brand">{t('flow.draftBadge', 'RASCUNHO')}</Badge>
              </button>
              <Button
                size="sm"
                tone="danger"
                className={styles.deletePlanAction}
                disabled={deleteBusy || loading}
                onClick={() => setPlanToDelete(plan)}
                aria-label={t('flow.deleteFlowNamed', 'Excluir Flow: {{title}}', {
                  title: plan.title || plan.id,
                })}
              >
                <Trash2 size={13} /> {t('common.delete', 'Excluir')}
              </Button>
            </Card>
          ))}
        </div>
      )}

      {track === 'runs' && (
        <div className={styles.runsList}>
          {sorted.map((run) => {
            const state = flowRunStateFromMission(run.state || '');
            const plan = plans.find((item) => item.id === run.plan_id);
            return (
              <Card key={run.id} className="nx-flow-run-history-card">
                <button type="button" onClick={() => onOpenRun(run)} className={styles.runButton}>
                  <div className={styles.runInfo}>
                    <strong className={styles.runTitle}>
                      <History size={14} />
                      {plan?.title ||
                        plan?.description ||
                        t('flow.runFallback', 'Execução · {{id}}', {
                          id: (run.id || '').slice(-8),
                        })}
                    </strong>
                    <small className={styles.runMeta}>
                      {run.started_at || t('flow.noTime', 'sem horário')} · plano{' '}
                      {(run.plan_id || '').slice(-6) || '—'}
                    </small>
                  </div>
                  <Badge
                    tone={
                      state === 'COMPLETED'
                        ? 'success'
                        : state === 'FAILED' || state === 'CANCELED'
                          ? 'danger'
                          : 'brand'
                    }
                  >
                    {t(`flow.state.${state}`, state)}
                  </Badge>
                </button>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
};
