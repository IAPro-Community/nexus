import React, { useCallback, useEffect, useState } from 'react';
import { Globe, RefreshCw, Server, Smartphone, Wifi, WifiOff } from 'lucide-react';
import QRCode from 'react-qr-code';
import { Badge, Button, Card, InlineAlert } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ProjectExecutionTargetCatalog, TunnelStatus } from '../../types';
import { useTranslation } from 'react-i18next';
import { asArray } from '../../lib/safeArray';
import styles from './SettingsSurface.module.scss';

export const RemoteAccessTab: React.FC<{ projectId?: string }> = ({ projectId }) => {
  const { t } = useTranslation();
  const [status, setStatus] = useState<TunnelStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showQR, setShowQR] = useState(false);
  const [qrData, setQrData] = useState<{ url: string; bootstrap_url: string } | null>(null);
  const [targetCatalog, setTargetCatalog] = useState<ProjectExecutionTargetCatalog | null>(null);
  const [targetLoading, setTargetLoading] = useState(false);
  const [targetError, setTargetError] = useState('');

  const refresh = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const s = await nexus.getTunnelStatus();
      setStatus(s);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('settings.remote.errors.status'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const refreshTargets = useCallback(async () => {
    if (!projectId) {
      setTargetCatalog(null);
      setTargetError('');
      return;
    }
    try {
      setTargetLoading(true);
      setTargetError('');
      setTargetCatalog(await nexus.getProjectExecutionTargets(projectId));
    } catch (e) {
      setTargetCatalog(null);
      setTargetError(
        e instanceof Error
          ? e.message
          : t('settings.remote.targets.errors.load', {
              defaultValue: 'Could not load project execution targets.',
            }),
      );
    } finally {
      setTargetLoading(false);
    }
  }, [projectId, t]);

  useEffect(() => {
    void refreshTargets();
  }, [refreshTargets]);

  const handleStart = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const s = await nexus.startTunnel();
      setStatus(s);
      if (s.error) setError(s.error);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('settings.remote.errors.start'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const handleStop = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      await nexus.stopTunnel();
      setStatus((prev) =>
        prev ? { ...prev, active: false, url: undefined, bootstrap_url: undefined } : null,
      );
      setShowQR(false);
      setQrData(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('settings.remote.errors.stop'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const handleShowQR = useCallback(async () => {
    try {
      const data = await nexus.getTunnelQR();
      setQrData(data);
      setShowQR(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('settings.remote.errors.qr'));
    }
  }, [t]);

  const active = status?.active ?? false;
  const installed = status?.cloudflared_installed ?? false;

  return (
    <>
      {/* Remote Access Card */}
      <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
        <div className="nx-settings-card__title">
          <Globe size={17} />
          <div>
            <strong>{t('settings.remote.title')}</strong>
            <small>{t('settings.remote.description')}</small>
          </div>
        </div>

        <p className={`nx-muted-copy ${styles.mutedTextSmall}`}>{t('settings.remote.intro')}</p>

        {!installed && <InlineAlert tone="info">{t('settings.remote.downloadNotice')}</InlineAlert>}

        {error && <InlineAlert tone="danger">{error}</InlineAlert>}

        <div className={styles.tunnelControls}>
          {active ? (
            <>
              <div className={styles.tunnelStatus}>
                <Wifi size={15} className={styles.tunnelStatusIcon} />
                <span className={styles.tunnelStatusActive}>{t('settings.remote.active')}</span>
              </div>

              <div className={styles.tunnelUrl}>
                <code>{status?.url}</code>
              </div>

              <div className={styles.tunnelActions}>
                <Button tone="brand" onClick={handleShowQR} disabled={loading}>
                  <Smartphone size={14} />
                  {showQR ? t('settings.remote.hideQR') : t('settings.remote.mobileQR')}
                </Button>

                <Button tone="danger" onClick={handleStop} disabled={loading}>
                  <WifiOff size={14} />
                  {loading ? t('settings.remote.stopping') : t('settings.remote.stop')}
                </Button>

                <Button tone="ghost" onClick={refresh} disabled={loading}>
                  <RefreshCw size={14} />
                </Button>
              </div>
            </>
          ) : (
            <div className={styles.tunnelActions}>
              <Button tone="brand" onClick={handleStart} disabled={loading}>
                <Wifi size={14} />
                {loading ? t('settings.remote.starting') : t('settings.remote.activate')}
              </Button>

              <Button tone="ghost" onClick={refresh} disabled={loading}>
                <RefreshCw size={14} />
              </Button>
            </div>
          )}
        </div>

        {status?.cloudflared_version && (
          <p className={`nx-muted-copy ${styles.mutedTextSmall}`}>
            {t('settings.remote.statusVersion', { version: status.cloudflared_version })}
          </p>
        )}
      </Card>

      {projectId && (
        <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
          <div className="nx-settings-card__title">
            <Server size={17} />
            <div>
              <strong>
                {t('settings.remote.targets.title', { defaultValue: 'Execution targets' })}
              </strong>
              <small>
                {t('settings.remote.targets.description', {
                  defaultValue: 'Read-only preview for the active Project.',
                })}
              </small>
            </div>
          </div>

          {targetError && <InlineAlert tone="danger">{targetError}</InlineAlert>}

          <div className={styles.executionTargetSummary}>
            <div>
              <span>{t('settings.remote.targets.declared', { defaultValue: 'Declared default' })}</span>
              <strong>{targetCatalog?.declared_default_target || 'local'}</strong>
            </div>
            <div>
              <span>{t('settings.remote.targets.effective', { defaultValue: 'Effective target' })}</span>
              <strong>{targetCatalog?.effective_default_target || 'local'}</strong>
            </div>
            <Badge tone={targetCatalog?.remote_execution_enabled ? 'info' : 'default'}>
              {targetCatalog?.remote_execution_enabled
                ? t('settings.remote.targets.flagEnabled', { defaultValue: 'remote_execution enabled' })
                : t('settings.remote.targets.flagDisabled', { defaultValue: 'remote_execution disabled' })}
            </Badge>
            <Button tone="ghost" onClick={refreshTargets} disabled={targetLoading}>
              <RefreshCw size={14} />
              {t('common.refresh')}
            </Button>
          </div>

          {targetLoading && !targetCatalog ? (
            <p className={`nx-muted-copy ${styles.mutedTextSmall}`}>
              {t('common.loading')}
            </p>
          ) : (
            <div className={styles.executionTargetList}>
              {(targetCatalog?.targets || []).map((target) => (
                <div key={target.id} className={styles.executionTargetRow}>
                  <div className={styles.executionTargetHeader}>
                    <div>
                      <strong>{target.id}</strong>
                      <span>{target.kind} · {target.location}</span>
                    </div>
                    <div className={styles.executionTargetBadges}>
                      {target.declared_default && (
                        <Badge tone="brand">
                          {t('settings.remote.targets.default', { defaultValue: 'declared default' })}
                        </Badge>
                      )}
                      {target.effective && (
                        <Badge tone="success">
                          {t('settings.remote.targets.active', { defaultValue: 'effective' })}
                        </Badge>
                      )}
                      <Badge
                        tone={
                          target.activation_status === 'READY'
                            ? 'success'
                            : target.activation_status === 'FEATURE_DISABLED'
                              ? 'default'
                              : 'info'
                        }
                      >
                        {target.activation_status}
                      </Badge>
                    </div>
                  </div>
                  <div className={styles.executionTargetMeta}>
                    <code>{target.workspace}</code>
                    <span>{target.transport}</span>
                    {target.machine_id && <span>{target.machine_id}</span>}
                  </div>
                  {target.reason && (
                    <p className={`nx-muted-copy ${styles.executionTargetReason}`}>
                      {target.reason}
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}

          {targetCatalog &&
            targetCatalog.declared_default_target !== targetCatalog.effective_default_target && (
              <InlineAlert tone="info">
                {t('settings.remote.targets.previewOnly', {
                  defaultValue:
                    'The repository declares a remote default, but Nexus keeps execution local until a managed remote runtime and explicit approval boundary are available.',
                })}
              </InlineAlert>
            )}
        </Card>
      )}

      {/* QR Code Card */}
      {showQR && qrData && (
        <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
          <div className="nx-settings-card__title">
            <Smartphone size={17} />
            <div>
              <strong>{t('settings.remote.connect')}</strong>
              <small>{t('settings.remote.scan')}</small>
            </div>
          </div>

          <div className={styles.qrContainer}>
            <div className={styles.qrCode}>
              <QRCode
                value={qrData.bootstrap_url}
                size={200}
                level="M"
                fgColor="var(--nx-text)"
                bgColor="var(--nx-surface)"
              />
            </div>

            <div className={styles.qrInfo}>
              <p className={styles.qrLabel}>{t('settings.remote.accessURL')}</p>
              <code className={styles.qrUrl}>{qrData.url}</code>

              <p className={`nx-muted-copy ${styles.mutedTextSmall}`}>
                {t('settings.remote.tokenNotice')}
              </p>
            </div>
          </div>
        </Card>
      )}

      {/* How it works Card */}
      <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
        <div className="nx-settings-card__title">
          <Wifi size={17} />
          <div>
            <strong>{t('settings.remote.how')}</strong>
            <small>{t('settings.remote.limitations')}</small>
          </div>
        </div>

        <ul className={styles.tunnelInfoList}>
          {asArray<string>(t('settings.remote.facts', { returnObjects: true })).map((fact) => (
            <li key={fact}>{fact}</li>
          ))}
        </ul>
      </Card>
    </>
  );
};
