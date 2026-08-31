import React, { useEffect, useState } from 'react';
import { Pause, Play, Square, Terminal } from 'lucide-react';
import { Badge, Button, Card, EmptyState, Spinner } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { MissionRun } from '../../types';

export const FlowRunSurface: React.FC<{ runId?: string; onOpenAgent?: (agentId: string) => void }> = ({ runId, onOpenAgent }) => {
  const [run, setRun] = useState<MissionRun | null>(null);
  const [error, setError] = useState('');
  useEffect(() => { if (!runId) return; let live = true; void nexusApi.getRun(runId).then((value) => live && setRun(value)).catch((e) => live && setError(e instanceof Error ? e.message : String(e))); return () => { live = false; }; }, [runId]);
  if (!runId) return <EmptyState title="Flow Run unavailable" />;
  if (error) return <EmptyState title="Unable to load Flow Run" hint={error} />;
  if (!run) return <div style={{ padding: 24 }}><Spinner /></div>;
  const current = run.package_runs[run.current_pkg_index];
  const terminal = ['COMPLETED_VERIFIED', 'FAILED', 'FAILED_NO_PROGRESS', 'FAILED_BUDGET_EXCEEDED', 'FAILED_VERIFICATION', 'CANCELED_BY_USER'].includes(run.state);
  return <div className="nx-surface-scroll" style={{ padding: 24 }}>
    <div className="nx-page-header"><div><span className="nx-eyebrow">Flow Run</span><h1>{run.plan_id}</h1><p>Revision {run.plan_revision} · Flow policy is governed by the approved run contract</p></div><Badge tone={terminal ? 'default' : 'brand'}>{run.state}</Badge></div>
    <Card><strong>Current Step</strong><p>{current?.title || 'No active step'}</p><small>{current?.dependencies?.length ? `Dependencies: ${current.dependencies.join(', ')}` : 'Root step'}</small><div style={{ display: 'flex', gap: 8, marginTop: 14 }}>
      {current?.assigned_agent && <Button onClick={() => onOpenAgent?.(current.assigned_agent)}><Terminal size={14} /> Open Agent</Button>}
      {run.state === 'PAUSED' ? <Button onClick={() => void nexusApi.resumeRun(run.id)}><Play size={14} /> Resume</Button> : <Button disabled={terminal} onClick={() => void nexusApi.pauseRun(run.id, 'paused from Flow Run')}><Pause size={14} /> Pause</Button>}
      <Button disabled={terminal} onClick={() => void nexusApi.cancelRun(run.id, 'canceled from Flow Run')}><Square size={14} /> Cancel</Button>
    </div></Card>
    <Card style={{ marginTop: 16 }}><strong>Steps and verification</strong>{run.package_runs.map((step) => <div key={step.id} style={{ borderTop: '1px solid var(--color-border)', padding: '10px 0' }}><strong>{step.title}</strong><span style={{ marginLeft: 8 }}><Badge tone="default">{step.state}</Badge></span><div style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>{step.verifications?.length || 0} verification result(s) · agent {step.assigned_agent || 'unassigned'}</div></div>)}</Card>
  </div>;
};
