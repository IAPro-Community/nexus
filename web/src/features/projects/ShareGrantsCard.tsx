import React, { useCallback, useEffect, useState } from 'react';
import { Share2, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ObservationShareGrant } from '../../types';
import {
  formatShareExpiry,
  parseShareTTLSeconds,
  shareGrantTone,
  shareRemainingBucket,
  sortShareGrants,
  type ShareGrantTone,
} from './shareGrantsModel';
import styles from './ShareGrantsCard.module.scss';

const toneBadgeTone: Record<ShareGrantTone, 'success' | 'warning' | 'default'> = {
  active: 'success',
  expired: 'warning',
  revoked: 'default',
};

/**
 * Viewer share grants for the selected project (workspace-sharing). Creating
 * a grant is an explicit human action; the card only shows what the server
 * returned (role, scope, expiry) and mirrors the current state — revoked and
 * expired grants never look active. Optional TTL bounds the link lifetime.
 */
export const ShareGrantsCard: React.FC<{ projectId: string }> = ({ projectId }) => {
  const { t, i18n } = useTranslation();
  const locale = i18n.language || 'en';
  const [grants, setGrants] = useState<ObservationShareGrant[]>([]);
  const [ttlInput, setTtlInput] = useState('');
  const [ttlError, setTtlError] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const payload = await nexus.listProjectShares(projectId);
      setGrants(Array.isArray(payload.grants) ? payload.grants : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    void load();
  }, [load]);

  const onCreate = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const ttl = parseShareTTLSeconds(ttlInput);
    if (ttlInput.trim() && ttl === null) {
      setTtlError(true);
      return;
    }
    setTtlError(false);
    nexus
      .createProjectShare(projectId, ttl !== null ? { ttl_seconds: ttl } : {})
      .then(() => load())
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  };

  const onRevoke = (grantId: string) => {
    nexus
      .revokeProjectShare(projectId, grantId)
      .then(() => load())
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  };

  const ordered = sortShareGrants(grants);

  return (
    <Card className={styles.card} data-testid="share-grants-card">
      <div className={styles.header}>
        <span className={styles.titleGroup}>
          <Share2 size={16} aria-hidden="true" />
          <span className={styles.title}>{t('projects.shareGrants.title')}</span>
        </span>
        <Button tone="ghost" onClick={() => void load()} disabled={loading}>
          {t('projects.shareGrants.refresh')}
        </Button>
      </div>
      <p className={styles.note}>{t('projects.shareGrants.note')}</p>

      <form className={styles.createRow} onSubmit={onCreate}>
        <label className={styles.ttlField} htmlFor="share-ttl">
          <span>{t('projects.shareGrants.ttlLabel')}</span>
          <Input
            id="share-ttl"
            type="number"
            min={1}
            max={604800}
            step={1}
            placeholder="3600"
            value={ttlInput}
            aria-invalid={ttlError}
            onChange={(value) => {
              setTtlInput(value);
              setTtlError(false);
            }}
          />
        </label>
        <Button type="submit" tone="brand">
          {t('projects.shareGrants.create')}
        </Button>
      </form>
      {ttlError && (
        <p className={styles.fieldError} role="alert">
          {t('projects.shareGrants.ttlInvalid')}
        </p>
      )}

      {error && (
        <p className={styles.error} role="alert">
          {t('projects.shareGrants.error', { message: error })}
        </p>
      )}

      {ordered.length === 0 && !loading && (
        <p className={styles.empty}>{t('projects.shareGrants.empty')}</p>
      )}

      <ul className={styles.list}>
        {ordered.map((grant) => {
          const tone = shareGrantTone(grant);
          const remaining = shareRemainingBucket(grant);
          return (
            <li key={grant.id} className={styles.row} data-tone={tone}>
              <code className={styles.grantId}>{grant.id.slice(0, 8)}</code>
              <Badge tone={toneBadgeTone[tone]}>{t(`projects.shareGrants.tone.${tone}`)}</Badge>
              <span className={styles.meta}>
                {t('projects.shareGrants.role', { role: grant.role })}
                {grant.agent_id ? ` · ${t('projects.shareGrants.agentScoped')}` : ''}
                {grant.expires_at && tone === 'active'
                  ? ` · ${t(`projects.shareGrants.remaining.${remaining}`, {
                      time: formatShareExpiry(grant.expires_at, locale),
                    })}`
                  : ''}
                {grant.expires_at && tone === 'expired'
                  ? ` · ${t('projects.shareGrants.expiredAt', {
                      time: formatShareExpiry(grant.expires_at, locale),
                    })}`
                  : ''}
              </span>
              {tone === 'active' && (
                <button
                  type="button"
                  className={styles.revoke}
                  onClick={() => onRevoke(grant.id)}
                  aria-label={t('projects.shareGrants.revokeAria')}
                >
                  <Trash2 size={14} aria-hidden="true" />
                  <span>{t('projects.shareGrants.revoke')}</span>
                </button>
              )}
            </li>
          );
        })}
      </ul>
    </Card>
  );
};
