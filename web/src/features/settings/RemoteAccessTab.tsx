import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Copy, Globe, RefreshCw, Smartphone, Wifi, WifiOff, Bell } from 'lucide-react';
import QRCode from 'react-qr-code';
import { Button, Card, InlineAlert } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { PushState, RemoteDevice } from '../../nexus/api';
import type { TunnelStatus } from '../../types';
import { useTranslation } from 'react-i18next';
import { asArray } from '../../lib/safeArray';
import { formatAccessCode, remoteAccessMode, tunnelOriginKind } from './remoteAccessModel';
import styles from './SettingsSurface.module.scss';

const formatDeviceSeen = (value: string): string => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'short', timeStyle: 'short' }).format(
    date,
  );
};

// PushManager.applicationServerKey wants raw bytes; the server hands out the
// uncompressed P-256 VAPID key as base64url.
const vapidKeyBytes = (base64url: string): Uint8Array<ArrayBuffer> => {
  const padded = base64url.replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(padded + '='.repeat((4 - (padded.length % 4)) % 4));
  const bytes = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i += 1) bytes[i] = raw.charCodeAt(i);
  return bytes;
};

export const RemoteAccessTab: React.FC = () => {
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
    access_code: string;
    role: 'controller' | 'viewer';
    expires_in_seconds: number;
  } | null>(null);
  const [inviteRole, setInviteRole] = useState<'controller' | 'viewer'>('viewer');
  const [consentGiven, setConsentGiven] = useState(false);
  const [devices, setDevices] = useState<RemoteDevice[]>([]);
  const [devicesManagedHere, setDevicesManagedHere] = useState(true);
  const [revokingDeviceId, setRevokingDeviceId] = useState('');

  const loadDevices = useCallback(async () => {
    try {
      const data = await nexus.listRemoteDevices();
      setDevices(asArray<RemoteDevice>(data?.devices));
      setDevicesManagedHere(true);
    } catch {
      // A paired device cannot manage devices; the server answers 403.
      setDevices([]);
      setDevicesManagedHere(false);
    }
  }, []);

  const handleRevokeDevice = useCallback(
    async (deviceId: string) => {
      if (!deviceId || revokingDeviceId) return;
      setRevokingDeviceId(deviceId);
      setError('');
      try {
        await nexus.revokeRemoteDevice(deviceId);
        setDevices((prev) => prev.filter((device) => device.device_id !== deviceId));
      } catch (e) {
        setError(
          e instanceof Error
            ? e.message
            : t('settings.remote.devices.revokeError', {
                defaultValue: 'Não foi possível revogar o dispositivo.',
              }),
        );
      } finally {
        setRevokingDeviceId('');
      }
    },
    [revokingDeviceId, t],
  );

  useEffect(() => {
    void loadDevices();
  }, [loadDevices]);

  const [pushInfo, setPushInfo] = useState<PushState | null>(null);
  const [pushSubscribed, setPushSubscribed] = useState(false);
  const [pushBusy, setPushBusy] = useState(false);
  const [pushMessage, setPushMessage] = useState('');
  const pushSupported =
    typeof window !== 'undefined' &&
    'serviceWorker' in navigator &&
    typeof window.PushManager !== 'undefined';

  const pushRegistration = useCallback(async (): Promise<ServiceWorkerRegistration | null> => {
    if (!pushSupported) return null;
    try {
      return await navigator.serviceWorker.ready;
    } catch {
      return null;
    }
  }, [pushSupported]);

  const loadPush = useCallback(async () => {
    try {
      setPushInfo(await nexus.pushState());
    } catch {
      setPushInfo(null);
      return;
    }
    const registration = await pushRegistration();
    if (!registration) {
      setPushSubscribed(false);
      return;
    }
    const existing = await registration.pushManager.getSubscription();
    setPushSubscribed(Boolean(existing));
  }, [pushRegistration]);

  useEffect(() => {
    void loadPush();
  }, [loadPush]);

  const enablePush = useCallback(async () => {
    if (pushBusy) return;
    const publicKey = pushInfo?.public_key;
    if (!publicKey) {
      setPushMessage(
        t('settings.remote.push.unavailable', {
          defaultValue: 'O serviço de notificações não está disponível neste host.',
        }),
      );
      return;
    }
    setPushBusy(true);
    setPushMessage('');
    try {
      const permission = await Notification.requestPermission();
      if (permission !== 'granted') {
        setPushMessage(
          t('settings.remote.push.blocked', {
            defaultValue: 'Permissão negada. Autorize notificações nas configurações do navegador.',
          }),
        );
        return;
      }
      const registration = await pushRegistration();
      if (!registration) {
        setPushMessage(
          t('settings.remote.push.unsupported', {
            defaultValue: 'Este navegador não expõe notificações em segundo plano.',
          }),
        );
        return;
      }
      const subscription =
        (await registration.pushManager.getSubscription()) ??
        (await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: vapidKeyBytes(publicKey),
        }));
      const serialized = subscription.toJSON();
      const p256dh = serialized.keys?.p256dh;
      const auth = serialized.keys?.auth;
      if (!p256dh || !auth) throw new Error('subscription keys missing');
      await nexus.subscribePush({
        endpoint: subscription.endpoint,
        keys: { p256dh, auth },
      });
      setPushSubscribed(true);
      setPushMessage(
        t('settings.remote.push.enabled', {
          defaultValue: 'Notificações ativadas neste dispositivo.',
        }),
      );
      await loadPush();
    } catch (e) {
      setPushMessage(
        e instanceof Error
          ? e.message
          : t('settings.remote.push.error', {
              defaultValue: 'Não foi possível ativar as notificações.',
            }),
      );
    } finally {
      setPushBusy(false);
    }
  }, [pushBusy, pushInfo, pushRegistration, loadPush, t]);

  const disablePush = useCallback(async () => {
    if (pushBusy) return;
    setPushBusy(true);
    setPushMessage('');
    try {
      const registration = await pushRegistration();
      const subscription = await registration?.pushManager.getSubscription();
      if (subscription) {
        await nexus.unsubscribePush(subscription.endpoint).catch(() => undefined);
        await subscription.unsubscribe();
      }
      setPushSubscribed(false);
      setPushMessage(
        t('settings.remote.push.disabled', {
          defaultValue: 'Notificações desativadas neste dispositivo.',
        }),
      );
      await loadPush();
    } catch {
      setPushMessage(
        t('settings.remote.push.error', {
          defaultValue: 'Não foi possível ativar as notificações.',
        }),
      );
    } finally {
      setPushBusy(false);
    }
  }, [pushBusy, pushRegistration, loadPush, t]);

  const sendPushTest = useCallback(async () => {
    if (pushBusy) return;
    setPushBusy(true);
    setPushMessage('');
    try {
      const result = await nexus.sendPushTest();
      setPushMessage(
        result.attempted === 0
          ? t('settings.remote.push.noSubscribers', {
              defaultValue: 'Nenhum dispositivo inscrito para receber notificações.',
            })
          : t('settings.remote.push.testSent', {
              defaultValue: 'Teste enviado para {{count}} dispositivo(s).',
              count: result.delivered,
            }),
      );
    } catch (e) {
      setPushMessage(
        e instanceof Error
          ? e.message
          : t('settings.remote.push.error', {
              defaultValue: 'Não foi possível ativar as notificações.',
            }),
      );
    } finally {
      setPushBusy(false);
    }
  }, [pushBusy, t]);

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
    // Full control requires an explicit consent acknowledgement; a read-only
    // invitation is the safe default.
    if (inviteRole === 'controller' && !consentGiven) {
      setError(
        t('settings.remote.errors.consentRequired', {
          defaultValue: 'Confirme o acesso completo antes de gerar um convite de controle.',
        }),
      );
      return;
    }
    setLoading(true);
    setError('');
    try {
      const data = await nexus.createTunnelInvite(inviteRole);
      setQrData(data);
      setShowQR(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : t('settings.remote.errors.qr'));
    } finally {
      setLoading(false);
    }
  }, [t, loading, inviteRole, consentGiven]);

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
  // A published private origin (tailnet or private DNS) is a stable address
  // that needs no Cloudflare tunnel: the device pairs against it directly.
  const accessMode = remoteAccessMode(active, status?.remote_origin);
  const privateOrigin = status?.remote_origin?.trim() || '';
  const privateMode = accessMode === 'private';
  const pairable = active || privateMode;

  return (
    <>
      {/* Remote Access Card */}
      <Card className={`nx-settings-card ${styles.gridFullWidth}`} data-tour="remote-tunnel-card">
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

              <p className={styles.tunnelOriginHint}>
                {tunnelOriginKind(status?.stable_url) === 'stable' ? (
                  <>
                    <strong>{t('settings.remote.stableUrl')}</strong> —{' '}
                    {t('settings.remote.stableUrlHint')}
                  </>
                ) : (
                  <>
                    <strong>{t('settings.remote.ephemeralUrl')}</strong> —{' '}
                    {t('settings.remote.ephemeralUrlHint')}
                  </>
                )}
              </p>

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
              {privateMode && (
                <>
                  <div className={styles.tunnelUrl}>
                    <code>{privateOrigin}</code>
                  </div>

                  <p className={styles.tunnelOriginHint}>
                    <strong>{t('settings.remote.privateUrl')}</strong> —{' '}
                    {t('settings.remote.privateUrlHint')}
                  </p>
                </>
              )}

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

              {privateMode && (
                <Button tone="brand" onClick={handleShowQR} disabled={loading}>
                  <Smartphone size={14} />
                  {showQR ? t('settings.remote.hideQR') : t('settings.remote.mobileQR')}
                </Button>
              )}

              <Button tone="ghost" onClick={refresh} disabled={loading}>
                <RefreshCw size={14} />
              </Button>
            </div>
          )}
        </div>

        {pairable && (
          <fieldset className={styles.tunnelControls}>
            <legend className={styles.mutedTextSmall}>
              {t('settings.remote.role.legend', {
                defaultValue: 'Permissão do próximo convite',
              })}
            </legend>
            <div className={styles.tunnelActions}>
              <label className={styles.mutedTextSmall}>
                <input
                  type="radio"
                  name="nexus-remote-role"
                  value="viewer"
                  checked={inviteRole === 'viewer'}
                  onChange={() => setInviteRole('viewer')}
                />{' '}
                {t('settings.remote.role.viewer', {
                  defaultValue: 'Somente leitura (recomendado)',
                })}
              </label>
              <label className={styles.mutedTextSmall}>
                <input
                  type="radio"
                  name="nexus-remote-role"
                  value="controller"
                  checked={inviteRole === 'controller'}
                  onChange={() => setInviteRole('controller')}
                />{' '}
                {t('settings.remote.role.controller', {
                  defaultValue: 'Controle completo (executa comandos)',
                })}
              </label>
            </div>
            {inviteRole === 'controller' && (
              <label className={styles.mutedTextSmall}>
                <input
                  type="checkbox"
                  checked={consentGiven}
                  onChange={(event) => setConsentGiven(event.target.checked)}
                />{' '}
                {t('settings.remote.role.consent', {
                  defaultValue:
                    'Confirmo que este dispositivo pode executar comandos e alterar o workspace.',
                })}
              </label>
            )}
          </fieldset>
        )}

        {status?.cloudflared_version && (
          <p className={`nx-muted-copy ${styles.mutedTextSmall}`}>
            {t('settings.remote.statusVersion', { version: status.cloudflared_version })}
          </p>
        )}
      </Card>

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
              <InlineAlert tone={qrData.role === 'viewer' ? 'info' : 'warning'}>
                {qrData.role === 'viewer'
                  ? t('settings.remote.viewerWarning', {
                      defaultValue:
                        'Este convite é somente leitura: o dispositivo pareado não pode executar comandos nem alterar o workspace.',
                    })
                  : t('settings.remote.fullControlWarning', {
                      defaultValue:
                        'Este convite concede acesso completo ao workspace. Compartilhe somente com alguém de confiança.',
                    })}
              </InlineAlert>
              <p className={styles.qrLabel}>{t('settings.remote.accessURL')}</p>
              <code className={styles.qrUrl}>{qrData.url}</code>

              {qrData.access_code && (
                <div className={styles.accessCodeBlock}>
                  <p className={styles.qrLabel}>{t('settings.remote.accessCode')}</p>
                  <div className={styles.accessCodeRow}>
                    <code className={styles.accessCodeValue}>
                      {formatAccessCode(qrData.access_code)}
                    </code>
                    <Button
                      tone="ghost"
                      onClick={() => {
                        void navigator.clipboard.writeText(qrData.access_code).catch(() =>
                          setError(
                            t('settings.remote.errors.copy', {
                              defaultValue: 'Não foi possível copiar o convite.',
                            }),
                          ),
                        );
                      }}
                    >
                      <Copy size={14} />
                      {t('settings.remote.copyCode', { defaultValue: 'Copiar código' })}
                    </Button>
                  </div>
                  <p className={styles.mutedTextSmall}>
                    {t('settings.remote.accessCodeHint', {
                      defaultValue:
                        'Digite este código no dispositivo remoto para parear sem escanear. Uso único, válido por 15 minutos.',
                    })}
                  </p>
                </div>
              )}

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

      {/* Push notifications Card */}
      <Card className={`nx-settings-card ${styles.gridFullWidth}`} data-tour="remote-push-card">
        <div className="nx-settings-card__title">
          <Bell size={17} />
          <div>
            <strong>
              {t('settings.remote.push.title', { defaultValue: 'Notificações no celular' })}
            </strong>
            <small>
              {t('settings.remote.push.subtitle', {
                defaultValue:
                  'Receba alertas mesmo com o Nexus fechado no aparelho. O conteúdo é cifrado ponta a ponta pela Web Push.',
              })}
            </small>
          </div>
        </div>

        {!pushSupported ? (
          <InlineAlert tone="info">
            {t('settings.remote.push.unsupported', {
              defaultValue: 'Este navegador não expõe notificações em segundo plano.',
            })}
          </InlineAlert>
        ) : (
          <>
            {pushMessage && <InlineAlert tone="info">{pushMessage}</InlineAlert>}
            <div className={styles.tunnelActions}>
              {pushSubscribed ? (
                <Button tone="danger" onClick={() => void disablePush()} disabled={pushBusy}>
                  {t('settings.remote.push.disable', { defaultValue: 'Desativar notificações' })}
                </Button>
              ) : (
                <Button tone="brand" onClick={() => void enablePush()} disabled={pushBusy}>
                  <Bell size={14} />
                  {t('settings.remote.push.enable', { defaultValue: 'Ativar notificações' })}
                </Button>
              )}
              {devicesManagedHere && pushSubscribed && (
                <Button tone="ghost" onClick={() => void sendPushTest()} disabled={pushBusy}>
                  {t('settings.remote.push.test', { defaultValue: 'Enviar teste' })}
                </Button>
              )}
            </div>
          </>
        )}
      </Card>

      {/* Paired devices Card */}
      <Card className={`nx-settings-card ${styles.gridFullWidth}`} data-tour="remote-devices-card">
        <div className="nx-settings-card__title">
          <Smartphone size={17} />
          <div>
            <strong>
              {t('settings.remote.devices.title', { defaultValue: 'Dispositivos pareados' })}
            </strong>
            <small>
              {t('settings.remote.devices.subtitle', {
                defaultValue:
                  'Revogue um dispositivo individualmente sem encerrar as outras sessões.',
              })}
            </small>
          </div>
        </div>

        {!devicesManagedHere ? (
          <InlineAlert tone="info">
            {t('settings.remote.devices.hostOnly', {
              defaultValue:
                'Somente o dispositivo principal do Nexus pode listar e revogar dispositivos pareados.',
            })}
          </InlineAlert>
        ) : devices.length === 0 ? (
          <p className={`nx-muted-copy ${styles.mutedTextSmall}`}>
            {t('settings.remote.devices.empty', {
              defaultValue: 'Nenhum dispositivo pareado no momento.',
            })}
          </p>
        ) : (
          <ul className={styles.tunnelInfoList}>
            {devices.map((device) => (
              <li key={device.device_id} className={styles.accessCodeRow}>
                <span>
                  <strong>{device.label}</strong>
                  {device.last_seen_at && (
                    <small className={styles.mutedTextSmall}>
                      {' · '}
                      {t('settings.remote.devices.lastSeen', {
                        defaultValue: 'visto em {{when}}',
                        when: formatDeviceSeen(device.last_seen_at),
                      })}
                    </small>
                  )}
                </span>
                <Button
                  tone="danger"
                  onClick={() => void handleRevokeDevice(device.device_id)}
                  disabled={revokingDeviceId === device.device_id}
                >
                  {revokingDeviceId === device.device_id
                    ? t('settings.remote.devices.revoking', { defaultValue: 'Revogando…' })
                    : t('settings.remote.devices.revoke', { defaultValue: 'Revogar' })}
                </Button>
              </li>
            ))}
          </ul>
        )}
      </Card>

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
