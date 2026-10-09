import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Copy, Globe, RefreshCw, Server, Smartphone, Wifi, WifiOff } from 'lucide-react';
import QRCode from 'react-qr-code';
import { Badge, Button, Card, InlineAlert } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ProjectExecutionTargetCatalog, ProjectExecutionTargetProbe, TunnelStatus } from '../../types';
import { useTranslation } from 'react-i18next';
import { asArray } from '../../lib/safeArray';
import styles from './SettingsSurface.module.scss';

export const RemoteAccessTab: React.FC<{ projectId?: string }> = ({ projectId }) => {
  const { t } = useTranslation();
  const [status, setStatus] = useState<TunnelStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const startAttempt = useRef(0);
  const [error, setError] = useState('');
  const [showQR, setShowQR] = useState(false);
  const [qrData, setQrData] = useState<{
    url: string;
    bootstrap_url: string;
    expires_in_seconds: number;
  } | null>(null);

  const [targetCatalog, setTargetCatalog] = useState<ProjectExecutionTargetCatalog | null>(null);
  const [targetLoading, setTargetLoading] = useState(false);
  const [targetError, setTargetError] = useState('');
  const [targetProbes, setTargetProbes] = useState<Record<string, ProjectExecutionTargetProbe>>({});
  const [targetProbeLoading, setTargetProbeLoading] = useState('');
  const [targetProbeErrors, setTargetProbeErrors] = useState<Record<string, string>>({});
  const [targetMutationLoading, setTargetMutationLoading] = useState('');

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
      setTargetProbes({});
      setTargetProbeErrors({});
      setTargetProbeLoading('');
      setTargetMutationLoading('');
      return;
    }
    try {
      setTargetLoading(true);
      setTargetError('');
      setTargetCatalog(null);
      setTargetProbes({});
      setTargetProbeErrors({});
      setTargetProbeLoading('');
      setTargetMutationLoading('');
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

  const handleTargetProbe = useCallback(
    async (targetId: string) => {
      if (!projectId) return;
      try {
        setTargetProbeLoading(targetId);
        setTargetProbeErrors((current) => {
          const next = { ...current };
          delete next[targetId];
          return next;
        });
        const probe = await nexus.probeProjectExecutionTarget(projectId, targetId);
        setTargetProbes((current) => ({ ...current, [targetId]: probe }));
      } catch (e) {
        setTargetProbes((current) => {
          const next = { ...current };
          delete next[targetId];
          return next;
        });
        setTargetProbeErrors((current) => ({
          ...current,
          [targetId]:
            e instanceof Error
              ? e.message
              : t('settings.remote.targets.errors.probe', {
                  defaultValue: 'Could not probe this execution target.',
                }),
        }));
      } finally {
        setTargetProbeLoading((current) => (current === targetId ? '' : current));
      }
    },
    [projectId, t],
  );

  const handleTargetActivate = useCallback(
    async (targetId: string) => {
      if (!projectId) return;
      const probe = targetProbes[targetId];
      if (!probe?.ready || !probe.approval_token) {
        setTargetProbeErrors((current) => ({
          ...current,
          [targetId]: t('settings.remote.targets.errors.activationProbe', {
            defaultValue: 'Test this target again before activating it.',
          }),
        }));
        return;
      }
      try {
        setTargetMutationLoading(targetId);
        setTargetProbeErrors((current) => {
          const next = { ...current };
          delete next[targetId];
          return next;
        });
        await nexus.activateProjectExecutionTarget(projectId, targetId, probe.approval_token);
        setTargetProbes({});
        await refreshTargets();
      } catch (e) {
        setTargetProbeErrors((current) => ({
          ...current,
          [targetId]:
            e instanceof Error
              ? e.message
              : t('settings.remote.targets.errors.activate', {
                  defaultValue: 'Could not activate this execution target.',
                }),
        }));
      } finally {
        setTargetMutationLoading('');
      }
    },
    [projectId, refreshTargets, t, targetProbes],
  );

  const handleTargetDeactivate = useCallback(async () => {
    if (!projectId) return;
    try {
      setTargetMutationLoading('local');
      setTargetError('');
      await nexus.deactivateProjectExecutionTarget(projectId);
      setTargetProbes({});
      setTargetProbeErrors({});
      await refreshTargets();
    } catch (e) {
      setTargetError(
        e instanceof Error
          ? e.message
          : t('settings.remote.targets.errors.deactivate', {
              defaultValue: 'Could not return future starts to the local target.',
            }),
      );
    } finally {
      setTargetMutationLoading('');
    }
  }, [projectId, refreshTargets, t]);

  const handleStart = useCallback(async () => {
    const attempt = ++startAttempt.current;
    setConnecting(true);
    setLoading(true);
    setError('');
    try {
      const s = await nexus.startTunnel();
      // A pending request may resolve after the user has canceled it.
      if (startAttempt.current !== attempt) return;
      setStatus(s);
      setShowQR(false);
      setQrData(null);
      if (s.error) setError(s.error);
    } catch (e) {
      if (startAttempt.current === attempt) {
        setError(e instanceof Error ? e.message : t('settings.remote.errors.start'));
      }
    } finally {
      if (startAttempt.current === attempt) {
        setConnecting(false);
        setLoading(false);
      }
    }
  }, [t]);

  const handleStop = useCallback(async () => {
    // Invalidate an in-flight start response before sending Stop. Otherwise
    // a late startup completion could mark an already stopped tunnel active.
    startAttempt.current++;
    setConnecting(false);
    setCancelling(true);
    setLoading(true);
    setError('');
    try {
      await nexus.stopTunnel();
      setStatus((prev) =>
        prev
          ? { ...prev, active: false, starting: false, url: undefined, bootstrap_url: undefined }
          : null,
      );
      setShowQR(false);
      setQrData(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('settings.remote.errors.stop'));
    } finally {
      setCancelling(false);
      setLoading(false);
    }
  }, [t]);

  const generateInvite = useCallback(async () => {
    if (loading) return;
    setLoading(true);
    setError('');
    try {
      const data = await nexus.createTunnelInvite();
      setQrData(data);
      setShowQR(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('settings.remote.errors.qr'));
    } finally {
      setLoading(false);
    }
  }, [t, loading]);

  const handleShowQR = useCallback(() => {
    if (showQR) {
      setShowQR(false);
      setQrData(null);
      return;
    }
    void generateInvite();
  }, [showQR, generateInvite]);

  const active = status?.active ?? false;
  const starting = connecting || (status?.starting ?? false);
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
        <InlineAlert tone="info">
          {t('settings.remote.lanHttpsRequired', {
            defaultValue:
              'Na mesma rede, o acesso autenticado exige HTTPS confiável. Utilize um proxy TLS ou o túnel HTTPS; não é seguro transportar cookies de sessão por HTTP na rede.',
          })}
        </InlineAlert>

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
              {starting ? (
                <Button tone="danger" onClick={handleStop} disabled={cancelling}>
                  <WifiOff size={14} />
                  {cancelling
                    ? t('settings.remote.stopping')
                    : t('settings.remote.cancelStart', { defaultValue: 'Cancelar conexão' })}
                </Button>
              ) : (
                <Button tone="brand" onClick={handleStart} disabled={loading}>
                  <Wifi size={14} />
                  {t('settings.remote.activate')}
                </Button>
              )}

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
                  {target.kind === 'SSH' && targetCatalog?.remote_execution_enabled && (
                    <div className={styles.executionTargetProbeActions}>
                      <Button
                        tone="ghost"
                        onClick={() => handleTargetProbe(target.id)}
                        disabled={
                          targetProbeLoading === target.id ||
                          targetMutationLoading === target.id
                        }
                      >
                        <RefreshCw size={14} />
                        {targetProbeLoading === target.id
                          ? t('settings.remote.targets.probing', { defaultValue: 'Testing…' })
                          : t('settings.remote.targets.probe', { defaultValue: 'Test connection' })}
                      </Button>

                      {!target.effective &&
                        targetProbes[target.id]?.ready &&
                        targetProbes[target.id]?.approval_token && (
                          <Button
                            tone="brand"
                            onClick={() => void handleTargetActivate(target.id)}
                            disabled={targetMutationLoading === target.id}
                          >
                            {targetMutationLoading === target.id
                              ? t('settings.remote.targets.activating', {
                                  defaultValue: 'Activating…',
                                })
                              : t('settings.remote.targets.activateFuture', {
                                  defaultValue: 'Activate for future starts',
                                })}
                          </Button>
                        )}

                      {target.effective && (
                        <Button
                          tone="ghost"
                          onClick={() => void handleTargetDeactivate()}
                          disabled={targetMutationLoading === 'local'}
                        >
                          {targetMutationLoading === 'local'
                            ? t('settings.remote.targets.deactivating', {
                                defaultValue: 'Returning to local…',
                              })
                            : t('settings.remote.targets.useLocalFuture', {
                                defaultValue: 'Use local for future starts',
                              })}
                        </Button>
                      )}
                    </div>
                  )}
                  {targetProbeErrors[target.id] && (
                    <InlineAlert tone="danger">{targetProbeErrors[target.id]}</InlineAlert>
                  )}
                  {targetProbes[target.id] && (
                    <div className={styles.executionTargetProbeResult}>
                      <div className={styles.executionTargetProbeHeader}>
                        <Badge tone={targetProbes[target.id].ready ? 'success' : 'warning'}>
                          {targetProbes[target.id].ready
                            ? t('settings.remote.targets.probeReady', { defaultValue: 'bridge ready' })
                            : t('settings.remote.targets.probeReached', {
                                defaultValue: 'reachable, execution disabled',
                              })}
                        </Badge>
                        <span>
                          {targetProbes[target.id].machine_id} · {targetProbes[target.id].os}/
                          {targetProbes[target.id].arch} · Nexus {targetProbes[target.id].nexus_version}
                        </span>
                      </div>
                      {targetProbes[target.id].reason && (
                        <p className={`nx-muted-copy ${styles.executionTargetReason}`}>
                          {targetProbes[target.id].reason}
                        </p>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          <InlineAlert tone="info">
            {t('settings.remote.targets.futureStartsNotice', {
              defaultValue:
                'Changing the effective target affects future Agent starts only. Existing runtimes are not moved or stopped.',
            })}
          </InlineAlert>

          {targetCatalog &&
            targetCatalog.declared_default_target !== targetCatalog.effective_default_target && (
              <InlineAlert tone="info">
                {t('settings.remote.targets.declaredDiffers', {
                  defaultValue:
                    'The repository default and the explicitly activated target are different. Nexus follows the explicit project binding for future starts.',
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
              <InlineAlert tone="warning">
                {t('settings.remote.fullControlWarning', {
                  defaultValue:
                    'Este convite concede acesso completo ao workspace. Compartilhe somente com alguém de confiança.',
                })}
              </InlineAlert>
              <p className={styles.qrLabel}>{t('settings.remote.accessURL')}</p>
              <code className={styles.qrUrl}>{qrData.url}</code>
              <Button tone="ghost" onClick={() => void generateInvite()} disabled={loading}>
                <RefreshCw size={14} />
                {t('settings.remote.generateInvite', { defaultValue: 'Gerar outro convite' })}
              </Button>
              <Button
                tone="brand"
                onClick={() => {
                  void navigator.clipboard.writeText(qrData.bootstrap_url).catch(() =>
                    setError(
                      t('settings.remote.errors.copy', {
                        defaultValue: 'Não foi possível copiar o convite.',
                      }),
                    ),
                  );
                }}
              >
                <Copy size={14} />
                {t('settings.remote.copyInvite', {
                  defaultValue: 'Copiar convite de acesso',
                })}
              </Button>
              <p className={styles.mutedTextSmall}>
                {t('settings.remote.inviteExpiry', {
                  defaultValue: 'Convite individual, de uso único, válido por 15 minutos.',
                })}
              </p>

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
