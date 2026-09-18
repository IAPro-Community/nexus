import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
  FlowRunEvidence,
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

const receiptSummary = (receipt?: WorkReceipt) => {
  if (!receipt) return 'No receipt yet';
  const files = asStringArray(receipt.changed_files);
  const commands = asStringArray(receipt.commands);
  if (files.length) return `${files.length} changed file${files.length === 1 ? '' : 's'}`;
  if (commands.length)
    return `${commands.length} verified command${commands.length === 1 ? '' : 's'}`;
  return receipt.summary || receipt.status;
};

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

export const FlowRunSurface: React.FC<{
  runId: string;
  project: Project;
  agents: Agent[];
  onOpenAgent?: (agent: Agent) => void;
}> = ({ runId, project, agents, onOpenAgent }) => {
  const { t } = useTranslation();
  const [run, setRun] = useState<MissionRun | null>(null);
  const [evidence, setEvidence] = useState<FlowRunEvidence>({
    run_id: runId,
    capsules: [],
    receipts: [],
  });
  const [plan, setPlan] = useState<WorkPlan | null>(null);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    try {
      const current = await nexusApi.getRun(runId);
      setRun(current);
      const [nextEvidence, detail] = await Promise.all([
        nexusApi.getRunEvidence(runId),
        plan?.id === current.plan_id ? Promise.resolve(null) : nexusApi.getPlan(current.plan_id),
      ]);
      setEvidence(nextEvidence);
      if (detail) setPlan(detail.plan);
      setError('');
    } catch (err) {
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
    const timer = window.setInterval(() => void refresh(), 1500);
    return () => window.clearInterval(timer);
  }, [run, refresh]);

  const flow = useMemo(() => (plan ? flowFromWorkPlan(plan) : null), [plan]);
  const userState = flowRunStateFromMission(run?.state || 'PENDING');
  const missionTeam = useMemo(() => readMissionTeamSelection(plan), [plan]);
  const currentPackage = useMemo(
    () =>
      (run?.package_runs || []).find(
        (pkg) => pkg.phase_id === run?.current_phase_id && pkg.state !== 'VERIFIED',
      ) || (run?.package_runs || [])[run?.current_pkg_index || 0],
    [run],
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
    if (busy) return;
    setBusy(name);
    setError('');
    try {
      const updated = await action();
      setRun(updated);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy('');
    }
  };

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
        <Spinner label="Loading Flow Run…" />
        {error && <InlineAlert tone="danger">{error}</InlineAlert>}
      </div>
    );
  }

  const canMutate = !terminalMissionStates.has(run.state);
  const resumable = run.state === 'PAUSED' || run.state === 'BLOCKED_NEEDS_USER';

  return (
    <div className="nx-surface-scroll nx-flow-run-surface" data-run-id={run.id}>
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">
            <Activity size={13} /> FLOW RUN
          </span>
          <h1>{flow?.title || plan?.title || `Run ${run.id}`}</h1>
          <p>Durable Mission Runner execution · Project {project.name}</p>
        </div>
        <div className="nx-flow-run-header-meta">
          <Badge>Rev {run.plan_revision}</Badge>
          <Badge tone="brand">{flow?.policy || 'GUIDED'}</Badge>
          <Badge tone={toneFor(userState)}>{userState}</Badge>
        </div>
      </div>

      {error && (
        <InlineAlert tone="danger" title="Flow Run action failed">
          {error}
        </InlineAlert>
      )}
      {run.state === 'BLOCKED_NEEDS_USER' && (
        <InlineAlert tone="warning" title="Human decision required">
          {run.paused_reason ||
            'Execution stopped fail-closed and will not redispatch automatically.'}
        </InlineAlert>
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
        </Card>
      )}

      <Card className="nx-flow-run-overview">
        <div>
          <strong>Execution snapshot</strong>
          <code>{run.execution_snapshot_id || '—'}</code>
        </div>
        <div>
          <strong>Progress</strong>
          <span>
            {(run.package_runs || []).filter((pkg) => pkg.state === 'VERIFIED').length} /{' '}
            {(run.package_runs || []).length} verified
          </span>
        </div>
        <div>
          <strong>Iteration budget</strong>
          <span>
            {run.total_iterations} / {run.contract.max_total_iterations}
          </span>
        </div>
        <div>
          <strong>Verification</strong>
          <span>{run.contract.require_verification ? 'Required' : 'Contract disabled'}</span>
        </div>
      </Card>

      <div className="nx-flow-run-grid">
        {(run.package_runs || []).map((pkg, index) => {
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
                  <small>Step {index + 1}</small>
                  <strong>{pkg.title}</strong>
                </div>
                <Badge tone={toneFor(state)}>{state}</Badge>
              </div>
              <p>{pkg.goal || 'No additional goal text.'}</p>
              <div className="nx-flow-run-step__meta">
                <span>
                  <Bot size={11} /> {pkg.assigned_agent || 'Awaiting allocation'}
                </span>
                <span>
                  <Clock3 size={11} /> attempt {pkg.attempt}
                </span>
              </div>
              {(pkg.dependencies || []).length ? (
                <small className="nx-flow-run-deps">
                  after {(pkg.dependencies || []).join(', ')}
                </small>
              ) : (
                <small className="nx-flow-run-deps">entry Step</small>
              )}

              <div className="nx-flow-run-evidence-row">
                <span data-ready={capsule ? 'true' : 'false'}>
                  <FileCode2 size={11} />{' '}
                  {capsule
                    ? `Capsule · ${dependencyReceiptCount} receipt input${dependencyReceiptCount === 1 ? '' : 's'}`
                    : 'Capsule pending'}
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
                    Work Receipt
                  </summary>
                  <div>
                    <strong>Summary</strong>
                    <span>{receipt.summary}</span>
                  </div>
                  <div>
                    <strong>Files</strong>
                    <span>
                      {files.length ? files.join(', ') : 'No factual file changes captured'}
                    </span>
                  </div>
                  <div>
                    <strong>Commands</strong>
                    <span>
                      {commands.length ? commands.join(' · ') : 'No verification commands captured'}
                    </span>
                  </div>
                  {remaining.length > 0 && (
                    <div>
                      <strong>Remaining</strong>
                      <span>{remaining.join(' · ')}</span>
                    </div>
                  )}
                </details>
              )}

              {pkg.error_message && <InlineAlert tone="danger">{pkg.error_message}</InlineAlert>}
              {pkg.assigned_agent && onOpenAgent && (
                <Button size="sm" onClick={() => void openAgent(pkg.assigned_agent)}>
                  <TerminalSquare size={12} /> Open Agent
                </Button>
              )}
            </Card>
          );
        })}
      </div>

      <div className="nx-flow-run-actions">
        <Button size="sm" onClick={() => void refresh()} disabled={Boolean(busy)}>
          <RefreshCw size={12} /> Refresh
        </Button>
        {canMutate && !resumable && (
          <Button
            size="sm"
            onClick={() =>
              void perform('pause', () =>
                nexusApi.pauseRun(run.id, 'paused from Flow Run workspace'),
              )
            }
            disabled={Boolean(busy)}
          >
            <Pause size={12} /> Pause
          </Button>
        )}
        {canMutate && !resumable && (
          <Button
            size="sm"
            onClick={() =>
              void perform('control', () =>
                nexusApi.takeControlRun(run.id, 'take control from Flow Run workspace'),
              )
            }
            disabled={Boolean(busy)}
          >
            <TerminalSquare size={12} /> Take Control
          </Button>
        )}
        {resumable && (
          <Button
            size="sm"
            tone="brand"
            onClick={() =>
              void perform('resume', () =>
                run.state === 'PAUSED'
                  ? nexusApi.resumeRun(run.id)
                  : nexusApi.returnToMission(run.id),
              )
            }
            disabled={Boolean(busy)}
          >
            <Play size={12} /> Resume / Return
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
            <Square size={12} /> Cancel
          </Button>
        )}
      </div>
    </div>
  );
};
