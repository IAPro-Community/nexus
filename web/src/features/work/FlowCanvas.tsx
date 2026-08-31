import React from 'react';
import { ArrowDown, GitBranch } from 'lucide-react';
import { Badge, Card } from '../../design-system';
import type { WorkPlan, WorkPackage } from '../../types';

export const FlowCanvas: React.FC<{ plan: WorkPlan | null; onSelectStep?: (step: WorkPackage) => void }> = ({ plan, onSelectStep }) => {
  const steps = (plan?.phases || []).flatMap((phase) => phase.packages.map((step) => ({ phase, step })));
  return (
    <Card aria-label="Flow Canvas" style={{ marginBottom: 16, padding: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
        <GitBranch size={15} /> <strong>Flow Canvas</strong>
        <Badge tone="default">deterministic DAG</Badge>
      </div>
      {!steps.length ? <span style={{ color: 'var(--color-text-muted)' }}>Create a Flow step to populate the canvas.</span> : (
        <div style={{ display: 'grid', gap: 8 }}>
          {steps.map(({ phase, step }, index) => (
            <React.Fragment key={step.id}>
              {index > 0 && <ArrowDown size={14} aria-hidden="true" style={{ margin: '0 auto', color: 'var(--color-text-muted)' }} />}
              <button type="button" onClick={() => onSelectStep?.(step)} style={{ textAlign: 'left', border: '1px solid var(--color-border)', background: 'var(--color-surface)', color: 'inherit', borderRadius: 6, padding: 10, cursor: 'pointer' }}>
                <strong>{step.title || step.id}</strong><div style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>{phase.title} · {step.dependencies?.length ? `after ${step.dependencies.join(', ')}` : 'root'}{step.parallel_group ? ` · parallel ${step.parallel_group}` : ''}</div>
              </button>
            </React.Fragment>
          ))}
        </div>
      )}
    </Card>
  );
};
