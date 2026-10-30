import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { nexusApi } from '../../nexus/api';
import type {
  ClarificationCheckpoint,
  FlowLeaderPolicy,
  FlowPreflightReport,
  MissionRun,
  PlanExecutionPreview,
  PlanRevision,
  Project,
  WorkPlan,
} from '../../types';
import { clarificationFromError } from './clarificationModel';
import {
  expandStepLocalized,
  flowFromWorkPlan,
  removeStepLocalized,
  splitStepLocalized,
  updateStepLocalized,
  validateFlow,
  workPlanFromFlow,
  type FlowDraftModel,
  type FlowStepModel,
} from './flowModel';
import {
  chipsFromPlan,
  composerDraftStorageKey,
  composerHistoryStorageKey,
  flowDraftSessionKey,
  loadStoredFlowDraft,
  persistStoredFlowDraft,
  loadGoalHistory,
  loadStoredGoal,
  persistStoredGoal,
  recordGoalHistory,
  refineStepPatch,
  splitStepIds,
  type FlowStepAction,
  type StudioView,
} from './flowDraftEditorModel';
import { normalizeWorkPlan } from './planBuilderModel';

export { GENERATE_BUDGET_SEC } from './flowDraftEditorModel';
export type { FlowStepAction, StudioView } from './flowDraftEditorModel';

/**
 * Encapsulates the Flow draft editor lifecycle: goal capture, manual draft
 * creation, leader policy, preflight, local step edits and Approve & Run. Draft
 * edits never execute work until {@link launchRun} persists and runs the plan.
 * Both complete prompts and Composer artifacts can produce an inspectable Flow DAG.
 */
