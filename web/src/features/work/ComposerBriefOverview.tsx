import React from 'react';
import { useTranslation } from 'react-i18next';
import type { ComposerSessionView } from '../../types';
import { asStringArray } from '../../lib/safeArray';
import {
  composerConstraintLines,
  composerExpectedOutcome,
  composerScopeLines,
} from './composerModel';
import styles from './ComposerInspector.module.scss';

export const ComposerBriefOverview: React.FC<{ view: ComposerSessionView }> = ({ view }) => {
  const { t } = useTranslation();
  const scope = composerScopeLines(view);
  const constraints = composerConstraintLines(view);
  const criteria = asStringArray(
    view.brief.quality?.acceptance_criteria ?? view.brief.success_criteria,
  );
  const expected = composerExpectedOutcome(view);

  const rows: Array<{ id: string; label: string; value: string }> = [
    { id: 'goal', label: t('work.composer.overview.goal', 'Objective'), value: view.brief.goal },
    {
      id: 'expected',
      label: t('work.composer.overview.expected', 'Expected result'),
      value: expected,
    },
    {
      id: 'scope',
      label: t('work.composer.overview.scope', 'Scope'),
      value: scope.join(' · '),
    },
    {
      id: 'constraints',
      label: t('work.composer.overview.constraints', 'Constraints'),
      value: constraints.join(' · '),
    },
    {
      id: 'criteria',
      label: t('work.composer.overview.criteria', 'Ready criteria'),
      value: criteria.join(' · '),
    },
    ...(view.brief.recommendations?.length
      ? [
          {
            id: 'recommendations',
            label: t('work.composer.overview.recommendations', 'Sugestões da IA (a validar)'),
            value: view.brief.recommendations.map((item) => item.value).join(' · '),
          },
        ]
      : []),
  ];

  return (
    <section className={styles.group} aria-labelledby="composer-brief-overview">
      <dl className={styles.overviewList} id="composer-brief-overview">
        {rows.map((row) => (
          <div key={row.id} className={styles.overviewRow}>
            <dt>{row.label}</dt>
            <dd>{row.value || t('work.composer.overview.empty', 'Not captured yet.')}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
};
