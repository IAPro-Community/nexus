import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { GitCompareArrows, Play, RefreshCw, ShieldCheck } from 'lucide-react';
import { Badge, Button, Card } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type {
  ComparisonCampaignView,
  PlanExecutionPreview,
  Project,
  StepExecutionPreview,
  WorkPlan,
} from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './ComparisonCampaignPanel.module.scss';

export const ComparisonCampaignPanel: React.FC<{
  project: Project;
  plans: WorkPlan[];
  onLaunched: () => void;
}> = ({ project, plans, onLaunched }) => {
  const { t } = useTranslation();
  const [campaigns, setCampaigns] = useState<ComparisonCampaignView[]>([]);
  const [selected, setSelected] = useState<ComparisonCampaignView | null>(null);
  const [sourcePlanId, setSourcePlanId] = useState('');
  const [nameA, setNameA] = useState('A');
  const [instructionsA, setInstructionsA] = useState('');
  const [nameB, setNameB] = useState('B');
  const [instructionsB, setInstructionsB] = useState('');
  const [approveA, setApproveA] = useState(false);
  const [approveB, setApproveB] = useState(false);
  const [costA, setCostA] = useState(false);
  const [costB, setCostB] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const sourcePlans = useMemo(
    () =>
      plans.filter(
        (plan) =>
          !plan.structured_facts?.['nexus.comparison_campaign.id'] &&
          (plan.phases || []).some((phase) => (phase.packages || []).length > 0),
      ),
    [plans],
  );

  const refresh = useCallback(async () => {
    try {
      const next = asArray<ComparisonCampaignView>(
        await nexusApi.getComparisonCampaigns(project.id),
      );
      setCampaigns(next);
      setSelected((current) =>
        current ? next.find((item) => item.campaign.id === current.campaign.id) || current : null,
      );
      setError('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    }
  }, [project.id]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    setApproveA(false);
    setApproveB(false);
    setCostA(false);
    setCostB(false);
  }, [
    selected?.campaign.id,
    selected?.preview_a?.approval_token,
    selected?.preview_a?.cost_confirmation_token,
    selected?.preview_b?.approval_token,
    selected?.preview_b?.cost_confirmation_token,
  ]);

  const prepare = async () => {
    const source = sourcePlans.find((plan) => plan.id === sourcePlanId);
    if (!source || !instructionsA.trim() || !instructionsB.trim() || busy) return;
    setBusy(true);
    setError('');
    try {
      const created = await nexusApi.createComparisonCampaign({
        project_id: project.id,
        plan_id: source.id,
        plan_revision: source.current_revision,
        variant_a: { name: nameA.trim(), instructions: instructionsA.trim() },
        variant_b: { name: nameB.trim(), instructions: instructionsB.trim() },
      });
      setSelected(created);
      setCampaigns((current) => [
        created,
        ...current.filter((entry) => entry.campaign.id !== created.campaign.id),
      ]);
      setApproveA(false);
      setApproveB(false);
      setCostA(false);
      setCostB(false);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
    }
  };

  const launch = async () => {
    if (!selected || !selected.preview_a || !selected.preview_b || !approveA || !approveB || busy)
      return;
    if (
      (selected.preview_a.cost_confirmation_required && !costA) ||
      (selected.preview_b.cost_confirmation_required && !costB)
    )
      return;
    setBusy(true);
    setError('');
    try {
      const updated = await nexusApi.launchComparisonCampaign(selected.campaign.id, {
        approval_token_a: selected.preview_a.approval_token,
        cost_confirmation_token_a: selected.preview_a.cost_confirmation_required
          ? selected.preview_a.cost_confirmation_token
          : undefined,
        approval_token_b: selected.preview_b.approval_token,
        cost_confirmation_token_b: selected.preview_b.cost_confirmation_required
          ? selected.preview_b.cost_confirmation_token
          : undefined,
      });
      setSelected(updated);
      setCampaigns((current) =>
        current.map((entry) => (entry.campaign.id === updated.campaign.id ? updated : entry)),
      );
      onLaunched();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
      await refresh();
    } finally {
      setBusy(false);
    }
  };

  const renderPreview = (
    label: string,
    preview: PlanExecutionPreview,
    approved: boolean,
    onApprove: (value: boolean) => void,
    costApproved: boolean,
    onCostApprove: (value: boolean) => void,
  ) => (
    <section
      className={styles.preview}
      aria-label={t('flow.comparisonCampaignPreviewLabel', { label })}
    >
      <header>
        <strong>{t('flow.comparisonCampaignPreviewTitle', { label })}</strong>
        <Badge tone={preview.ready ? 'success' : 'danger'}>
          {preview.ready ? t('flow.comparisonCampaignReady') : t('flow.comparisonCampaignBlocked')}
        </Badge>
      </header>
      {asArray<StepExecutionPreview>(preview.steps).map((step) => (
        <p className={styles.route} key={step.package_id}>
          <span>{step.title}</span>
          <span>
            {[step.agent.name, step.resource.provider, step.resource.profile, step.model.model]
              .filter(Boolean)
              .join(' · ') || asArray<string>(step.blockers).join(' · ')}
          </span>
        </p>
      ))}
      {asArray<string>(preview.blockers).map((blocker) => (
        <p className={styles.blocker} key={blocker}>
          {blocker}
        </p>
      ))}
      {asArray<string>(preview.warnings).map((warning) => (
        <p className={styles.warning} key={warning}>
          {warning}
        </p>
      ))}
      <p className={styles.costSummary}>
        {t('flow.comparisonCampaignCostSummary', {
          level: preview.cost_budget.level,
          amount: new Intl.NumberFormat(undefined, {
            style: 'currency',
            currency: 'USD',
            maximumFractionDigits: 4,
          }).format(
            (preview.cost_budget.reported_cost_microusd +
              preview.cost_budget.api_equivalent_microusd +
              preview.cost_budget.inferred_microusd) /
              1_000_000,
          ),
        })}
      </p>
      <label className={styles.approval}>
        <input
          type="checkbox"
          checked={approved}
          disabled={busy || !preview.ready}
          onChange={(event) => onApprove(event.target.checked)}
        />
        {t('flow.comparisonCampaignApproveRoute', { label })}
      </label>
      {preview.cost_confirmation_required && (
        <label className={styles.approval}>
          <input
            type="checkbox"
            checked={costApproved}
            disabled={busy || !preview.ready}
            onChange={(event) => onCostApprove(event.target.checked)}
          />
          {t('flow.comparisonCampaignApproveCost', { label })}
        </label>
      )}
    </section>
  );

  const selectedCampaign = selected?.campaign;
  const canLaunch = Boolean(
    selected &&
    selectedCampaign?.state === 'PREPARED' &&
    selected.preview_a?.ready &&
    selected.preview_b?.ready &&
    approveA &&
    approveB &&
    (!selected.preview_a.cost_confirmation_required || costA) &&
    (!selected.preview_b.cost_confirmation_required || costB),
  );

  return (
    <Card className={styles.panel} data-tour="flow-comparison-campaigns">
      <header className={styles.header}>
        <div className={styles.title}>
          <GitCompareArrows size={16} />
          <div>
            <h2>{t('flow.comparisonCampaignTitle')}</h2>
            <p>{t('flow.comparisonCampaignIntro')}</p>
          </div>
        </div>
        <Button
          size="sm"
          onClick={() => void refresh()}
          disabled={busy}
          aria-label={t('flow.comparisonCampaignRefresh')}
        >
          <RefreshCw size={14} />
        </Button>
      </header>

      <p className={styles.notice}>{t('flow.comparisonCampaignCostNotice')}</p>

      <div className={styles.form}>
        <label>
          <span>{t('flow.comparisonCampaignSourcePlan')}</span>
          <select
            className="nx-input"
            value={sourcePlanId}
            onChange={(event) => setSourcePlanId(event.target.value)}
            disabled={busy}
          >
            <option value="">{t('flow.comparisonCampaignChoosePlan')}</option>
            {sourcePlans.map((plan) => (
              <option key={plan.id} value={plan.id}>
                {plan.title} · r{plan.current_revision}
              </option>
            ))}
          </select>
        </label>
        <div className={styles.variants}>
          <fieldset>
            <legend>{t('flow.comparisonCampaignVariant', { label: 'A' })}</legend>
            <label>
              <span>{t('flow.comparisonCampaignName')}</span>
              <input
                className="nx-input"
                value={nameA}
                maxLength={80}
                onChange={(event) => setNameA(event.target.value)}
                disabled={busy}
              />
            </label>
            <label>
              <span>{t('flow.comparisonCampaignInstructions')}</span>
              <textarea
                className="nx-input"
                value={instructionsA}
                maxLength={4000}
                onChange={(event) => setInstructionsA(event.target.value)}
                disabled={busy}
              />
            </label>
          </fieldset>
          <fieldset>
            <legend>{t('flow.comparisonCampaignVariant', { label: 'B' })}</legend>
            <label>
              <span>{t('flow.comparisonCampaignName')}</span>
              <input
                className="nx-input"
                value={nameB}
                maxLength={80}
                onChange={(event) => setNameB(event.target.value)}
                disabled={busy}
              />
            </label>
            <label>
              <span>{t('flow.comparisonCampaignInstructions')}</span>
              <textarea
                className="nx-input"
                value={instructionsB}
                maxLength={4000}
                onChange={(event) => setInstructionsB(event.target.value)}
                disabled={busy}
              />
            </label>
          </fieldset>
        </div>
        <Button
          tone="brand"
          onClick={() => void prepare()}
          disabled={
            busy ||
            !sourcePlanId ||
            !nameA.trim() ||
            !nameB.trim() ||
            !instructionsA.trim() ||
            !instructionsB.trim()
          }
        >
          <ShieldCheck size={14} /> {t('flow.comparisonCampaignPrepare')}
        </Button>
      </div>

      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
      {campaigns.length > 0 && (
        <nav className={styles.saved} aria-label={t('flow.comparisonCampaignSaved')}>
          <strong>{t('flow.comparisonCampaignSaved')}</strong>
          {campaigns.map((item) => (
            <Button
              key={item.campaign.id}
              size="sm"
              tone={selectedCampaign?.id === item.campaign.id ? 'brand' : 'default'}
              onClick={() => setSelected(item)}
            >
              {item.campaign.variant_a_name} / {item.campaign.variant_b_name}
              <Badge
                tone={
                  item.campaign.state === 'RUNNING'
                    ? 'warning'
                    : item.campaign.state === 'COMPLETED'
                      ? 'success'
                      : 'default'
                }
              >
                {t(`flow.comparisonCampaignState.${item.campaign.state}`)}
              </Badge>
            </Button>
          ))}
        </nav>
      )}

      {selected && (
        <section className={styles.selected} aria-label={t('flow.comparisonCampaignSelected')}>
          <p className={styles.base}>
            {t('flow.comparisonCampaignBase', {
              revision: selected.campaign.base_revision.slice(0, 12),
              plan: selected.campaign.source_plan_revision,
            })}
          </p>
          {selected.preview_a && selected.preview_b ? (
            <div className={styles.previews}>
              {renderPreview(
                selected.campaign.variant_a_name,
                selected.preview_a,
                approveA,
                setApproveA,
                costA,
                setCostA,
              )}
              {renderPreview(
                selected.campaign.variant_b_name,
                selected.preview_b,
                approveB,
                setApproveB,
                costB,
                setCostB,
              )}
            </div>
          ) : (
            <div className={styles.runs}>
              {selected.run_a && (
                <p>
                  {selected.campaign.variant_a_name}: {selected.run_a.id} · {selected.run_a.state}
                </p>
              )}
              {selected.run_b && (
                <p>
                  {selected.campaign.variant_b_name}: {selected.run_b.id} · {selected.run_b.state}
                </p>
              )}
              {selected.campaign.last_error && (
                <p className={styles.error}>{selected.campaign.last_error}</p>
              )}
            </div>
          )}
          {selected.campaign.state === 'PREPARED' && (
            <Button tone="brand" onClick={() => void launch()} disabled={busy || !canLaunch}>
              <Play size={14} /> {t('flow.comparisonCampaignLaunchBoth')}
            </Button>
          )}
          <p className={styles.noMerge}>{t('flow.comparisonNoMerge')}</p>
        </section>
      )}
    </Card>
  );
};
