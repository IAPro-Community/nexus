import React from 'react';
import { Activity, ShieldAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Progress } from '../../design-system';
import { translateStatus } from '../../i18n';
import type { ProviderAccount } from '../../types';
import { asArray } from '../../lib/safeArray';
import {
  accountQuotaState,
  capabilityLabelKey,
  displayLabelKey,
  formatResetDescription,
  groupHasCapacity,
  relativeAge,
  windowKindLabelKey,
} from './usageModel';
import styles from './UsageAccountCard.module.scss';

const healthTone = (health: string) =>
  health === 'healthy'
    ? 'success'
    : health === 'degraded'
      ? 'warning'
      : health === 'unhealthy'
        ? 'danger'
        : 'default';

const quotaTone = (state: string) =>
  state === 'confirmed' ? 'success' : state === 'stale' ? 'warning' : 'danger';

export const UsageAccountCard: React.FC<{ account: ProviderAccount; now?: number }> = ({
  account,
  now,
}) => {
  const { t, i18n } = useTranslation();
  const qv = account.quota_view;
  const groups = asArray<
    NonNullable<NonNullable<ProviderAccount['quota_view']>['model_groups']>[number]
  >(qv?.model_groups);
  const multi = groups.length > 1;
  const quotaState = accountQuotaState(account);
  const capabilities = Object.entries(account.capabilities || {})
    .filter(([, value]) => value && value !== 'false')
    .slice(0, 4);
  const planKey = displayLabelKey(qv?.plan);
  const age = relativeAge(qv?.fetched_at, now);
  const statusLabel = translateStatus(qv?.status || 'UNKNOWN');
  const freshness = age
    ? t(`usage.freshness.${age.unit}`, { status: statusLabel, count: age.count })
    : statusLabel;

  return (
    <article
      className={styles.card}
      data-quota-state={quotaState}
      data-available={account.available}
    >
      <header className={styles.header}>
        <span className={styles.icon} aria-hidden="true">
          {account.rate_limited ? <ShieldAlert size={16} /> : <Activity size={16} />}
        </span>
        <div className={styles.titleBlock}>
          <div className={styles.titleRow}>
            <strong>{account.display_name || account.provider}</strong>
            <Badge>{account.profile}</Badge>
            <Badge tone={healthTone(account.health)}>{translateStatus(account.health)}</Badge>
            <Badge tone={quotaTone(quotaState)}>
              {quotaState === 'confirmed'
                ? t('resources.confirmed')
                : quotaState === 'exhausted'
                  ? t('resources.exhausted')
                  : quotaState === 'blocked'
                    ? t('resources.rateLimited')
                    : quotaState === 'unauthenticated'
                      ? t('resources.reauthRequired')
                      : quotaState === 'stale'
                        ? t('resources.stale')
                        : t('resources.unknown')}
            </Badge>
            {account.is_default ? <Badge>{t('common.default')}</Badge> : null}
          </div>
          <div className={styles.metaRow}>
            {qv?.plan ? <small>{planKey ? t(planKey) : qv.plan}</small> : null}
            {qv?.account ? <small>{qv.account}</small> : null}
            <small>{freshness}</small>
            {account.cooldown_until ? (
              <small>
                {t('usage.cooldown')}: {new Date(account.cooldown_until).toLocaleString()}
              </small>
            ) : null}
          </div>
        </div>
      </header>

      {capabilities.length > 0 ? (
        <div className={styles.caps}>
          {capabilities.map(([key]) => (
            <Badge key={key}>{t(capabilityLabelKey(key), { defaultValue: key })}</Badge>
          ))}
        </div>
      ) : null}

      <div className={styles.quota}>
        {groups.length > 0 ? (
          groups.map((group, index) => {
            const groupLabelKey = displayLabelKey(group.name, group.key);
            return (
              <section key={group.key || group.name || index} className={styles.group}>
                {multi && (groupLabelKey || group.name) ? (
                  <div className={styles.groupHeading}>
                    <span>{groupLabelKey ? t(groupLabelKey) : group.name}</span>
                    <Badge tone={groupHasCapacity(group) ? 'success' : 'danger'}>
                      {groupHasCapacity(group)
                        ? t('usage.groupAvailable')
                        : t('usage.groupUnavailable')}
                    </Badge>
                  </div>
                ) : null}
                {asArray<NonNullable<typeof group.windows>[number]>(group.windows).map((window) => {
                  const windowKey =
                    windowKindLabelKey(window.kind) || displayLabelKey(window.label);
                  const reset = formatResetDescription(
                    window.reset_desc,
                    i18n.language,
                    (key, options) => t(key, options),
                  );
                  return (
                    <div key={window.kind || window.label} className={styles.window}>
                      <span className={styles.windowLabel}>
                        {windowKey ? t(windowKey) : window.label || window.kind}
                      </span>
                      <Progress
                        value={Number(window.remaining) || 0}
                        label={`${Math.round(Number(window.remaining) || 0)}%`}
                      />
                      {reset ? <span className={styles.reset}>{reset}</span> : null}
                    </div>
                  );
                })}
              </section>
            );
          })
        ) : (
          <p className={styles.unknown}>{t('resources.state.unknown')}</p>
        )}
      </div>
      <p className={styles.stateLine} data-state={quotaState}>
        {t(`resources.state.${quotaState}`)}
      </p>
    </article>
  );
};
