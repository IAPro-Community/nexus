import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Activity,
  Bot,
  CheckCircle2,
  CircleAlert,
  Clock3,
  FileCode2,
  Pause,
  Play,
  RefreshCw,
  ShieldCheck,
  Square,
  TerminalSquare,
} from 'lucide-react';
import { Badge, Button, Card, InlineAlert, Spinner } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import { useTranslation } from 'react-i18next';
import type {
  Agent,
  ContextCapsule,
  FlowRunEvidence,
  InterventionOption,
  MissionEvidenceBundle,
  MissionRun,
  PackageRun,
  Project,
  WorkPlan,
  WorkReceipt,
  MissionTeamSelection,
} from '../../types';
import { asArray, asStringArray } from '../../lib/safeArray';
import { flowFromWorkPlan } from './flowModel';
import { flowRunStateFromMission, packageRunState, type FlowRunState } from './flowRunModel';
import { createRefreshGuard } from './flowRefreshGuard';

const terminalMissionStates = new Set([
  'COMPLETED_VERIFIED',
  'FAILED',
  'FAILED_NO_PROGRESS',
  'FAILED_BUDGET_EXCEEDED',
  'FAILED_VERIFICATION',
  'CANCELED_BY_USER',
]);

const toneFor = (state: FlowRunState): 'default' | 'brand' | 'success' | 'warning' | 'danger' => {
  if (state === 'COMPLETED') return 'success';
  if (state === 'FAILED' || state === 'CANCELED') return 'danger';
  if (state === 'VERIFYING' || state === 'BLOCKED') return 'warning';
  if (state === 'RUNNING' || state === 'READY') return 'brand';
  return 'default';
};

const providerIssuePattern =
  /provider|quota|budget|credential|auth|rate.?limit|resource|unauthorized|forbidden|insufficient|payment/i;

const readMissionTeamSelection = (plan: WorkPlan | null): MissionTeamSelection | null => {
  const raw = plan?.structured_facts?.['nexus.mission_team.selection'];
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as MissionTeamSelection;
    return parsed.direct ? null : parsed;
  } catch {
    return null;
  }
};

const readSelectedResource = (pkg: PackageRun | undefined) => {
  if (!pkg) return '';
  try {
    const routing = JSON.parse(pkg.routing_decision || '{}') as {
      selected_provider?: string;
      selected_profile?: string;
      selected_model?: string;
    };
    const provider = routing.selected_provider || pkg.provider;
    const profile = routing.selected_profile || pkg.profile;
    const model = routing.selected_model;
    return [provider, profile, model].filter(Boolean).join(' / ');
  } catch {
    return [pkg.provider, pkg.profile].filter(Boolean).join(' / ');
  }
};

// readExecutionSummary projects the unified execution summary from existing
// models only: skills pinned on the package, reasoning/model from the routing
// decision, isolation from the package workspace, and the routing mode with
// its typed reason. Nothing is invented when the data is absent.
const readExecutionSummary = (pkg: PackageRun | undefined) => {
  if (!pkg) return null;
  let routing: {
    selected_reasoning?: string;
    selected_model?: string;
    fallback?: boolean;
    reason?: string;
  } = {};
  try {
    routing = JSON.parse(pkg.routing_decision || '{}') as typeof routing;
  } catch {
    routing = {};
  }
  const skills = [...asArray<string>(pkg.skill_ids), ...asArray<string>(pkg.maestro_skills)];
  return {
    skillsText: skills.join(', '),
    reasoning: routing.selected_reasoning || routing.selected_model || '',
    isolation: pkg.workspace || '',
    fallback: routing.fallback === true,
    reason: routing.reason || '',
  };
};

