import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { FlowDraftModel, FlowStepModel } from './flowModel';
import styles from './FlowPipeline.module.scss';
import { asArray } from '../../lib/safeArray';

const toneForStatus = (status: string) => {
  const value = String(status || '').toUpperCase();
  if (value === 'COMPLETED' || value === 'READY') return 'success';
  if (value === 'FAILED' || value === 'BLOCKED') return 'danger';
  if (value === 'EXECUTING' || value === 'RUNNING' || value === 'VERIFYING') return 'brand';
  return 'default';
};

/** Pure helper — unit-tested without a DOM renderer. */
export function groupStepsByPhase(steps: FlowStepModel[]): Map<string, FlowStepModel[]> {
  const map = new Map<string, FlowStepModel[]>();
  for (const step of steps) {
    const list = map.get(step.phaseId) || [];
    list.push(step);
    map.set(step.phaseId, list);
  }
  for (const [key, list] of map) {
    map.set(
      key,
      [...list].sort((a, b) => a.order - b.order),
    );
  }
  return map;
}

export const FlowPipeline: React.FC<{
  flow: FlowDraftModel;
  selectedId?: string;
  onSelect: (stepId: string) => void;
}> = ({ flow, selectedId, onSelect }) => {
  const { t } = useTranslation();
  const phases = asArray<FlowDraftModel['phases'][number]>(flow.phases);
  const stepsByPhase = useMemo(
    () => groupStepsByPhase(asArray<FlowStepModel>(flow.steps)),
    [flow.steps],
  );

  if (phases.length === 0) {
    return (
      <div className={styles.empty}>
        <strong>{t('flow.emptyCanvasTitle', 'Empty pipeline')}</strong>
        <p>
          {t(
            'flow.emptyCanvasHint',
            'Generate a draft with AI or create a base structure manually.',
          )}
        </p>
      </div>
    );
  }

  return (
    <div className={styles.root} role="list" aria-label={t('flow.viewPipeline', 'Pipeline')}>
      {phases
        .slice()
        .sort((a, b) => a.order - b.order)
        .map((phase) => (
          <section key={phase.id} className={styles.column} role="listitem">
            <header className={styles.columnHeader}>
              <strong>{phase.title || phase.id}</strong>
              <span className={styles.count}>{stepsByPhase.get(phase.id)?.length || 0}</span>
            </header>
            <div className={styles.cards}>
              {(stepsByPhase.get(phase.id) || []).map((step) => (
                <button
                  key={step.id}
                  type="button"
                  className={styles.card}
                  data-selected={selectedId === step.id ? 'true' : undefined}
                  data-tone={toneForStatus(step.status)}
                  onClick={() => onSelect(step.id)}
                >
                  <span className={styles.cardTitle}>{step.title || step.id}</span>
                  <span className={styles.cardMeta}>
                    {step.role || 'auto'} · {step.status || 'READY'}
                  </span>
                  {step.goal ? <span className={styles.cardGoal}>{step.goal}</span> : null}
                </button>
              ))}
            </div>
          </section>
        ))}
    </div>
  );
};
