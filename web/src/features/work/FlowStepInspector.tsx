import React from 'react';
import { Diff, Rows3, Scissors, Trash2, WandSparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, Input, Select, Textarea } from '../../design-system';
import type { Agent } from '../../types';
import type { FlowDraftModel, FlowStepModel } from './flowModel';

const lines = (value: string) =>
  value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
const joined = (value: string[] | undefined) => (value || []).join('\n');

export const FlowStepInspector: React.FC<{
  flow: FlowDraftModel;
  step: FlowStepModel | null;
  agents: Agent[];
  onChange: (patch: Partial<FlowStepModel>) => void;
  onAction: (action: 'REFINE' | 'EXPAND' | 'COMPARE' | 'SPLIT' | 'REMOVE') => void;
}> = ({ flow, step, agents, onChange, onAction }) => {
  const { t } = useTranslation();

  if (!step)
    return (
      <Card className="nx-flow-inspector nx-flow-inspector--empty">
        <strong>{t('flowInspector.emptyTitle', 'Select a Flow Step')}</strong>
        <p>
          {t(
            'flowInspector.emptyHint',
            'Assignment, dependencies, context and verification are edited here.',
          )}
        </p>
      </Card>
    );

  const candidates = flow.steps.filter((candidate) => candidate.id !== step.id);
  const deps = step.dependencies || [];

  return (
    <Card className="nx-flow-inspector">
      <div className="nx-flow-inspector__header">
        <div>
          <span className="nx-eyebrow">{t('flowInspector.eyebrow', 'STEP INSPECTOR')}</span>
          <h3>{step.title}</h3>
        </div>
        <Badge tone="brand">{step.assignmentStrategy}</Badge>
      </div>
      <label>
        <span>{t('flowInspector.title', 'Title')}</span>
        <Input value={step.title} onChange={(title) => onChange({ title })} />
      </label>
      <label>
        <span>{t('flowInspector.goal', 'Goal')}</span>
        <Textarea rows={4} value={step.goal} onChange={(goal) => onChange({ goal })} />
      </label>
      <div className="nx-flow-inspector__grid">
        <Select
          label={t('flowInspector.assignment', 'Assignment')}
          value={step.assignmentStrategy}
          onChange={(value) =>
            onChange({
              assignmentStrategy: value as FlowStepModel['assignmentStrategy'],
              agentId: value === 'EXISTING' ? step.agentId : undefined,
            })
          }
          options={[
            { value: 'EXISTING', label: t('flowInspector.existingAgent', 'Existing Agent') },
            { value: 'CREATE', label: t('flowInspector.createSpecialist', 'Create specialist') },
            { value: 'AUTO', label: t('flowInspector.autoScheduler', 'Auto / Scheduler') },
          ]}
        />
        {step.assignmentStrategy === 'EXISTING' ? (
          <Select
            label={t('flowInspector.agent', 'Agent')}
            value={step.agentId || ''}
            onChange={(agentId) => onChange({ agentId })}
            placeholder={t('flowInspector.selectAgent', 'Select Agent')}
            options={agents.map((agent) => ({
              value: agent.id,
              label: `${agent.name} · ${agent.status}`,
            }))}
          />
        ) : (
          <label>
            <span>{t('flowInspector.role', 'Role')}</span>
            <Input value={step.role} onChange={(role) => onChange({ role })} />
          </label>
        )}
        <Select
          label={t('flowInspector.resourcePolicy', 'Resource policy')}
          value={step.resourcePolicy || 'BALANCED'}
          onChange={(resourcePolicy) => onChange({ resourcePolicy })}
          options={[
            { value: 'BALANCED', label: t('flowInspector.balanced', 'Balanced') },
            { value: 'PRESERVE_QUOTA', label: t('flowInspector.preserveQuota', 'Preserve quota') },
            {
              value: 'PREFER_PROVIDER',
              label: t('flowInspector.preferProvider', 'Prefer provider'),
            },
            { value: 'MANUAL', label: t('flowInspector.manualRestriction', 'Manual restriction') },
          ]}
        />
        <label>
          <span>{t('flowInspector.parallelGroup', 'Parallel group')}</span>
          <Input
            value={step.parallelGroup || ''}
            onChange={(parallelGroup) => onChange({ parallelGroup: parallelGroup || undefined })}
          />
        </label>
        <label>
          <span>{t('flowInspector.providerRestriction', 'Provider restriction')}</span>
          <Input
            value={step.provider || ''}
            onChange={(provider) => onChange({ provider: provider || undefined })}
            placeholder={t('flowInspector.optional', 'optional')}
          />
        </label>
        <label>
          <span>{t('flowInspector.profileRestriction', 'Profile restriction')}</span>
          <Input
            value={step.profile || ''}
            onChange={(profile) => onChange({ profile: profile || undefined })}
            placeholder={t('flowInspector.optional', 'optional')}
          />
        </label>
      </div>
      <div className="nx-flow-inspector__section">
        <strong>{t('flowInspector.dependencies', 'Dependencies')}</strong>
        <div className="nx-flow-dependencies">
          {candidates.map((candidate) => {
            const checked = deps.includes(candidate.id);
            return (
              <label key={candidate.id}>
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() =>
                    onChange({
                      dependencies: checked
                        ? deps.filter((id) => id !== candidate.id)
                        : [...deps, candidate.id],
                    })
                  }
                />
                <span>{candidate.title}</span>
              </label>
            );
          })}
        </div>
      </div>
      <label>
        <span>{t('flowInspector.acceptanceCriteria', 'Acceptance criteria · one per line')}</span>
        <Textarea
          rows={4}
          value={joined(step.acceptanceCriteria)}
          onChange={(value) => onChange({ acceptanceCriteria: lines(value) })}
        />
      </label>
      <label>
        <span>{t('flowInspector.relevantPaths', 'Relevant paths · one per line')}</span>
        <Textarea
          rows={3}
          value={joined(step.relevantPaths)}
          onChange={(value) => onChange({ relevantPaths: lines(value) })}
        />
      </label>
      <label>
        <span>{t('flowInspector.skills')}</span>
        <Textarea
          rows={3}
          value={joined(step.skillIds.length ? step.skillIds : step.maestroSkills)}
          onChange={(value) => {
            const skillIds = lines(value);
            onChange({ skillIds, maestroSkills: skillIds });
          }}
        />
      </label>
      <label>
        <span>
          {t('flowInspector.verificationRequirements', 'Verification requirements · one per line')}
        </span>
        <Textarea
          rows={3}
          value={joined(step.verificationRequirements)}
          onChange={(value) => onChange({ verificationRequirements: lines(value) })}
        />
      </label>
      <div className="nx-flow-inspector__actions">
        <Button size="sm" onClick={() => onAction('REFINE')}>
          <WandSparkles size={12} />
          {t('flowInspector.refineLocally', 'Refine locally')}
        </Button>
        <Button size="sm" onClick={() => onAction('EXPAND')}>
          <Rows3 size={12} />
          {t('flowInspector.expand', 'Expand')}
        </Button>
        <Button size="sm" onClick={() => onAction('COMPARE')}>
          <Diff size={12} />
          {t('flowInspector.compare', 'Compare')}
        </Button>
        <Button size="sm" onClick={() => onAction('SPLIT')}>
          <Scissors size={12} />
          {t('flowInspector.split', 'Split')}
        </Button>
        <Button size="sm" tone="danger" onClick={() => onAction('REMOVE')}>
          <Trash2 size={12} />
          {t('flowInspector.remove', 'Remove')}
        </Button>
      </div>
    </Card>
  );
};