export const FlowRunSurface: React.FC<{
  runId: string;
  project: Project;
  agents: Agent[];
  onOpenAgent?: (agent: Agent) => void;
  onOpenResources?: () => void;
}> = ({ runId, project, agents, onOpenAgent, onOpenResources }) => {
  const { t } = useTranslation();
  const [run, setRun] = useState<MissionRun | null>(null);
  const [evidence, setEvidence] = useState<FlowRunEvidence>({
    run_id: runId,
    capsules: [],
    receipts: [],
  });
  const [evidenceBundle, setEvidenceBundle] = useState<MissionEvidenceBundle | null>(null);
  const [plan, setPlan] = useState<WorkPlan | null>(null);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [resolvingOption, setResolvingOption] = useState('');
  const [decisionNotice, setDecisionNotice] = useState('');
  const interventionRef = useRef<HTMLDivElement>(null);
  // Synchronous action guards: setState is async, so two rapid clicks (or a
  // pause/cancel racing an intervention resolve) would both pass the `busy`
  // check before re-render. Refs flip synchronously.
  const busyRef = useRef(false);
  const resolvingRef = useRef(false);
  // Latest-wins guard: refresh() calls overlap (manual click + 1500ms poll).
  // A slow earlier request must never overwrite fresher state that arrived later.
  const refreshGuardRef = useRef(createRefreshGuard());

  const receiptSummary = useCallback(
    (receipt?: WorkReceipt) => {
      if (!receipt) return t('flowRun.noReceipt');
      const files = asStringArray(receipt.changed_files);
      const commands = asStringArray(receipt.commands);
      if (files.length) return t('flowRun.changedFiles', { count: files.length });
      if (commands.length) return t('flowRun.verifiedCommands', { count: commands.length });
      return receipt.summary || receipt.status;
    },
    [t],
  );

  const refresh = useCallback(async () => {
    const guard = refreshGuardRef.current;
    const seq = guard.next();
    try {
      const current = await nexusApi.getRun(runId);
      if (!guard.isLatest(seq)) return;
      setRun(current);
      const [nextEvidence, detail, nextBundle] = await Promise.all([
        nexusApi.getRunEvidence(runId),
        plan?.id === current.plan_id ? Promise.resolve(null) : nexusApi.getPlan(current.plan_id),
        nexusApi.getRunEvidenceBundle(runId).catch(() => null),
      ]);
      if (!guard.isLatest(seq)) return;
      // Normalize at the API boundary: Go encodes empty slices as null.
      setEvidence({
        run_id: nextEvidence.run_id || runId,
        capsules: asArray<ContextCapsule>(nextEvidence.capsules),
        receipts: asArray<WorkReceipt>(nextEvidence.receipts),
      });
      setEvidenceBundle(nextBundle);
      if (detail) setPlan(detail.plan);
      setError('');
    } catch (err) {
      if (!guard.isLatest(seq)) return;
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [runId, plan?.id]);

  useEffect(() => {
    void refresh();
  }, [refresh]);
  useEffect(() => {
    if (
      !run ||
      terminalMissionStates.has(run.state) ||
      run.state === 'PAUSED' ||
      run.state === 'BLOCKED_NEEDS_USER'
    )
      return;
    // Hidden documents (browser tab or backgrounded view) must not keep
    // polling; refresh immediately when visible again.
    const tick = () => {
      if (!document.hidden) void refresh();
    };
    const timer = window.setInterval(tick, 1500);
    document.addEventListener('visibilitychange', tick);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', tick);
    };
    // Depend on stable scalars: `run` identity changes on every refresh,
    // which would otherwise tear down and rebuild the interval each tick.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [run?.id, run?.state, refresh]);

  const flow = useMemo(() => (plan ? flowFromWorkPlan(plan) : null), [plan]);
  const userState = flowRunStateFromMission(run?.state || 'PENDING');
  const missionTeam = useMemo(() => readMissionTeamSelection(plan), [plan]);
  const packageRuns = useMemo(() => asArray<PackageRun>(run?.package_runs), [run]);
  const currentPackage = useMemo(
    () =>
      packageRuns.find(
        (pkg) => pkg.phase_id === run?.current_phase_id && pkg.state !== 'VERIFIED',
      ) || packageRuns[run?.current_pkg_index || 0],
    [packageRuns, run],
  );
  const verifiedCount = useMemo(
    () => packageRuns.filter((pkg) => pkg.state === 'VERIFIED').length,
    [packageRuns],
  );
  const capsules = useMemo(
    () => new Map(evidence.capsules.map((item) => [item.step.id, item])),
    [evidence.capsules],
  );
  const receipts = useMemo(
    () => new Map(evidence.receipts.map((item) => [item.step_id, item])),
    [evidence.receipts],
  );

  const perform = async (name: string, action: () => Promise<MissionRun>) => {
    if (busyRef.current || resolvingRef.current) return;
    busyRef.current = true;
    setBusy(name);
    setError('');
    try {
      const updated = await action();
      setRun(updated);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      busyRef.current = false;
      setBusy('');
    }
  };

  const resolveOption = async (optionId: string) => {
    const intervention = run?.needs_human;
    if (!run || !intervention || busyRef.current || resolvingRef.current) return;
    resolvingRef.current = true;
    setResolvingOption(optionId);
    setError('');
    try {
      // Same intervention model + resolve API as the Attention Center: no
      // second decision system, the decision is recorded on the run itself.
      const updated = await nexusApi.resolveIntervention(
        run.id,
        intervention.id,
        intervention.version,
        optionId,
      );
      setRun(updated);
      setDecisionNotice(t('flowRun.decisionResolved'));
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      resolvingRef.current = false;
      setResolvingOption('');
    }
  };

  const focusIntervention = () => {
    interventionRef.current?.scrollIntoView({ block: 'nearest' });
    interventionRef.current?.focus();
  };

  // A recorded decision belongs to one intervention: clear it when the run
  // or its pending intervention changes so stale success never lingers.
  const interventionId = run?.needs_human?.id || '';
  useEffect(() => {
    setDecisionNotice('');
  }, [runId, run?.id, interventionId]);

  const openAgent = async (agentId: string) => {
    if (!agentId || !onOpenAgent) return;
    const known = agents.find((agent) => agent.id === agentId);
    if (known) {
      onOpenAgent(known);
      return;
    }
    try {
      const detail = await nexusApi.getAgent(agentId);
      onOpenAgent(detail.agent);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  if (!run) {
    return (
      <div className="nx-surface-center">
        <Spinner label={t('flowRun.loading')} />
        {error && <InlineAlert tone="danger">{error}</InlineAlert>}
      </div>
    );
  }

  const canMutate = !terminalMissionStates.has(run.state);
  const isBlocked = run.state === 'BLOCKED_NEEDS_USER';
  const resumable = run.state === 'PAUSED';
  const intervention = run.needs_human;
  const interventionOptions = asArray<InterventionOption>(intervention?.options);
  const needsAction = isBlocked || userState === 'BLOCKED' || userState === 'FAILED';
  const providerIssue =
    providerIssuePattern.test(error || '') ||
    providerIssuePattern.test(run.paused_reason || '') ||
    packageRuns.some((pkg) => providerIssuePattern.test(pkg.error_message || ''));

  return (
    <div className="nx-surface-scroll nx-flow-run-surface" data-run-id={run.id}>
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">
            <Activity size={13} /> {t('flowRun.eyebrow')}
          </span>
          <h1>{flow?.title || plan?.title || `Run ${run.id}`}</h1>
          <p>{t('flowRun.subtitle', { project: project.name })}</p>
        </div>
        <div className="nx-flow-run-header-meta">
          <Badge>Rev {run.plan_revision}</Badge>
          <Badge tone="brand">{flow?.policy || 'GUIDED'}</Badge>
          <Badge tone={toneFor(userState)}>{t(`flow.state.${userState}`, userState)}</Badge>
        </div>
      </div>

      <Card className="nx-flow-run-status">
        <div>
          <strong>
            {t('flowRun.statusStep', { done: verifiedCount, total: packageRuns.length })}
          </strong>
          <span>{currentPackage?.title || run.current_phase_id || t('common.unknown')}</span>
        </div>
        <div>
          <strong>{t('flowRun.statusAgent')}</strong>
          <span>{currentPackage?.assigned_agent || t('flowRun.awaitingAllocation')}</span>
        </div>
        <div>
          <strong>{t('flowRun.statusState')}</strong>
          <span>{t(`flow.state.${userState}`, userState)}</span>
        </div>
        <div>
          <strong>{t('flowRun.statusAction')}</strong>
          <span>{needsAction ? t('flowRun.actionRequired') : t('flowRun.noActionRequired')}</span>
        </div>
      </Card>

      {evidenceBundle && (
        <Card
          className="nx-flow-run-status"
          data-evidence-complete={evidenceBundle.evidence_complete ? 'true' : 'false'}
        >
          <div>
            <strong>{t('flowRun.evidenceBundle', 'Evidence Bundle')}</strong>
            <span>
              {evidenceBundle.evidence_complete
                ? t('flowRun.evidenceComplete', 'Complete and verified')
                : t('flowRun.evidenceIncomplete', 'Evidence still incomplete')}
            </span>
          </div>
          <div>
            <strong>{t('flowRun.evidenceReceipts', 'Verified receipts')}</strong>
            <span>
              {evidenceBundle.receipts.filter((receipt) => receipt.status === 'VERIFIED').length}/
              {packageRuns.length}
            </span>
          </div>
          <div>
            <strong>{t('flowRun.evidenceChain', 'Validation chain')}</strong>
            <span>
              {evidenceBundle.validation.chain_verified
                ? t('flowRun.evidenceChainVerified', 'Verified')
                : t('flowRun.evidenceChainMissing', 'Not verified')}
            </span>
          </div>
          <div>
            <strong>{t('flowRun.evidenceMissing', 'Missing proof')}</strong>
            <span>
              {asArray<string>(evidenceBundle.missing_evidence).length
                ? asArray<string>(evidenceBundle.missing_evidence).join(' · ')
                : t('flowRun.evidenceNoneMissing', 'None')}
            </span>
          </div>
        </Card>
      )}

      {error && (
        <>
          <InlineAlert tone="danger" title={t('flowRun.actionFailed')}>
            {error}
          </InlineAlert>
          <div className="nx-flow-run-error-actions">
            <Button size="sm" onClick={() => void refresh()} disabled={Boolean(busy)}>
              <RefreshCw size={12} /> {t('flowRun.retry')}
            </Button>
            {isBlocked && interventionOptions.length > 0 && (
              <Button size="sm" tone="brand" onClick={focusIntervention}>
                {t('flowRun.reviewDecision')}
              </Button>
            )}
            {providerIssue && onOpenResources && (
              <Button size="sm" onClick={onOpenResources}>
                {t('flowRun.openResources')}
              </Button>
            )}
          </div>
          {providerIssue && <p className="nx-flow-run-hint">{t('flowRun.resourcesHint')}</p>}
          <details className="nx-flow-run-error-detail">
            <summary>{t('flowRun.errorDetails')}</summary>
            <code>{error}</code>
          </details>
        </>
      )}
      {isBlocked && (
        <InlineAlert tone="warning" title={t('flowRun.needsDecision')}>
          {run.paused_reason || t('flowRun.blockedFallback')}
        </InlineAlert>
      )}
      {decisionNotice && <InlineAlert tone="success">{decisionNotice}</InlineAlert>}

      {isBlocked && intervention && (
        <Card className="nx-flow-run-decision" data-intervention={intervention.id}>
          <div
            ref={interventionRef}
            tabIndex={-1}
            role="region"
            aria-label={t('flowRun.needsDecision')}
          >
            {intervention.question && (
              <div>
                <strong>{t('flowRun.decisionQuestion')}</strong>
                <p>{intervention.question}</p>
              </div>
            )}
            {intervention.context && (
              <div>
                <strong>{t('flowRun.decisionContext')}</strong>
                <p>{intervention.context}</p>
              </div>
            )}
            {intervention.impact && (
              <div>
                <strong>{t('flowRun.decisionImpact')}</strong>
                <p>{intervention.impact}</p>
              </div>
            )}
            {asArray<string>(intervention.recommended_actions).length > 0 && (
              <ul className="nx-flow-run-decision__recommended">
                {asArray<string>(intervention.recommended_actions).map((action) => (
                  <li key={action}>
                    <CircleAlert size={11} /> {action}
                  </li>
                ))}
              </ul>
            )}
            {interventionOptions.length > 0 && (
              <div>
                <strong>{t('flowRun.decisionOptions')}</strong>
                <div className="nx-flow-run-decision__options">
                  {interventionOptions.map((option) => (
                    <Button
                      key={option.id}
                      size="sm"
                      tone="brand"
                      disabled={
                        Boolean(busy) || (Boolean(resolvingOption) && resolvingOption !== option.id)
                      }
                      onClick={() => void resolveOption(option.id)}
                    >
                      <Play size={12} />{' '}
                      {resolvingOption === option.id ? t('flowRun.resolving') : option.label}
                    </Button>
                  ))}
                </div>
              </div>
            )}
          </div>
        </Card>
      )}

      {missionTeam && (
        <Card className="nx-flow-run-overview" data-mission-team="true">
          <div>
            <strong>{t('flowRuns.missionTeam')}</strong>
            <span>{missionTeam.preset || t('flowRuns.automaticResource')}</span>
          </div>
          <div>
            <strong>{t('flowRuns.preset')}</strong>
            <span>{missionTeam.cost_class}</span>
          </div>
          <div>
            <strong>{t('flowRuns.currentPhase')}</strong>
            <span>{currentPackage?.phase_type || run.current_phase_id || t('common.unknown')}</span>
          </div>
          <div>
            <strong>{t('flowRuns.activeAgent')}</strong>
            <span>{currentPackage?.assigned_agent || t('common.unknown')}</span>
          </div>
          <div>
            <strong>{t('flowRuns.actualResource')}</strong>
            <span>{readSelectedResource(currentPackage) || t('flowRuns.automaticResource')}</span>
          </div>
          {(() => {
            const summary = readExecutionSummary(currentPackage);
            if (!summary) return null;
            return (
              <details className="nx-flow-run-advanced">
                <summary>{t('flowRun.advancedDetails')}</summary>
                <div>
                  <strong>{t('flowRuns.executionSkills')}</strong>
                  <span>
                    {summary.skillsText ? summary.skillsText : t('flowRuns.executionNoSkills')}
                  </span>
                </div>
                {summary.reasoning ? (
                  <div>
                    <strong>{t('flowRuns.executionReasoning')}</strong>
                    <span>{summary.reasoning}</span>
                  </div>
                ) : null}
                {summary.isolation ? (
                  <div>
                    <strong>{t('flowRuns.executionIsolation')}</strong>
                    <span>{summary.isolation}</span>
                  </div>
                ) : null}
                <div>
                  <strong>{t('flowRuns.executionRoutingMode')}</strong>
                  <span>
                    {summary.fallback
                      ? t('flowRuns.executionFallback')
                      : t('flowRuns.executionDirect')}
                  </span>
                </div>
                {summary.reason ? (
                  <div>
                    <strong>{t('flowRuns.executionRoutingWhy')}</strong>
                    <span>{summary.reason}</span>
                  </div>
                ) : null}
              </details>
            );
          })()}
        </Card>
      )}

      <details className="nx-flow-run-advanced">
        <summary>{t('flowRun.advancedDetails')}</summary>
        <Card className="nx-flow-run-overview">
          <div>
            <strong>{t('flowRun.execSnapshot')}</strong>
            <code>{run.execution_snapshot_id || '—'}</code>
          </div>
          <div>
            <strong>{t('flowRun.progress')}</strong>
            <span>
              {t('flowRun.progressValue', { done: verifiedCount, total: packageRuns.length })}
            </span>
          </div>
          <div>
            <strong>{t('flowRun.iterationBudget')}</strong>
            <span>
              {run.total_iterations} / {run.contract?.max_total_iterations ?? '—'}
            </span>
          </div>
          <div>
            <strong>{t('flowRun.verification')}</strong>
            <span>
              {run.contract?.require_verification
                ? t('flowRun.verificationRequired')
                : t('flowRun.verificationDisabled')}
            </span>
          </div>
        </Card>
      </details>

      <div className="nx-flow-run-grid">
        {packageRuns.map((pkg, index) => {
          const state = packageRunState(pkg.state);
          const capsule = capsules.get(pkg.package_id);
          const receipt = receipts.get(pkg.package_id);
          const files = asStringArray(receipt?.changed_files);
          const commands = asStringArray(receipt?.commands);
          const remaining = asStringArray(receipt?.remaining_issues);
          const dependencyReceiptCount = asArray(capsule?.dependency_receipts).length;
          return (
            <Card className="nx-flow-run-step" data-state={state} key={pkg.id}>
              <div className="nx-flow-run-step__header">
                <div>
                  <small>{t('flowRun.stepN', { n: index + 1 })}</small>
                  <strong>{pkg.title}</strong>
                </div>
                <Badge tone={toneFor(state)}>{t(`flow.state.${state}`, state)}</Badge>
              </div>
              <p>{pkg.goal || t('flowRun.noGoal')}</p>
              <div className="nx-flow-run-step__meta">
                <span>
                  <Bot size={11} /> {pkg.assigned_agent || t('flowRun.awaitingAllocation')}
                </span>
                <span>
                  <Clock3 size={11} /> {t('flowRun.attempt', { n: pkg.attempt })}
                </span>
              </div>
              {(pkg.dependencies || []).length ? (
                <small className="nx-flow-run-deps">
                  {t('flowRun.afterDeps', { deps: (pkg.dependencies || []).join(', ') })}
                </small>
              ) : (
                <small className="nx-flow-run-deps">{t('flowRun.entryStep')}</small>
              )}

              <div className="nx-flow-run-evidence-row">
                <span data-ready={capsule ? 'true' : 'false'}>
                  <FileCode2 size={11} />{' '}
                  {capsule
                    ? t('flowRun.capsuleReady', { count: dependencyReceiptCount })
                    : t('flowRun.capsulePending')}
                </span>
                <span data-ready={receipt ? 'true' : 'false'}>
                  <ShieldCheck size={11} /> {receiptSummary(receipt)}
                </span>
              </div>

              {receipt && (
                <details className="nx-flow-run-receipt">
                  <summary>
                    {receipt.status === 'VERIFIED' ? (
                      <CheckCircle2 size={12} />
                    ) : (
                      <CircleAlert size={12} />
                    )}{' '}
                    {t('flowRun.workReceipt')}
                  </summary>
                  <div>
                    <strong>{t('flowRun.receiptSummary')}</strong>
                    <span>{receipt.summary}</span>
                  </div>
                  <div>
                    <strong>{t('flowRun.receiptFiles')}</strong>
                    <span>{files.length ? files.join(', ') : t('flowRun.noFileChanges')}</span>
                  </div>
                  <div>
                    <strong>{t('flowRun.receiptCommands')}</strong>
                    <span>{commands.length ? commands.join(' · ') : t('flowRun.noCommands')}</span>
                  </div>
                  {remaining.length > 0 && (
                    <div>
                      <strong>{t('flowRun.receiptRemaining')}</strong>
                      <span>{remaining.join(' · ')}</span>
                    </div>
                  )}
                </details>
              )}

              {pkg.error_message && <InlineAlert tone="danger">{pkg.error_message}</InlineAlert>}
              {pkg.assigned_agent && onOpenAgent && (
                <Button size="sm" onClick={() => void openAgent(pkg.assigned_agent)}>
                  <TerminalSquare size={12} /> {t('flowRun.openAgent')}
                </Button>
              )}
            </Card>
          );
        })}
      </div>

      <div className="nx-flow-run-actions">
        <Button size="sm" onClick={() => void refresh()} disabled={Boolean(busy)}>
          <RefreshCw size={12} /> {t('flowRun.refresh')}
        </Button>
        {canMutate && !resumable && !isBlocked && (
          <Button
            size="sm"
            onClick={() =>
              void perform('pause', () =>
                nexusApi.pauseRun(run.id, 'paused from Flow Run workspace'),
              )
            }
            disabled={Boolean(busy)}
          >
            <Pause size={12} /> {t('flowRun.pause')}
          </Button>
        )}
        {canMutate && !resumable && !isBlocked && (
          <Button
            size="sm"
            onClick={() =>
              void perform('control', () =>
                nexusApi.takeControlRun(run.id, 'take control from Flow Run workspace'),
              )
            }
            disabled={Boolean(busy)}
          >
            <TerminalSquare size={12} /> {t('flowRun.takeControl')}
          </Button>
        )}
        {resumable && (
          <Button
            size="sm"
            tone="brand"
            onClick={() => void perform('resume', () => nexusApi.resumeRun(run.id))}
            disabled={Boolean(busy)}
          >
            <Play size={12} /> {t('flowRun.resume')}
          </Button>
        )}
        {isBlocked && interventionOptions.length > 0 && (
          <Button size="sm" tone="brand" onClick={focusIntervention}>
            <Play size={12} /> {t('flowRun.reviewDecision')}
          </Button>
        )}
        {canMutate && (
          <Button
            size="sm"
            tone="danger"
            onClick={() =>
              void perform('cancel', () =>
                nexusApi.cancelRun(run.id, 'canceled from Flow Run workspace'),
              )
            }
            disabled={Boolean(busy)}
          >
            <Square size={12} /> {t('flowRun.cancel')}
          </Button>
        )}
      </div>
    </div>
  );
};
