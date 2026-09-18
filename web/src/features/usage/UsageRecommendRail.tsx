import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, CheckCircle2, HelpCircle, Sparkles, Terminal } from 'lucide-react';
import { Badge, Card, EmptyState, Spinner } from '../../design-system';
import type { RecommendationResult } from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './UsageRecommendRail.module.scss';

export interface UsageRecommendRailProps {
  loading: boolean;
  result: RecommendationResult | null;
  error?: string;
  onRequestExplain: () => void;
  onOpenTerminal?: (provider: string, profile: string) => void;
}

export const UsageRecommendRail: React.FC<UsageRecommendRailProps> = ({
  loading,
  result,
  error,
  onRequestExplain,
  onOpenTerminal,
}) => {
  const { t } = useTranslation();
  const recommended = result?.recommended;
  const candidates = asArray<RecommendationResult['candidates'][number]>(result?.candidates).slice(
    0,
    4,
  );

  const handleLaunchTerminal = () => {
    if (!recommended) return;
    if (onOpenTerminal) {
      onOpenTerminal(recommended.account.provider, recommended.account.profile);
    } else {
      window.dispatchEvent(
        new CustomEvent('nexus:terminal-launch', {
          detail: {
            provider: recommended.account.provider,
            profile: recommended.account.profile,
            args: [],
            title: `${recommended.account.provider} · ${recommended.account.profile}`,
          },
        }),
      );
    }
  };

  return (
    <Card className={styles.card}>
      <header className={styles.header}>
        <div className={styles.headerTitle}>
          <Sparkles size={16} />
          <span>{t('usage.recommendTitle', 'Recomendação do scheduler')}</span>
        </div>
        <button
          type="button"
          className="nx-icon-button"
          onClick={onRequestExplain}
          title={t('usage.explainSchedulerCta')}
          aria-label={t('usage.explainSchedulerCta')}
        >
          <HelpCircle size={15} />
        </button>
      </header>

      {loading ? <Spinner label={t('usage.recommendLoading', 'Avaliando contas…')} /> : null}

      {!loading && error ? <p className={styles.error}>{error}</p> : null}

      {!loading && !error && !recommended ? (
        <EmptyState
          title={t('usage.recommendEmpty', 'Sem recomendação')}
          hint={t('usage.recommendEmptyHint', 'Nenhuma conta elegível para a política atual.')}
        />
      ) : null}

      {!loading && recommended ? (
        <div className={styles.recommended}>
          <div className={styles.row}>
            <strong>
              {recommended.account.display_name ||
                `${recommended.account.provider} (${recommended.account.profile})`}
            </strong>
            <div className={styles.badges}>
              <Badge tone="brand">#{recommended.rank}</Badge>
              {recommended.total_score ? (
                <Badge>{recommended.total_score.toFixed(1)} pts</Badge>
              ) : null}
            </div>
          </div>

          <p className={styles.explanation}>{result?.explanation}</p>

          {asArray<string>(recommended.pros).length > 0 ? (
            <ul className={styles.prosList}>
              {asArray<string>(recommended.pros)
                .slice(0, 3)
                .map((item) => (
                  <li key={item} className={styles.proItem}>
                    <CheckCircle2 size={13} className={styles.proIcon} />
                    <span>{item}</span>
                  </li>
                ))}
            </ul>
          ) : null}

          {asArray<string>(recommended.cons).length > 0 ? (
            <ul className={styles.consList}>
              {asArray<string>(recommended.cons)
                .slice(0, 2)
                .map((item) => (
                  <li key={item} className={styles.conItem}>
                    <AlertTriangle size={13} className={styles.conIcon} />
                    <span>{item}</span>
                  </li>
                ))}
            </ul>
          ) : null}

          <div className={styles.actionsRow}>
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              data-size="sm"
              onClick={handleLaunchTerminal}
            >
              <Terminal size={13} />
              {t('usage.useInTerminal', 'Usar no terminal')}
            </button>
          </div>
        </div>
      ) : null}

      {!loading && candidates.length > 1 ? (
        <div className={styles.candidatesSection}>
          <span className={styles.candidatesLabel}>
            {t('usage.otherCandidates', 'Outros candidatos')}
          </span>
          <div className={styles.candidatesList}>
            {candidates
              .filter((item) => item.account.id !== recommended?.account.id)
              .map((item) => (
                <div key={item.account.id} className={styles.candidateRow}>
                  <span className={styles.candidateName}>
                    #{item.rank} {item.account.display_name || item.account.profile}
                  </span>
                  <Badge tone={item.eligible ? 'success' : 'danger'}>
                    {item.eligible
                      ? t('usage.eligible', 'Elegível')
                      : item.rejection_reason || t('usage.ineligible', 'Inelegível')}
                  </Badge>
                </div>
              ))}
          </div>
        </div>
      ) : null}
    </Card>
  );
};