export function useFlowDraftEditor(options: {
  project: Project;
  onRunCreated?: (run: MissionRun) => void;
  onPlanDeleted?: () => void;
  initialGoal?: string;
  initialPlan?: WorkPlan | null;
}) {
  const { project, onRunCreated, onPlanDeleted, initialGoal = '', initialPlan = null } = options;
  const { t } = useTranslation();

  const [selectedPlan, setSelectedPlan] = useState<WorkPlan | null>(null);
  const [deletingPlan, setDeletingPlan] = useState(false);
  const [autoGoal, setAutoGoal] = useState('');
  const [generating, setGenerating] = useState(false);
  const [generateStage, setGenerateStage] = useState('');
  const [generateProvider, setGenerateProvider] = useState('');
  const [generateElapsedSec, setGenerateElapsedSec] = useState(0);
  const [generateHeartbeat, setGenerateHeartbeat] = useState(false);
  const generateAbortRef = useRef<AbortController | null>(null);
  const [revisions, setRevisions] = useState<PlanRevision[]>([]);
  const [runBusy, setRunBusy] = useState(false);
  const [clarification, setClarification] = useState<ClarificationCheckpoint | null>(null);
  const [clarificationAnswers, setClarificationAnswers] = useState<Record<string, string>>({});
  const [planError, setPlanError] = useState('');
  const [flowDraft, setFlowDraft] = useState<FlowDraftModel | null>(null);
  const [selectedStepId, setSelectedStepId] = useState('');
  const [flowNotice, setFlowNotice] = useState('');
  const [stepComparison, setStepComparison] = useState('');
  const [goalChips, setGoalChips] = useState<string[]>([]);
  const [leader, setLeader] = useState<FlowLeaderPolicy>({
    role: 'orchestrator',
    strategy: 'AUTO',
  });
  const [preflightReport, setPreflightReport] = useState<FlowPreflightReport | null>(null);
  const [preflightBusy, setPreflightBusy] = useState(false);
  const [executionPreview, setExecutionPreview] = useState<PlanExecutionPreview | null>(null);
  const [executionPreviewBusy, setExecutionPreviewBusy] = useState(false);
  const [pendingCostConfirmationToken, setPendingCostConfirmationToken] = useState('');
  const [studioView, setStudioView] = useState<StudioView>('pipeline');

  const composerDraftKey = composerDraftStorageKey(project.id);
  const composerHistoryKey = composerHistoryStorageKey(project.id);
  // Tracks which plan the in-memory draft was seeded for. The persist effect
  // must not write a stale previous-plan draft under a newly selected plan id
  // in the transition render before this sync effect re-seeds the draft.
  const seededPlanRef = useRef('');

  useEffect(() => {
    const saved = loadStoredGoal(composerDraftKey);
    if (saved) setAutoGoal((current) => current || saved);
  }, [composerDraftKey]);

  useEffect(() => {
    const history = loadGoalHistory(composerHistoryKey);
    if (history.length) setGoalChips((current) => (current.length ? current : history.slice(0, 5)));
  }, [composerHistoryKey]);

  useEffect(() => {
    persistStoredGoal(composerDraftKey, autoGoal.trim());
  }, [autoGoal, composerDraftKey]);

  useEffect(() => {
    if (initialGoal.trim()) setAutoGoal((current) => current || initialGoal.trim());
  }, [initialGoal]);

  const selectedPlanId = selectedPlan?.id;
  const selectedPlanRevision = selectedPlan?.current_revision;

  useEffect(() => {
    setExecutionPreview(null);
    setPendingCostConfirmationToken('');
  }, [selectedPlanId, selectedPlanRevision]);

  useEffect(() => {
    if (!selectedPlanId) {
      setRevisions([]);
      return;
    }
    let cancelled = false;
    nexusApi
      .getPlan(selectedPlanId)
      .then((detail) => {
        if (cancelled) return;
        setRevisions(detail.revisions || []);
        setSelectedPlan((current) => {
          if (!current || current.id !== detail.plan.id) return current;
          if (current.current_revision === detail.plan.current_revision) return current;
          return normalizeWorkPlan(detail.plan);
        });
      })
      .catch((error) => console.error('Failed to load plan revisions:', error));
    return () => {
      cancelled = true;
    };
  }, [selectedPlanId, selectedPlanRevision]);

  useEffect(() => {
    if (!selectedPlanId) return;
    let cancelled = false;
    nexusApi
      .getFlowLeader(selectedPlanId)
      .then((policy) => {
        if (cancelled) return;
        setLeader((current) => {
          const role = policy.role?.trim() || current.role;
          const strategy = policy.strategy || current.strategy;
          if (
            current.role === role &&
            current.strategy === strategy &&
            current.preferred_agent_id === policy.preferred_agent_id
          ) {
            return current;
          }
          return {
            ...current,
            ...policy,
            role,
            strategy,
          };
        });
      })
      .catch(() => {
        if (!cancelled) setLeader({ role: 'orchestrator', strategy: 'AUTO' });
      });
    return () => {
      cancelled = true;
    };
  }, [selectedPlanId]);

  useEffect(() => {
    if (!selectedPlan) {
      seededPlanRef.current = '';
      setFlowDraft(null);
      setSelectedStepId('');
      return;
    }
    seededPlanRef.current = selectedPlan.id;
    const next = flowFromWorkPlan(selectedPlan);
    // Restore unsaved step edits that survived a surface unmount (product-surface
    // isolation) when they still apply to this exact plan revision.
    const stored = loadStoredFlowDraft(flowDraftSessionKey(project.id, selectedPlan.id));
    const restored =
      stored &&
      stored.planId === selectedPlan.id &&
      stored.revision === selectedPlan.current_revision &&
      stored.draft &&
      typeof stored.draft === 'object' &&
      Array.isArray((stored.draft as FlowDraftModel).steps) &&
      (stored.draft as FlowDraftModel).steps.every(
        (step) =>
          step &&
          typeof step === 'object' &&
          typeof (step as FlowStepModel).id === 'string' &&
          typeof (step as FlowStepModel).title === 'string',
      )
        ? (stored.draft as FlowDraftModel)
        : null;
    if (restored) {
      setFlowDraft((current) =>
        current && JSON.stringify(current) === JSON.stringify(restored) ? current : restored,
      );
      setSelectedStepId((current) =>
        current && restored.steps.some((step) => step.id === current)
          ? current
          : restored.steps[0]?.id || '',
      );
      return;
    }
    setFlowDraft((current) =>
      current && JSON.stringify(current) === JSON.stringify(next) ? current : next,
    );
    setSelectedStepId((current) =>
      current && next.steps.some((step) => step.id === current) ? current : next.steps[0]?.id || '',
    );
  }, [selectedPlan, project.id]);

  const selectGeneratedPlan = useCallback(
    (plan: WorkPlan) => {
      const next = normalizeWorkPlan(plan);
      setSelectedPlan(next);
      setGoalChips(chipsFromPlan(next, autoGoal || next.title || next.description || ''));
      setClarification(null);
      setClarificationAnswers({});
    },
    [autoGoal],
  );

  const deleteSelectedFlow = useCallback(async () => {
    if (!selectedPlan || deletingPlan || generating || runBusy) return;
    setDeletingPlan(true);
    setPlanError('');
    const deletingId = selectedPlan.id;
    try {
      await nexusApi.deletePlan(deletingId, selectedPlan.current_revision);
      persistStoredFlowDraft(flowDraftSessionKey(project.id, deletingId), null);
      try {
        const key = `iapro:nexus:flow-draft:${project.id}`;
        if (window.sessionStorage.getItem(key) === deletingId)
          window.sessionStorage.removeItem(key);
      } catch {
        /* optional local session storage */
      }
      setSelectedPlan(null);
      setFlowDraft(null);
      setSelectedStepId('');
      setRevisions([]);
      setPreflightReport(null);
      setExecutionPreview(null);
      setPendingCostConfirmationToken('');
      setClarification(null);
      setFlowNotice(t('flow.deleted', 'Flow removido. Você pode iniciar outro planejamento.'));
      onPlanDeleted?.();
    } catch (error) {
      setPlanError(error instanceof Error ? error.message : String(error));
    } finally {
      setDeletingPlan(false);
    }
  }, [selectedPlan, deletingPlan, generating, runBusy, project.id, t, onPlanDeleted]);

  const appliedInitialPlanId = useRef<string | null>(null);
  useEffect(() => {
    if (!initialPlan) return;
    if (appliedInitialPlanId.current === initialPlan.id) return;
    appliedInitialPlanId.current = initialPlan.id;
    selectGeneratedPlan(initialPlan);
  }, [initialPlan, selectGeneratedPlan]);

  const selectedStep = useMemo(
    () => flowDraft?.steps.find((step) => step.id === selectedStepId) || null,
    [flowDraft, selectedStepId],
  );
  const baselineFlow = useMemo(
    () => (selectedPlan ? flowFromWorkPlan(selectedPlan) : null),
    [selectedPlan],
  );
  const flowDirty = Boolean(
    flowDraft && baselineFlow && JSON.stringify(flowDraft) !== JSON.stringify(baselineFlow),
  );
  const flowErrors = useMemo(() => (flowDraft ? validateFlow(flowDraft) : []), [flowDraft]);

  // Keep unsaved step edits alive across product-surface unmounts. Anything
  // persisted to the server clears the session copy via the clean-draft path.
  // The seeded-plan gate skips the transition render where the draft still
  // belongs to the previously selected plan.
  useEffect(() => {
    if (!selectedPlan || seededPlanRef.current !== selectedPlan.id) return;
    const key = flowDraftSessionKey(project.id, selectedPlan.id);
    if (flowDirty && flowDraft) {
      persistStoredFlowDraft(key, {
        planId: selectedPlan.id,
        revision: selectedPlan.current_revision,
        draft: flowDraft,
      });
    } else {
      const stored = loadStoredFlowDraft(key);
      if (stored && stored.planId === selectedPlan.id) persistStoredFlowDraft(key, null);
    }
  }, [flowDirty, flowDraft, selectedPlan, project.id]);

  const cancelGenerate = useCallback(() => {
    generateAbortRef.current?.abort();
    generateAbortRef.current = null;
    setGenerating(false);
    setGenerateStage('');
    setGenerateHeartbeat(false);
    setPlanError(t('flow.generateCanceled', 'Geração cancelada.'));
  }, [t]);

  useEffect(() => {
    if (!generating) {
      setGenerateElapsedSec(0);
      setGenerateHeartbeat(false);
      return;
    }
    const started = Date.now();
    const timer = window.setInterval(() => {
      const elapsed = Math.floor((Date.now() - started) / 1000);
      setGenerateElapsedSec(elapsed);
      if (elapsed >= 15 && generateStage.startsWith('Pedindo')) setGenerateHeartbeat(true);
    }, 500);
    return () => window.clearInterval(timer);
  }, [generating, generateStage]);

  const persistPlan = useCallback(async (nextPlan: WorkPlan, summary: string) => {
    const res = await nexusApi.updatePlan(nextPlan.id, nextPlan, summary);
    const plan = normalizeWorkPlan(res.plan);
    setSelectedPlan(plan);
    setRevisions((prev) => [res.revision, ...prev]);
    return plan;
  }, []);

  const generatePlan = useCallback(async () => {
    const prompt = autoGoal.trim();
    if (!prompt || generating) return;
    const history = recordGoalHistory(composerHistoryKey, prompt);
    setGoalChips((current) => (current.length ? current : history.slice(0, 5)));
    generateAbortRef.current?.abort();
    const controller = new AbortController();
    generateAbortRef.current = controller;
    try {
      setGenerating(true);
      setPlanError('');
      setClarification(null);
      setGenerateHeartbeat(false);
      setGenerateProvider('');
      setGenerateStage(t('flow.stageDecompose', 'Analisando o prompt e organizando atividades…'));
      // Flow accepts a complete prompt directly, without a Composer session.
      // The provider produces an explicit, validated DAG; on failure we keep
      // the input and never create a misleading empty or generic Flow.
      const plan = await nexusApi.generateFlowPlan(project.id, prompt, {
        signal: controller.signal,
      });
      if (controller.signal.aborted) return;
      selectGeneratedPlan(plan);
      setFlowNotice(
        t(
          'flow.generatedReview',
          'Atividades geradas. Revise objetivos, dependências, responsáveis e verificações antes de executar.',
        ),
      );
    } catch (e) {
      if (controller.signal.aborted) {
        setPlanError(t('flow.generateCanceled', 'Geração cancelada.'));
        return;
      }
      const checkpoint = clarificationFromError(e);
      if (checkpoint) {
        setClarification(checkpoint);
      }
      setPlanError(e instanceof Error ? e.message : String(e));
    } finally {
      if (generateAbortRef.current === controller) generateAbortRef.current = null;
      setGenerating(false);
      setGenerateStage('');
      setGenerateHeartbeat(false);
    }
  }, [autoGoal, generating, composerHistoryKey, project.id, selectGeneratedPlan, t]);

  const resolveClarification = useCallback(async () => {
    if (!clarification) return;
    try {
      setGenerating(true);
      setPlanError('');
      const result = await nexusApi.resolveClarification(clarification.id, clarificationAnswers);
      selectGeneratedPlan(result.plan);
    } catch (e) {
      const checkpoint = clarificationFromError(e);
      if (checkpoint) setClarification(checkpoint);
      else setPlanError(e instanceof Error ? e.message : String(e));
    } finally {
      setGenerating(false);
    }
  }, [clarification, clarificationAnswers, selectGeneratedPlan]);

  const createManualPlan = useCallback(async () => {
    try {
      setGenerating(true);
      setPlanError('');
      const title = autoGoal.trim() || t('flow.manualDraftGoal', 'Novo Flow Manual');
      const created = await nexusApi.createPlan(project.id, {
        title,
        goal: title,
        auto_plan: false,
      });
      selectGeneratedPlan(created);
      setFlowNotice(
        t('flow.manualCreated', 'Flow manual criado. Adicione etapas e configure dependências.'),
      );
    } catch (e) {
      setPlanError(e instanceof Error ? e.message : String(e));
    } finally {
      setGenerating(false);
    }
  }, [autoGoal, project.id, selectGeneratedPlan, t]);

  const saveLeader = useCallback(async () => {
    if (!selectedPlan) return;
    try {
      const result = await nexusApi.setFlowLeader(selectedPlan.id, leader);
      setSelectedPlan(normalizeWorkPlan(result.plan));
      setFlowNotice(t('flow.leaderSaved', 'Líder sugerido salvo nesta revisão do Flow.'));
    } catch (err) {
      setPlanError(err instanceof Error ? err.message : String(err));
    }
  }, [leader, selectedPlan, t]);

  const launchRun = useCallback(async () => {
    if (!selectedPlan || !flowDraft) return;
    const errors = validateFlow(flowDraft);
    if (errors.length) {
      setPlanError(errors.join(' '));
      return;
    }
    try {
      setPlanError('');
      setRunBusy(true);
      let approved = selectedPlan;
      if (flowDirty || selectedPlan.status === 'DRAFT') {
        const nextDraft = { ...flowDraft, policyStored: true, status: 'READY' as const };
        setFlowDraft(nextDraft);
        approved = await persistPlan(
          workPlanFromFlow(nextDraft, selectedPlan),
          'Flow approved for execution',
        );
        persistStoredFlowDraft(flowDraftSessionKey(project.id, selectedPlan.id), null);
      }

      // GO is never based on a stale preview. Re-evaluate the exact persisted
      // revision immediately before Mission admission. This is still
      // side-effect-free: Mission remains the only execution authority.
      const preview = await nexusApi.previewPlanExecution(approved.id);
      const previouslyApprovedPreviewToken =
        executionPreview?.plan_id === approved.id &&
        executionPreview.revision === approved.current_revision
          ? executionPreview.approval_token
          : '';
      setExecutionPreview(preview);
      if (preview.revision !== approved.current_revision) {
        const revisionMismatch = t(
          'flow.executionPreviewRevisionMismatch',
          'A revisão mudou durante a preparação. Revise o plano de execução novamente antes de iniciar.',
        );
        setPlanError(revisionMismatch);
        return;
      }
      if (!preview.ready) {
        const blockedNotice = t(
          'flow.executionPreviewBlocked',
          'Execution Preview encontrou bloqueios antes do GO.',
        );
        const noEligibleRoute = t(
          'flow.executionPreviewNoEligibleRoute',
          'Nenhuma combinação elegível de Agent e recurso está disponível para todas as etapas.',
        );
        setPendingCostConfirmationToken('');
        setFlowNotice(blockedNotice);
        setPlanError(preview.blockers?.join(' · ') || noEligibleRoute);
        return;
      }

      if (
        !previouslyApprovedPreviewToken ||
        previouslyApprovedPreviewToken !== preview.approval_token
      ) {
        setPendingCostConfirmationToken(
          preview.cost_confirmation_required ? preview.cost_confirmation_token || '' : '',
        );
        setFlowNotice(
          t(
            'flow.executionPreviewApprovalRequired',
            'O plano de execução foi atualizado. Revise Agent, recurso, modelo, custo e bloqueios; depois clique em GO novamente para aprovar exatamente esta preview.',
          ),
        );
        return;
      }

      const costToken = preview.cost_confirmation_required
        ? preview.cost_confirmation_token || ''
        : '';
      if (preview.cost_confirmation_required) {
        if (!costToken) {
          setPendingCostConfirmationToken('');
          setPlanError(
            t(
              'flow.costConfirmationUnavailable',
              'A política de custo exige confirmação, mas o preview não produziu um token válido. Atualize o plano de execução.',
            ),
          );
          return;
        }
        if (pendingCostConfirmationToken !== costToken) {
          setPendingCostConfirmationToken(costToken);
          setFlowNotice(
            t(
              'flow.costConfirmationRequired',
              'O estado atual de custo exige confirmação explícita. Revise o plano de execução e clique em Confirmar custo e executar.',
            ),
          );
          return;
        }
      } else {
        setPendingCostConfirmationToken('');
      }

      const run = await nexusApi.runPlan(
        approved.id,
        approved.current_revision,
        undefined,
        undefined,
        costToken || undefined,
        preview.approval_token,
      );
      setPendingCostConfirmationToken('');
      onRunCreated?.(run);
      setFlowNotice(
        t('flow.runStarted', 'Flow Run {{id}} iniciado a partir da revisão {{revision}}.', {
          id: run.id,
          revision: run.plan_revision,
        }),
      );
    } catch (e) {
      setPlanError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunBusy(false);
    }
  }, [
    flowDirty,
    flowDraft,
    onRunCreated,
    pendingCostConfirmationToken,
    executionPreview,
    persistPlan,
    selectedPlan,
    t,
    project.id,
  ]);

  const runExecutionPreview = useCallback(async () => {
    if (!selectedPlan) return;
    try {
      setExecutionPreviewBusy(true);
      setPlanError('');
      const preview = await nexusApi.previewPlanExecution(selectedPlan.id);
      setExecutionPreview(preview);
      setPendingCostConfirmationToken(
        preview.cost_confirmation_required ? preview.cost_confirmation_token || '' : '',
      );

      let notice = preview.ready
        ? t(
            'flow.executionPreviewReady',
            'Execution Preview pronto: especialistas e recursos atuais foram avaliados sem iniciar execução.',
          )
        : t('flow.executionPreviewBlocked', 'Execution Preview encontrou bloqueios antes do GO.');
      if (flowDirty) {
        notice = t(
          'flow.executionPreviewSavedRevision',
          'Execution Preview usa a revisão salva {{revision}}; alterações locais ainda não salvas não estão incluídas.',
          { revision: preview.revision },
        );
      }
      setFlowNotice(notice);
    } catch (err) {
      setPlanError(err instanceof Error ? err.message : String(err));
    } finally {
      setExecutionPreviewBusy(false);
    }
  }, [flowDirty, selectedPlan, t]);

  const runPreflight = useCallback(async () => {
    if (!selectedPlan) return;
    try {
      setPreflightBusy(true);
      const report = await nexusApi.preflightFlow(selectedPlan.id);
      setPreflightReport(report);
      setFlowNotice(
        report.ready
          ? t('flow.preflightReady', 'Preflight aprovado: Flow pronto para execução.')
          : t('flow.preflightBlocked', 'Preflight com pendências: resolva os bloqueios.'),
      );
    } catch (err) {
      setPlanError(err instanceof Error ? err.message : String(err));
    } finally {
      setPreflightBusy(false);
    }
  }, [selectedPlan, t]);

  const changeFlowStep = useCallback(
    (patch: Partial<FlowStepModel>) => {
      setFlowDraft((current) =>
        current && selectedStep ? updateStepLocalized(current, selectedStep.id, patch) : current,
      );
      setStepComparison('');
    },
    [selectedStep],
  );

  const runStepAction = useCallback(
    (action: FlowStepAction) => {
      if (!flowDraft || !selectedStep) return;
      if (action === 'REFINE') {
        setFlowDraft(
          updateStepLocalized(flowDraft, selectedStep.id, refineStepPatch(selectedStep)),
        );
        setFlowNotice(
          t('flow.stepRefined', 'Refinado apenas “{{title}}”.', { title: selectedStep.title }),
        );
        return;
      }
      if (action === 'EXPAND') {
        setFlowDraft(expandStepLocalized(flowDraft, selectedStep.id));
        setFlowNotice(
          t('flow.stepExpanded', 'Expandido apenas “{{title}}”.', { title: selectedStep.title }),
        );
        return;
      }
      if (action === 'COMPARE') {
        const previous = revisions.find(
          (revision) => revision.revision < (selectedPlan?.current_revision || 0),
        );
        if (!previous) {
          setStepComparison(t('flow.stepNoPrevRevision', 'Nenhuma revisão anterior disponível.'));
          return;
        }
        try {
          const previousPlan = JSON.parse(previous.snapshot_json) as WorkPlan;
          const prior = flowFromWorkPlan(previousPlan).steps.find(
            (step) => step.id === selectedStep.id,
          );
          setStepComparison(
            prior
              ? `Revision ${previous.revision}\n${JSON.stringify(prior, null, 2)}\n\nCurrent draft\n${JSON.stringify(selectedStep, null, 2)}`
              : t('flow.stepMissingInPrev', 'Esta etapa não existia na revisão anterior.'),
          );
        } catch {
          setStepComparison(
            t('flow.stepDecodeError', 'A revisão anterior não pôde ser decodificada.'),
          );
        }
        return;
      }
      if (action === 'SPLIT') {
        const next = splitStepLocalized(
          flowDraft,
          selectedStep.id,
          splitStepIds(selectedStep.id, Date.now().toString(36)),
        );
        setFlowDraft(next);
        setSelectedStepId(
          next.steps.find((step) => step.id.startsWith(`${selectedStep.id}-a-`))?.id || '',
        );
        setFlowNotice(
          t('flow.stepSplit', 'Etapa “{{title}}” dividida localmente.', {
            title: selectedStep.title,
          }),
        );
        return;
      }
      const next = removeStepLocalized(flowDraft, selectedStep.id);
      setFlowDraft(next);
      setSelectedStepId(next.steps[0]?.id || '');
      setFlowNotice(
        t('flow.stepRemoved', 'Etapa “{{title}}” removida.', { title: selectedStep.title }),
      );
    },
    [flowDraft, revisions, selectedPlan?.current_revision, selectedStep, t],
  );

  const connectStep = useCallback((from: string, to: string) => {
    setFlowDraft((current) => {
      if (!current) return current;
      const target = current.steps.find((step) => step.id === to);
      if (!target) return current;
      const deps = Array.from(new Set([...(target.dependencies || []), from]));
      return updateStepLocalized(current, to, { dependencies: deps });
    });
  }, []);

  return {
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
    clarification,
    clarificationAnswers,
    setClarificationAnswers,
    planError,
    flowDraft,
    setFlowDraft,
    selectedStepId,
    setSelectedStepId,
    flowNotice,
    stepComparison,
    goalChips,
    leader,
    setLeader,
    preflightReport,
    preflightBusy,
    executionPreview,
    executionPreviewBusy,
    costConfirmationPending: Boolean(pendingCostConfirmationToken),
    studioView,
    setStudioView,
    selectedStep,
    flowDirty,
    flowErrors,
    cancelGenerate,
    generatePlan,
    resolveClarification,
    createManualPlan,
    saveLeader,
    launchRun,
    runExecutionPreview,
    runPreflight,
    changeFlowStep,
    runStepAction,
    connectStep,
  };
}

export type FlowDraftEditorController = ReturnType<typeof useFlowDraftEditor>;
