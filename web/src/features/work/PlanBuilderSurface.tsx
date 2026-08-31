import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  ArrowRight,
  Bot,
  ChevronDown,
  ChevronRight,
  Copy,
  FolderGit2,
  GripVertical,
  History,
  Layers,
  Play,
  Pause,
  Plus,
  RotateCcw,
  Route,
  XCircle,
  Sparkles,
  Trash2,
} from 'lucide-react';
import { Badge, Button, Card, InlineAlert, Input } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { Agent, AutonomyContract, ClarificationCheckpoint, MissionRun, MissionSchedule, PlanPhase, PlanRevision, PlanRevisionDiff, Project, WorkPackage, WorkPlan } from '../../types';
import { clarificationFromError, seedProjectPathAnswers, unresolvedBlocking } from './clarificationModel';
import { defaultMissionAutonomyContract } from './missionAutonomyModel';
import { MissionAutonomyCard } from './MissionAutonomyCard';
import { getProviderLock, mergePackages, normalizeWorkPlan, planSuggestionDiff, setProviderLock, splitPackage } from './planBuilderModel';
import { FlowCanvas } from './FlowCanvas';

export const PlanBuilderSurface: React.FC<{
  project: Project;
  agents: Agent[];
  onOpenAgent?: (agent: Agent) => void;
  onClose?: () => void;
}> = ({ project, agents, onOpenAgent }) => {
  const { t } = useTranslation();
  const [plans, setPlans] = useState<WorkPlan[]>([]);
  const [selectedPlan, setSelectedPlan] = useState<WorkPlan | null>(null);
  const [_loading, setLoading] = useState(false);
  const [autoGoal, setAutoGoal] = useState('');
  const [generating, setGenerating] = useState(false);
  const [compiledPrompt, setCompiledPrompt] = useState<any>(null);
  const [activeRun, setActiveRun] = useState<MissionRun | null>(null);
  const [schedules, setSchedules] = useState<MissionSchedule[]>([]);
  const [scheduleAt, setScheduleAt] = useState('');
  const [afterRunId, setAfterRunId] = useState('');
  const [recentRuns, setRecentRuns] = useState<MissionRun[]>([]);
  const [autonomyContract, setAutonomyContract] = useState<AutonomyContract>(() => defaultMissionAutonomyContract());
  const [revisions, setRevisions] = useState<PlanRevision[]>([]);
  const [revisionDiff, setRevisionDiff] = useState<PlanRevisionDiff | null>(null);
  const [dragPackage, setDragPackage] = useState<{ phaseId: string; packageId: string } | null>(null);
  const [runBusy, setRunBusy] = useState(false);
  const [expandedPhases, setExpandedPhases] = useState<Record<string, boolean>>({});
  const [clarification, setClarification] = useState<ClarificationCheckpoint | null>(null);
  const [clarificationAnswers, setClarificationAnswers] = useState<Record<string, string>>({});
  const [planError, setPlanError] = useState('');
  const [resources, setResources] = useState<any[]>([]);
  const [pendingAIPlan, setPendingAIPlan] = useState<{ plan: WorkPlan; diff: ReturnType<typeof planSuggestionDiff> } | null>(null);

  const loadPlans = useCallback(async () => {
    try {
      setLoading(true);
      const list = await nexusApi.getPlans(project.id);
      const normalizedPlans = (list || []).map(normalizeWorkPlan);
      setPlans(normalizedPlans);
      if (normalizedPlans.length > 0 && !selectedPlan) {
        setSelectedPlan(normalizedPlans[0]);
        // Expand first phase by default
        if (normalizedPlans[0].phases.length > 0) {
          setExpandedPhases({ [normalizedPlans[0].phases[0].id]: true });
        }
      }
    } catch (e) {
      console.error('Failed to load work plans:', e);
    } finally {
      setLoading(false);
    }
  }, [project.id, selectedPlan]);

  useEffect(() => {
    loadPlans();
  }, [loadPlans]);

  useEffect(() => {
    nexusApi.getSchedules(project.id).then((res) => setSchedules(Array.isArray(res) ? res : [])).catch((error) => console.error('Failed to load mission schedules:', error));
  }, [project.id]);

  useEffect(() => {
    nexusApi.listResources()
      .then((result) => setResources(((result && result.accounts) || []).filter((account: any) => account.authenticated && account.available)))
      .catch((error) => console.error('Failed to load provider/profile locks:', error));
  }, [project.id]);

  useEffect(() => {
    nexusApi.getRuns()
      .then((runs) => setRecentRuns((Array.isArray(runs) ? runs : []).filter((run) => run.project_id === project.id)))
      .catch((error) => console.error('Failed to load Mission dependencies:', error));
  }, [project.id, activeRun?.id, activeRun?.state]);

  useEffect(() => {
    if (!selectedPlan) { setRevisions([]); return; }
    nexusApi.getPlan(selectedPlan.id).then((detail) => {
      setRevisions((detail && Array.isArray(detail.revisions)) ? detail.revisions : []);
      if (detail && detail.plan && detail.plan.current_revision !== selectedPlan.current_revision) setSelectedPlan(detail.plan);
    }).catch((error) => console.error('Failed to load plan revisions:', error));
  }, [selectedPlan?.id, selectedPlan?.current_revision]);

  useEffect(() => {
    if (!activeRun) return;
    const terminal = new Set(['COMPLETED_VERIFIED', 'FAILED', 'FAILED_NO_PROGRESS', 'FAILED_BUDGET_EXCEEDED', 'FAILED_VERIFICATION', 'CANCELED_BY_USER']);
    if (terminal.has(activeRun.state) || activeRun.state === 'PAUSED' || activeRun.state === 'BLOCKED_NEEDS_USER') return;
    let disposed = false;
    const refresh = async () => {
      try {
        const run = await nexusApi.getRun(activeRun.id);
        if (!disposed) setActiveRun(run);
      } catch (error) {
        if (!disposed) console.error('Failed to refresh mission run:', error);
      }
    };
    const timer = window.setInterval(() => void refresh(), 1500);
    return () => { disposed = true; window.clearInterval(timer); };
  }, [activeRun?.id, activeRun?.state]);

  const selectGeneratedPlan = (plan: WorkPlan) => {
    setPlans((prev) => [plan, ...prev.filter((item) => item.id !== plan.id)]);
    setSelectedPlan(plan);
    setAutoGoal('');
    setClarification(null);
    setClarificationAnswers({});
    if (plan.phases && plan.phases.length > 0) {
      setExpandedPhases({ [plan.phases[0].id]: true });
    }
  };

  const handleGenerateAIPlan = async () => {
    if (!autoGoal.trim()) return;
    try {
      setGenerating(true);
      setPlanError('');
      setClarification(null);
      const plan = await nexusApi.createPlan(project.id, {
        goal: autoGoal,
        auto_plan: true,
      });
      const normalizedPlan = normalizeWorkPlan(plan);
      setPendingAIPlan({ plan: normalizedPlan, diff: planSuggestionDiff(selectedPlan, normalizedPlan) });
    } catch (e) {
      const checkpoint = clarificationFromError(e);
      if (checkpoint) {
        setClarification(checkpoint);
        setClarificationAnswers(seedProjectPathAnswers(checkpoint, project.canonical_path));
      } else {
        setPlanError(e instanceof Error ? e.message : String(e));
      }
    } finally {
      setGenerating(false);
    }
  };

  const handleResolveClarification = async () => {
    if (!clarification) return;
    try {
      setGenerating(true);
      setPlanError('');
      const result = await nexusApi.resolveClarification(clarification.id, clarificationAnswers);
      selectGeneratedPlan(result.plan);
    } catch (e) {
      const checkpoint = clarificationFromError(e);
      if (checkpoint) {
        setClarification(checkpoint);
        setClarificationAnswers((previous) => ({
          ...seedProjectPathAnswers(checkpoint, project.canonical_path),
          ...previous,
        }));
      } else {
        setPlanError(e instanceof Error ? e.message : String(e));
      }
    } finally {
      setGenerating(false);
    }
  };

  const persistPlan = async (nextPlan: WorkPlan, summary: string) => {
    const res = await nexusApi.updatePlan(nextPlan.id, nextPlan, summary);
    const normalizedPlan = normalizeWorkPlan(res.plan);
    setSelectedPlan(normalizedPlan);
    setPlans((prev) => prev.map((p) => (p.id === normalizedPlan.id ? normalizedPlan : p)));
    setRevisions((prev) => [res.revision, ...prev]);
    return res.plan;
  };

  const patchPackage = async (phaseId: string, packageId: string, patch: Partial<WorkPackage>, summary: string) => {
    if (!selectedPlan) return;
    const next = {
      ...selectedPlan,
      phases: selectedPlan.phases.map((phase) => phase.id === phaseId ? {
        ...phase,
        packages: phase.packages.map((pkg) => pkg.id === packageId ? { ...pkg, ...patch } : pkg),
      } : phase),
    };
    try { await persistPlan(next, summary); } catch (error) { console.error('Failed to update work package:', error); }
  };

  const handleDeletePackage = async (phaseId: string, packageId: string) => {
    if (!selectedPlan) return;
    const next = {
      ...selectedPlan,
      phases: selectedPlan.phases.map((phase) => ({
        ...phase,
        packages: phase.packages
          .filter((pkg) => !(phase.id === phaseId && pkg.id === packageId))
          .map((pkg) => ({ ...pkg, dependencies: (pkg.dependencies || []).filter((dep) => dep !== packageId) })),
      })),
    };
    try { await persistPlan(next, t('planBuilder.packageDeleted')); } catch (error) { console.error(error); }
  };

  const handleDropPackage = async (targetPhaseId: string, targetPackageId: string) => {
    if (!selectedPlan || !dragPackage || (dragPackage.phaseId === targetPhaseId && dragPackage.packageId === targetPackageId)) return;
    let moving: WorkPackage | undefined;
    let phases = selectedPlan.phases.map((phase) => {
      const found = phase.packages.find((pkg) => phase.id === dragPackage.phaseId && pkg.id === dragPackage.packageId);
      if (found) moving = found;
      return { ...phase, packages: phase.packages.filter((pkg) => !(phase.id === dragPackage.phaseId && pkg.id === dragPackage.packageId)) };
    });
    if (!moving) return;
    phases = phases.map((phase) => {
      if (phase.id !== targetPhaseId) return phase;
      const index = phase.packages.findIndex((pkg) => pkg.id === targetPackageId);
      const packages = [...phase.packages];
      packages.splice(index < 0 ? packages.length : index, 0, moving!);
      return { ...phase, packages };
    });
    setDragPackage(null);
    try { await persistPlan({ ...selectedPlan, phases }, 'Pacotes reordenados por drag/drop'); } catch (error) { console.error(error); }
  };

  const handleRestoreRevision = async (revision: number) => {
    if (!selectedPlan || revision === selectedPlan.current_revision) return;
    try {
      const result = await nexusApi.restorePlanRevision(selectedPlan.id, revision);
      setSelectedPlan(normalizeWorkPlan(result.plan));
      const normalizedPlan = normalizeWorkPlan(result.plan);
      setSelectedPlan(normalizedPlan);
      setPlans((prev) => prev.map((plan) => plan.id === normalizedPlan.id ? normalizedPlan : plan));
      setRevisions((prev) => [result.revision, ...prev]);
      setRevisionDiff(null);
    } catch (error) { console.error('Failed to restore plan revision:', error); }
  };

  const handleCompareRevision = async (revision: number) => {
    if (!selectedPlan || revision === selectedPlan.current_revision) return;
    try { setRevisionDiff(await nexusApi.diffPlanRevisions(selectedPlan.id, revision, selectedPlan.current_revision)); }
    catch (error) { console.error('Failed to compare revisions:', error); }
  };

  const allPackages = selectedPlan?.phases.flatMap((phase) => phase.packages) || [];

  const handleAddPhase = async () => {
    if (!selectedPlan) return;
    const newPhase: PlanPhase = {
      id: `phase_${Date.now()}`,
      title: `Nova Fase ${selectedPlan.phases.length + 1}`,
      order: selectedPlan.phases.length + 1,
      packages: [
        {
          id: `pkg_${Date.now()}`,
          title: 'Novo Pacote de Trabalho',
          goal: 'Descrever o objetivo técnico deste pacote',
          priority: 'NORMAL',
          status: 'READY',
          dependencies: [],
          role: 'implementer',
          acceptance_criteria: ['Critério de aceitação 1'],
        },
      ],
    };
    const updatedPlan: WorkPlan = {
      ...selectedPlan,
      phases: [...selectedPlan.phases, newPhase],
    };
    try {
      const res = await nexusApi.updatePlan(selectedPlan.id, updatedPlan, 'Fase adicionada');
      const normalizedPlan = normalizeWorkPlan(res.plan);
      setSelectedPlan(normalizedPlan);
      setPlans((prev) => prev.map((p) => (p.id === normalizedPlan.id ? normalizedPlan : p)));
      setExpandedPhases((prev) => ({ ...prev, [newPhase.id]: true }));
    } catch (e) {
      console.error('Failed to add phase:', e);
    }
  };

  const handleAddPackage = async (phaseId: string) => {
    if (!selectedPlan) return;
    const newPkg: WorkPackage = {
      id: `pkg_${Date.now()}`,
      title: 'Novo Pacote de Trabalho',
      goal: 'Descrever objetivo do pacote',
      priority: 'NORMAL',
      status: 'READY',
      dependencies: [],
      role: 'implementer',
      acceptance_criteria: ['Testes passam com sucesso'],
    };
    const updatedPhases = selectedPlan.phases.map((ph) =>
      ph.id === phaseId ? { ...ph, packages: [...ph.packages, newPkg] } : ph
    );
    const updatedPlan: WorkPlan = {
      ...selectedPlan,
      phases: updatedPhases,
    };
    try {
      const res = await nexusApi.updatePlan(selectedPlan.id, updatedPlan, 'Pacote adicionado');
      const normalizedPlan = normalizeWorkPlan(res.plan);
      setSelectedPlan(normalizedPlan);
      setPlans((prev) => prev.map((p) => (p.id === normalizedPlan.id ? normalizedPlan : p)));
    } catch (e) {
      console.error('Failed to add package:', e);
    }
  };

  const handleSplitPackage = async (phaseId: string, packageId: string) => {
    if (!selectedPlan) return;
    try {
      const next = splitPackage(selectedPlan, phaseId, packageId, `pkg_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`);
      await persistPlan(next, 'Pacote dividido preservando dependências DAG');
    } catch (error) {
      setPlanError(error instanceof Error ? error.message : String(error));
    }
  };

  const handleMergeNextPackage = async (phaseId: string, packageId: string) => {
    if (!selectedPlan) return;
    const phase = selectedPlan.phases.find((item) => item.id === phaseId);
    const index = phase?.packages.findIndex((item) => item.id === packageId) ?? -1;
    const nextPackage = phase && index >= 0 ? phase.packages[index + 1] : undefined;
    if (!nextPackage) return;
    try {
      await persistPlan(mergePackages(selectedPlan, phaseId, packageId, nextPackage.id), 'Pacotes mesclados e dependências reescritas');
    } catch (error) {
      setPlanError(error instanceof Error ? error.message : String(error));
    }
  };

  const handleProviderLock = async (phaseId: string, pkg: WorkPackage, value: string) => {
    const [provider = '', profile = ''] = value ? value.split('::') : ['', ''];
    try {
      await patchPackage(phaseId, pkg.id, { task_requirements: setProviderLock(pkg.task_requirements, provider, profile) }, provider ? `Provider lock: ${provider}/${profile}` : 'Provider lock removido');
    } catch (error) {
      setPlanError(error instanceof Error ? error.message : String(error));
    }
  };

  const handleApplyAISuggestion = () => {
    if (!pendingAIPlan) return;
    selectGeneratedPlan(pendingAIPlan.plan);
    setPendingAIPlan(null);
  };

  const handleRejectAISuggestion = async () => {
    if (!pendingAIPlan) return;
    try { await nexusApi.deletePlan(pendingAIPlan.plan.id); } catch (error) { console.error('Failed to discard AI plan suggestion:', error); }
    setPendingAIPlan(null);
  };

  const handleCompilePrompt = async (packageId: string, phaseId: string) => {
    if (!selectedPlan) return;
    try {
      const res = await nexusApi.compilePackagePrompt(selectedPlan.id, packageId, phaseId);
      setCompiledPrompt(res);
    } catch (e) {
      console.error('Failed to compile prompt:', e);
    }
  };

  const handleLaunchRun = async () => {
    if (!selectedPlan) return;
    try {
      const run = await nexusApi.runPlan(selectedPlan.id, { contract: autonomyContract, autonomous: true, approvedRevision: selectedPlan.current_revision });
      setActiveRun(run);
    } catch (e) {
      console.error('Failed to launch run:', e);
    }
  };

  const handleTakeControl = async () => {
    if (!activeRun) return;
    try {
      setRunBusy(true);
      const run = await nexusApi.takeControlRun(activeRun.id, 'manual takeover from Workspace OS');
      setActiveRun(run);
      const current = run.package_runs[run.current_pkg_index];
      if (current?.assigned_agent && onOpenAgent) {
        const detail = await nexusApi.getAgent(current.assigned_agent);
        onOpenAgent(detail.agent);
      }
    } catch (e) {
      console.error('Failed to take control of mission:', e);
    } finally {
      setRunBusy(false);
    }
  };

  const handleResumeRun = async () => {
    if (!activeRun) return;
    try {
      setRunBusy(true);
      setActiveRun(await nexusApi.returnToMission(activeRun.id));
    } catch (e) {
      console.error('Failed to return mission to autopilot:', e);
    } finally {
      setRunBusy(false);
    }
  };

  const handleCancelRun = async () => {
    if (!activeRun) return;
    try {
      setRunBusy(true);
      setActiveRun(await nexusApi.cancelRun(activeRun.id, 'canceled from Workspace OS'));
    } catch (e) {
      console.error('Failed to cancel mission:', e);
    } finally {
      setRunBusy(false);
    }
  };

  const handleScheduleAt = async () => {
    if (!selectedPlan || !scheduleAt) return;
    try {
      const item = await nexusApi.schedulePlan(selectedPlan.id, 'AT', { scheduledFor: new Date(scheduleAt).toISOString(), contract: autonomyContract, approvedRevision: selectedPlan.current_revision });
      setSchedules((prev) => [item, ...prev]);
      setScheduleAt('');
    } catch (e) {
      console.error('Failed to schedule mission:', e);
    }
  };

  const handleWhenResources = async () => {
    if (!selectedPlan) return;
    try {
      const item = await nexusApi.schedulePlan(selectedPlan.id, 'WHEN_RESOURCES', { contract: autonomyContract, approvedRevision: selectedPlan.current_revision });
      setSchedules((prev) => [item, ...prev]);
    } catch (e) {
      console.error('Failed to schedule mission for resource availability:', e);
    }
  };

  const handleAfterRun = async () => {
    if (!selectedPlan || !afterRunId) return;
    try {
      const item = await nexusApi.schedulePlan(selectedPlan.id, 'AFTER_RUN', { afterRunId, contract: autonomyContract, approvedRevision: selectedPlan.current_revision });
      setSchedules((prev) => [item, ...prev]);
      setAfterRunId('');
    } catch (e) {
      console.error('Failed to schedule dependent Mission:', e);
    }
  };


  return (
    <div className="nx-surface-scroll nx-plan-builder" style={{ padding: '24px', maxWidth: '1400px', margin: '0 auto' }}>
      <div className="nx-page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <span className="nx-eyebrow" style={{ display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--color-brand)' }}>
            <Route size={14} /> NEXUS INTELLIGENCE & WORKPLANS
          </span>
          <h1 style={{ fontSize: '24px', fontWeight: 600, margin: '4px 0' }}>Visual Plan Builder & Autonomy Runner</h1>
          <p style={{ color: 'var(--color-text-muted)' }}>
            Decomposição estruturada de objetivos em WorkPackages auditáveis com compilação determinística e portões de verificação.
          </p>
        </div>
        <Badge tone="brand">
          <FolderGit2 size={12} style={{ marginRight: '4px' }} /> {project.name}
        </Badge>
      </div>

      {/* AI Intent Decomposer Bar */}
      <Card style={{ marginBottom: '24px', padding: '16px', border: '1px solid var(--color-brand-border, #3b82f640)', background: 'var(--color-surface-elevated, #18181b)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px', fontWeight: 600 }}>
          <Sparkles size={16} style={{ color: 'var(--color-brand)' }} />
          <span>Nexus Intent Decomposer (AI Planning)</span>
        </div>
        <div style={{ display: 'flex', gap: '12px' }}>
          <Input
            value={autoGoal}
            onChange={setAutoGoal}
            placeholder="Ex: Implementar autenticação JWT RS256 com middleware, migrations SQLite e testes com race detector..."
            style={{ flex: 1 }}
          />
          <Button tone="brand" disabled={!autoGoal.trim() || generating} onClick={handleGenerateAIPlan}>
            <Sparkles size={14} /> {generating ? 'Decompondo...' : 'Gerar Plano Estruturado'}
          </Button>
        </div>
      </Card>

      {pendingAIPlan && (
        <Card style={{ marginBottom: '16px', padding: '16px', border: '1px solid var(--color-brand-border)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, alignItems: 'flex-start' }}>
            <div>
              <strong>AI plan suggestion — review required</strong>
              <div style={{ marginTop: 6, fontSize: 12, color: 'var(--color-text-muted)' }}>
                +{pendingAIPlan.diff.added.length} added · -{pendingAIPlan.diff.removed.length} removed · ~{pendingAIPlan.diff.changed.length} changed.
                The current approved plan has not been modified.
              </div>
              {pendingAIPlan.diff.added.length > 0 && <div style={{ marginTop: 4, fontSize: 11 }}>Added: {pendingAIPlan.diff.added.join(', ')}</div>}
              {pendingAIPlan.diff.removed.length > 0 && <div style={{ marginTop: 4, fontSize: 11 }}>Removed: {pendingAIPlan.diff.removed.join(', ')}</div>}
              {pendingAIPlan.diff.changed.length > 0 && <div style={{ marginTop: 4, fontSize: 11 }}>Changed: {pendingAIPlan.diff.changed.join(', ')}</div>}
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <Button onClick={() => void handleRejectAISuggestion()}>Reject</Button>
              <Button tone="brand" onClick={handleApplyAISuggestion}>Apply suggestion</Button>
            </div>
          </div>
        </Card>
      )}

      {planError && (
        <InlineAlert tone="danger" title="Nexus Intelligence could not generate the plan">
          {planError}. Direct AI sessions remain available; configure a real Intelligence provider in Settings to use AI Planning.
        </InlineAlert>
      )}

      {clarification && (
        <Card style={{ margin: '16px 0 24px', padding: '16px' }}>
          <div style={{ marginBottom: 12 }}>
            <strong>Confirmação necessária antes de gerar o plano</strong>
            <p style={{ margin: '6px 0 0', color: 'var(--color-text-muted)' }}>
              O Nexus pausa quando falta um fato importante. Revise as respostas abaixo; elas serão registradas no WorkPlan.
            </p>
            {project.canonical_path && unresolvedBlocking(clarification).some((item) => clarificationAnswers[item.key] === project.canonical_path) && (
              <p className="nx-plan-builder__clarification-hint">
                Caminho do repositório preenchido a partir do Projeto atual. Você pode revisá-lo antes de continuar.
              </p>
            )}
          </div>
          <div style={{ display: 'grid', gap: 14 }}>
            {unresolvedBlocking(clarification).map((item) => (
              <label key={item.key} style={{ display: 'grid', gap: 6 }}>
                <span><strong>{item.question}</strong></span>
                {item.rationale && <small style={{ color: 'var(--color-text-muted)' }}>{item.rationale}</small>}
                {item.suggested_options && item.suggested_options.length > 0 ? (
                  <select className="nx-select"
                    value={clarificationAnswers[item.key] || ''}
                    onChange={(event) => setClarificationAnswers((prev) => ({ ...prev, [item.key]: event.target.value }))}
                  >
                    <option value="">Selecione uma opção…</option>
                    {item.suggested_options.map((option) => <option key={option} value={option}>{option}</option>)}
                  </select>
                ) : (
                  <Input
                    value={clarificationAnswers[item.key] || ''}
                    onChange={(value) => setClarificationAnswers((prev) => ({ ...prev, [item.key]: value }))}
                    placeholder="Informe uma resposta"
                  />
                )}
              </label>
            ))}
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 14 }}>
            <Button
              tone="brand"
              disabled={generating || unresolvedBlocking(clarification).some((item) => !(clarificationAnswers[item.key] || '').trim())}
              onClick={() => void handleResolveClarification()}
            >
              <ArrowRight size={14} /> Continuar planejamento
            </Button>
          </div>
        </Card>
      )}

      <FlowCanvas plan={selectedPlan} />

      {/* Main Two-Column Layout: Plan Hierarchy & Inspector/Runner */}
      <div className="nx-plan-builder__columns">
        {/* Left Column: Plan Hierarchy */}
        <div>
          {/* Plan Selector & Actions */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <select className="nx-select nx-plan-selector"
                value={selectedPlan?.id || ''}
                onChange={(e) => {
                  const p = plans.find((x) => x.id === e.target.value);
                  if (p) setSelectedPlan(p);
                }}
                style={{
                  padding: '8px 12px',
                  borderRadius: '6px',
                  background: 'var(--color-surface)',
                  color: 'inherit',
                  border: '1px solid var(--color-border)',
                }}
              >
                {plans.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.title} (Rev {p.current_revision})
                  </option>
                ))}
              </select>
              {selectedPlan && (
                <Badge tone="default">
                  <History size={12} style={{ marginRight: '4px' }} /> Rev {selectedPlan.current_revision}
                </Badge>
              )}
            </div>

            <div style={{ display: 'flex', gap: '8px' }}>
              <Button onClick={handleAddPhase}>
                <Plus size={14} /> Adicionar Fase
              </Button>
              <Button tone="brand" disabled={!selectedPlan} onClick={handleLaunchRun}>
                <Play size={14} /> Approve Contract & Run Mission
              </Button>
            </div>
          </div>

          {/* Phases & Packages Tree */}
          {selectedPlan?.phases?.map((phase, phaseIdx) => {
            const isExpanded = expandedPhases[phase.id] ?? true;
            return (
              <Card key={phase.id} style={{ marginBottom: '16px', padding: '16px' }}>
                <div
                  style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }}
                  onClick={() =>
                    setExpandedPhases((prev) => ({
                      ...prev,
                      [phase.id]: !isExpanded,
                    }))
                  }
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontWeight: 600 }}>
                    {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                    <Layers size={16} style={{ color: 'var(--color-brand)' }} />
                    <span>Fase {phaseIdx + 1}: {phase.title}</span>
                    <Badge tone="default">{phase.packages.length} pacotes</Badge>
                  </div>
                  <Button
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleAddPackage(phase.id);
                    }}
                  >
                    <Plus size={12} /> Pacote
                  </Button>
                </div>

                {isExpanded && (
                  <div style={{ marginTop: '16px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
                    {phase.packages.map((pkg) => (
                      <div
                        key={pkg.id}
                        draggable
                        onDragStart={() => setDragPackage({ phaseId: phase.id, packageId: pkg.id })}
                        onDragOver={(event) => event.preventDefault()}
                        onDrop={() => void handleDropPackage(phase.id, pkg.id)}
                        style={{
                          padding: '14px',
                          borderRadius: '8px',
                          border: '1px solid var(--color-border)',
                          background: 'var(--color-surface)',
                        }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                          <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
                            <GripVertical size={15} style={{ color: 'var(--color-text-muted)', cursor: 'grab', marginTop: 2 }} />
                            <div>
                            <strong style={{ fontSize: '15px' }}>{pkg.title}</strong>
                            <div style={{ display: 'flex', gap: '6px', marginTop: '4px' }}>
                              <Badge tone={pkg.priority === 'CRITICAL' ? 'danger' : pkg.priority === 'HIGH' ? 'warning' : 'default'}>
                                {pkg.priority}
                              </Badge>
                              <Badge tone="default">
                                <Bot size={12} style={{ marginRight: '3px' }} /> {pkg.role}
                              </Badge>
                            </div>
                            </div>
                          </div>
                          <div style={{ display: 'flex', gap: 6 }}>
                            <Button size="sm" onClick={() => handleCompilePrompt(pkg.id, phase.id)}>
                            <Copy size={12} /> Compilar Prompt
                          </Button>
                            <Button size="sm" disabled={(pkg.acceptance_criteria || []).length < 2} onClick={() => void handleSplitPackage(phase.id, pkg.id)}>Split</Button>
                            <Button size="sm" disabled={phase.packages[phase.packages.findIndex((item) => item.id === pkg.id) + 1] == null} onClick={() => void handleMergeNextPackage(phase.id, pkg.id)}>Merge next</Button>
                            <Button size="sm" onClick={() => void handleDeletePackage(phase.id, pkg.id)}><Trash2 size={12} /></Button>
                          </div>
                        </div>

                        <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', margin: '6px 0' }}>{pkg.goal}</p>

                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 8, marginTop: 10, fontSize: 11 }}>
                          <label style={{ display: 'grid', gap: 4 }}>Priority
                            <select className="nx-select" value={pkg.priority} onChange={(event) => void patchPackage(phase.id, pkg.id, { priority: event.target.value as WorkPackage['priority'] }, 'Prioridade alterada')}>
                              {['CRITICAL','HIGH','NORMAL','LOW'].map((value) => <option key={value} value={value}>{value}</option>)}
                            </select>
                          </label>
                          <label style={{ display: 'grid', gap: 4 }}>Agent allocation
                            <select className="nx-select" value={pkg.agent_allocation || ''} onChange={(event) => void patchPackage(phase.id, pkg.id, { agent_allocation: event.target.value }, 'Agent allocation alterado')}>
                              <option value="">Auto scheduler</option>
                              {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
                            </select>
                          </label>
                          <label style={{ display: 'grid', gap: 4 }}>Provider/profile lock
                            <select className="nx-select"
                              value={(() => { const lock = getProviderLock(pkg.task_requirements); return lock.provider ? `${lock.provider}::${lock.profile}` : ''; })()}
                              onChange={(event) => void handleProviderLock(phase.id, pkg, event.target.value)}
                            >
                              <option value="">Auto scheduler</option>
                              {resources.map((account: any) => <option key={account.id || `${account.provider}:${account.profile}`} value={`${account.provider}::${account.profile}`}>{account.display_name || account.provider} · {account.profile}</option>)}
                            </select>
                          </label>
                          <label style={{ display: 'grid', gap: 4 }}>Parallel group
                            <input value={pkg.parallel_group || ''} onChange={(event) => void patchPackage(phase.id, pkg.id, { parallel_group: event.target.value }, 'Parallel group alterado')} />
                          </label>
                          <label style={{ display: 'grid', gap: 4 }}>Dependencies
                            <select className="nx-select" multiple value={pkg.dependencies || []} onChange={(event) => void patchPackage(phase.id, pkg.id, { dependencies: Array.from(event.target.selectedOptions).map((option) => option.value) }, 'Dependências alteradas')}>
                              {allPackages.filter((candidate) => candidate.id !== pkg.id).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.title}</option>)}
                            </select>
                          </label>
                        </div>

                        {pkg.acceptance_criteria && pkg.acceptance_criteria.length > 0 && (
                          <div style={{ marginTop: '8px', fontSize: '12px' }}>
                            <span style={{ fontWeight: 600, color: 'var(--color-text-muted)' }}>Critérios de Aceitação:</span>
                            <ul style={{ margin: '4px 0 0 16px', padding: 0 }}>
                              {pkg.acceptance_criteria.map((c, i) => (
                                <li key={i}>{c}</li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </Card>
            );
          })}

          {(!selectedPlan || selectedPlan.phases.length === 0) && (
            <Card style={{ padding: '32px', textAlign: 'center', color: 'var(--color-text-muted)' }}>
              <Route size={32} style={{ margin: '0 auto 12px', opacity: 0.5 }} />
              <p>{t('planBuilder.emptyPlan')}</p>
            </Card>
          )}
        </div>

        {/* Right Column: Prompt Compiler Preview & Live Mission Runner */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <Card style={{ padding: '16px' }}>
            <div style={{ fontWeight: 600, marginBottom: 10 }}><History size={14} /> {t('planBuilder.revisionHistory')}</div>
            <div style={{ display: 'grid', gap: 6 }}>
              {(revisions || []).slice(0, 6).map((revision) => (
                <div key={revision.id} style={{ display: 'grid', gridTemplateColumns: '1fr auto auto', gap: 6, alignItems: 'center', fontSize: 11 }}>
                  <span>Rev {revision.revision} · {revision.change_summary}</span>
                  <Button size="sm" disabled={revision.revision === selectedPlan?.current_revision} onClick={() => void handleCompareRevision(revision.revision)}>{t('planBuilder.diff')}</Button>
                  <Button size="sm" disabled={revision.revision === selectedPlan?.current_revision} onClick={() => void handleRestoreRevision(revision.revision)}>{t('planBuilder.restore')}</Button>
                </div>
              ))}
              {revisionDiff && (
                <div style={{ padding: 8, border: '1px solid var(--color-border)', borderRadius: 6, fontSize: 11 }}>
                  {t('planBuilder.diffSummary', {
                    from: revisionDiff.from_revision,
                    to: revisionDiff.to_revision,
                    added: revisionDiff.added_packages.length,
                    removed: revisionDiff.removed_packages.length,
                    changed: revisionDiff.changed_packages.length,
                  })}
                </div>
              )}
            </div>
          </Card>

          <MissionAutonomyCard value={autonomyContract} onChange={setAutonomyContract} />

          <Card style={{ padding: '16px' }}>
            <div style={{ fontWeight: 600, marginBottom: 10 }}>{t('planBuilder.scheduling')}</div>
            <div style={{ display: 'grid', gap: 8 }}>
              <input
                type="datetime-local"
                value={scheduleAt}
                onChange={(event) => setScheduleAt(event.target.value)}
                style={{ padding: '8px 10px', borderRadius: 6, border: '1px solid var(--color-border)', background: 'var(--color-surface)', color: 'inherit' }}
              />
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                <Button disabled={!selectedPlan || !scheduleAt} onClick={handleScheduleAt}>{t('planBuilder.scheduleDateTime')}</Button>
                <Button disabled={!selectedPlan} onClick={handleWhenResources}>{t('planBuilder.whenResourcesFree')}</Button>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 8 }}>
                <select className="nx-select"
                  value={afterRunId}
                  onChange={(event) => setAfterRunId(event.target.value)}
                  aria-label="Mission dependency"
                  style={{ padding: '8px 10px', borderRadius: 6, border: '1px solid var(--color-border)', background: 'var(--color-surface)', color: 'inherit', minWidth: 0 }}
                >
                  <option value="">{t('planBuilder.afterMission')}</option>
                  {(recentRuns || []).filter((run) => run && run.id !== activeRun?.id).map((run) => (
                    <option key={run.id} value={run.id}>{(run.id || '').slice(0, 12)} · {run.state}</option>
                  ))}
                  {activeRun ? <option value={activeRun.id}>{(activeRun.id || '').slice(0, 12)} · {t('planBuilder.currentRun', { state: activeRun.state })}</option> : null}
                </select>
                <Button disabled={!selectedPlan || !afterRunId} onClick={handleAfterRun}>{t('planBuilder.runAfterMission')}</Button>
              </div>
              {(schedules || []).slice(0, 3).map((item) => (
                <div key={item.id} style={{ fontSize: 11, color: 'var(--color-text-muted)', display: 'flex', justifyContent: 'space-between' }}>
                  <span>{item.mode}{item.scheduled_for ? ` · ${new Date(item.scheduled_for).toLocaleString()}` : ''}</span>
                  <Badge tone={item.status === 'FAILED' ? 'danger' : item.status === 'COMPLETED' ? 'success' : 'default'}>{item.status}</Badge>
                </div>
              ))}
            </div>
          </Card>

          {/* Active Mission Runner Panel */}
          {activeRun && (
            <Card style={{ padding: '16px', border: '1px solid var(--color-brand)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                <span style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <Play size={14} style={{ color: 'var(--color-brand)' }} /> Autonomous Runner
                </span>
                <Badge tone={activeRun.state === 'COMPLETED_VERIFIED' ? 'success' : activeRun.state.startsWith('FAILED') || activeRun.state === 'BLOCKED_NEEDS_USER' ? 'danger' : activeRun.state === 'PAUSED' ? 'warning' : 'brand'}>
                  {activeRun.state}
                </Badge>
              </div>

              <div style={{ fontSize: '13px', marginBottom: '12px' }}>
                <div>Progresso: {activeRun.package_runs.filter((pkg) => pkg.state === 'VERIFIED').length} de {activeRun.package_runs.length} pacotes verificados</div>
                <div style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>Snapshot: {activeRun.execution_snapshot_id || '—'} · Rev {activeRun.plan_revision} · Iteração {activeRun.total_iterations}/{activeRun.contract.max_total_iterations}</div>
                <div style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>
                  Tentativas permitidas: {activeRun.contract.max_retries} | Verificação: {activeRun.contract.require_verification ? 'Ativa' : 'Desativada'}
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginBottom: '16px' }}>
                {activeRun.package_runs.map((pr, idx) => (
                  <div
                    key={pr.id}
                    style={{
                      padding: '8px 12px',
                      borderRadius: '6px',
                      background: idx === activeRun.current_pkg_index ? 'var(--color-surface-elevated)' : 'transparent',
                      border: idx === activeRun.current_pkg_index ? '1px solid var(--color-brand-border)' : '1px solid var(--color-border)',
                      fontSize: '12px',
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <span>{pr.title}</span>
                    <Badge tone={pr.state === 'VERIFIED' ? 'success' : pr.state === 'FAILED' ? 'danger' : 'default'}>
                      {pr.state} (T{pr.attempt})
                    </Badge>
                  </div>
                ))}
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                {activeRun.state === 'PAUSED' || activeRun.state === 'BLOCKED_NEEDS_USER' ? (
                  <Button tone="brand" disabled={runBusy} onClick={handleResumeRun}>
                    <RotateCcw size={14} /> Return to Mission
                  </Button>
                ) : (
                  <Button disabled={runBusy || activeRun.state === 'COMPLETED_VERIFIED' || activeRun.state.startsWith('FAILED') || activeRun.state === 'CANCELED_BY_USER'} onClick={handleTakeControl}>
                    <Pause size={14} /> Take Control
                  </Button>
                )}
                <Button disabled={runBusy || activeRun.state === 'COMPLETED_VERIFIED' || activeRun.state === 'CANCELED_BY_USER'} onClick={handleCancelRun}>
                  <XCircle size={14} /> Cancel
                </Button>
              </div>
              <p style={{ fontSize: 11, color: 'var(--color-text-muted)', margin: '10px 0 0' }}>
                The runner advances automatically. Take Control pauses autonomy and opens the current persistent Agent; Return to Mission resumes from the persisted checkpoint.
              </p>
            </Card>
          )}

          {/* Compiled Prompt Inspector */}
          <Card style={{ padding: '16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
              <span style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Copy size={14} /> Prompt Compilado
              </span>
              {compiledPrompt && (
                <Badge tone="default">{compiledPrompt.estimated_tokens} tokens est.</Badge>
              )}
            </div>

            {compiledPrompt ? (
              <div style={{ fontSize: '12px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
                <div>
                  <strong style={{ color: 'var(--color-text-muted)' }}>System Prompt Scoped:</strong>
                  <pre
                    style={{
                      padding: '8px',
                      borderRadius: '4px',
                      background: 'var(--color-surface)',
                      border: '1px solid var(--color-border)',
                      maxHeight: '160px',
                      overflowY: 'auto',
                      whiteSpace: 'pre-wrap',
                      marginTop: '4px',
                    }}
                  >
                    {compiledPrompt.system_prompt}
                  </pre>
                </div>

                <div>
                  <strong style={{ color: 'var(--color-text-muted)' }}>User Prompt (Task & Acceptance):</strong>
                  <pre
                    style={{
                      padding: '8px',
                      borderRadius: '4px',
                      background: 'var(--color-surface)',
                      border: '1px solid var(--color-border)',
                      maxHeight: '140px',
                      overflowY: 'auto',
                      whiteSpace: 'pre-wrap',
                      marginTop: '4px',
                    }}
                  >
                    {compiledPrompt.user_prompt}
                  </pre>
                </div>
              </div>
            ) : (
              <p style={{ fontSize: '13px', color: 'var(--color-text-muted)' }}>
                Selecione &quot;Compilar Prompt&quot; em qualquer pacote de trabalho para inspecionar o escopo determinístico compilado.
              </p>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
};
