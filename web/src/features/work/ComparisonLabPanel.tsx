import React from 'react';
import { useTranslation } from 'react-i18next';
import { GitCompareArrows, Scale } from 'lucide-react';
import { Badge, Button, Card } from '../../design-system';
import type { ComparisonCampaign, ComparisonLab, MissionRun } from '../../types';
import styles from './ComparisonLabPanel.module.scss';

export const ComparisonLabPanel: React.FC<{
  runs: MissionRun[];
  campaigns: ComparisonCampaign[];
  comparisons: ComparisonLab[];
  selectedRunIds: string[];
  selectedComparison: ComparisonLab | null;
  reason: string;
  busy: boolean;
  error: string;
  onCompare: () => void;
  onSelectComparison: (comparison: ComparisonLab) => void;
  onReasonChange: (reason: string) => void;
  onDecide: (decision: 'SELECT_A' | 'SELECT_B' | 'REJECT_BOTH') => void;
}> = ({
  runs,
  campaigns,
  comparisons,
  selectedRunIds,
  selectedComparison,
  reason,
  busy,
  error,
  onCompare,
  onSelectComparison,
  onReasonChange,
  onDecide,
}) => {
  const { t } = useTranslation();
  const eligibleRuns = runs.filter(
    (run) =>
      run.state === 'COMPLETED_VERIFIED' && run.delivery_integration && run.integrated_workspace,
  );
  const selectedRuns = selectedRunIds.map((id) => eligibleRuns.find((run) => run.id === id));
  const first = selectedRuns[0];
  const second = selectedRuns[1];
  const sameCampaign = Boolean(
    first &&
    second &&
    campaigns.some(
      (campaign) =>
        (campaign.run_a_id === first.id && campaign.run_b_id === second.id) ||
        (campaign.run_a_id === second.id && campaign.run_b_id === first.id),
    ),
  );
  const compatibleSelection =
    selectedRuns.length === 2 &&
    Boolean(first && second) &&
    Boolean(
      first &&
      second &&
      ((first.plan_id === second.plan_id && first.plan_revision === second.plan_revision) ||
        sameCampaign),
    ) &&
    first?.delivery_integration?.base_revision === second?.delivery_integration?.base_revision &&
    first?.integrated_workspace !== second?.integrated_workspace;
  const formatMicroUSD = (value: number) =>
    new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 6,
    }).format(value / 1_000_000);

  return (
    <Card className={styles.panel} data-tour="flow-comparison-lab">
      <header className={styles.header}>
        <div className={styles.heading}>
          <GitCompareArrows size={16} />
          <div>
            <h2>{t('flow.comparisonTitle')}</h2>
            <p>{t('flow.comparisonIntro')}</p>
          </div>
        </div>
        <Button size="sm" tone="brand" onClick={onCompare} disabled={busy || !compatibleSelection}>
          <Scale size={14} /> {t('flow.comparisonCreate')}
        </Button>
      </header>

      <p className={styles.notice}>{t('flow.comparisonIsolationNotice')}</p>
      <p className={styles.selectionHint}>
        {eligibleRuns.length === 0
          ? t('flow.comparisonNoEligibleRuns')
          : selectedRunIds.length === 2 && !compatibleSelection
            ? t('flow.comparisonSelectionMismatch')
            : t('flow.comparisonSelectionCount', { count: selectedRunIds.length })}
      </p>

      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}

      {comparisons.length > 0 && (
        <nav className={styles.savedComparisons} aria-label={t('flow.comparisonSaved')}>
          <strong>{t('flow.comparisonSaved')}</strong>
          {comparisons.map((comparison) => (
            <Button
              key={comparison.id}
              size="sm"
              tone={selectedComparison?.id === comparison.id ? 'brand' : 'default'}
              onClick={() => onSelectComparison(comparison)}
            >
              {comparison.run_a_id.slice(-8)} / {comparison.run_b_id.slice(-8)}
              <Badge tone={comparison.decision === 'PENDING' ? 'warning' : 'success'}>
                {t(`flow.comparisonDecision.${comparison.decision}`)}
              </Badge>
            </Button>
          ))}
        </nav>
      )}

      {selectedComparison && (
        <section className={styles.comparison} aria-labelledby="comparison-evidence-title">
          <h3 id="comparison-evidence-title">{t('flow.comparisonEvidence')}</h3>
          <div className={styles.columns}>
            {(
              [
                ['A', selectedComparison.run_a],
                ['B', selectedComparison.run_b],
              ] as const
            ).map(([label, evidence]) => (
              <article className={styles.alternative} key={evidence.run_id}>
                <header>
                  <strong>
                    {evidence.variant_name || t('flow.comparisonAlternative', { label })}
                  </strong>
                  <Badge tone={evidence.state === 'COMPLETED_VERIFIED' ? 'success' : 'warning'}>
                    {evidence.state}
                  </Badge>
                </header>
                {evidence.variant_instructions && <p>{evidence.variant_instructions}</p>}
                <small>
                  {evidence.run_id} ·{' '}
                  {t('flow.comparisonBase', { revision: evidence.base_revision.slice(0, 10) })}
                </small>

                <h4>{t('flow.comparisonChecks')}</h4>
                {evidence.checks.length === 0 ? (
                  <p>{t('flow.comparisonChecksUnknown')}</p>
                ) : (
                  <ul>
                    {evidence.checks.map((check, index) => (
                      <li key={`${check.name}-${index}`} data-passed={check.passed}>
                        <span>{check.name}</span>
                        <Badge tone={check.passed ? 'success' : 'danger'}>
                          {check.passed ? t('flow.comparisonPassed') : t('flow.comparisonFailed')}
                        </Badge>
                      </li>
                    ))}
                  </ul>
                )}

                <h4>{t('flow.comparisonDiff')}</h4>
                <ul className={styles.files}>
                  {evidence.changed_files.map((file) => (
                    <li key={file}>
                      <code>{file}</code>
                    </li>
                  ))}
                </ul>
                {evidence.diff ? (
                  <pre>
                    {evidence.diff}
                    {evidence.diff_truncated ? `\n${t('flow.comparisonDiffTruncated')}` : ''}
                  </pre>
                ) : (
                  <p>{t('flow.comparisonDiffUnavailable')}</p>
                )}

                <h4>{t('flow.comparisonCost')}</h4>
                <p>{t(`flow.comparisonCostStatus.${evidence.cost_evidence_status}`)}</p>
                <dl>
                  <div>
                    <dt>{t('flow.comparisonReportedCost')}</dt>
                    <dd>{formatMicroUSD(evidence.cost.provider_reported_microusd)}</dd>
                  </div>
                  <div>
                    <dt>{t('flow.comparisonEstimatedCost')}</dt>
                    <dd>{formatMicroUSD(evidence.cost.api_equivalent_microusd)}</dd>
                  </div>
                  <div>
                    <dt>{t('flow.comparisonInferredCost')}</dt>
                    <dd>{formatMicroUSD(evidence.cost.inferred_microusd)}</dd>
                  </div>
                  <div>
                    <dt>{t('flow.comparisonUnknownCost')}</dt>
                    <dd>{evidence.cost.unknown_cost_observations}</dd>
                  </div>
                </dl>
              </article>
            ))}
          </div>

          {selectedComparison.decision === 'PENDING' ? (
            <div className={styles.decision}>
              <label htmlFor="comparison-decision-reason">{t('flow.comparisonReasonLabel')}</label>
              <textarea
                id="comparison-decision-reason"
                value={reason}
                maxLength={2000}
                onChange={(event) => onReasonChange(event.target.value)}
                disabled={busy}
              />
              <div className={styles.decisionActions}>
                <Button
                  size="sm"
                  tone="brand"
                  disabled={busy || !reason.trim()}
                  onClick={() => onDecide('SELECT_A')}
                >
                  {t('flow.comparisonSelectA')}
                </Button>
                <Button
                  size="sm"
                  tone="brand"
                  disabled={busy || !reason.trim()}
                  onClick={() => onDecide('SELECT_B')}
                >
                  {t('flow.comparisonSelectB')}
                </Button>
                <Button
                  size="sm"
                  disabled={busy || !reason.trim()}
                  onClick={() => onDecide('REJECT_BOTH')}
                >
                  {t('flow.comparisonRejectBoth')}
                </Button>
              </div>
              <p>{t('flow.comparisonNoMerge')}</p>
            </div>
          ) : (
            <p className={styles.decisionRecorded}>
              {t(`flow.comparisonDecision.${selectedComparison.decision}`)}:{' '}
              {selectedComparison.decision_reason}
            </p>
          )}
        </section>
      )}
    </Card>
  );
};
