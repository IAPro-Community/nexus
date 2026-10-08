import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Copy, Globe, RefreshCw, Smartphone, Wifi, WifiOff } from 'lucide-react';
import QRCode from 'react-qr-code';
import { Button, Card, InlineAlert } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { TunnelStatus } from '../../types';
import { useTranslation } from 'react-i18next';
import { asArray } from '../../lib/safeArray';
import styles from './SettingsSurface.module.scss';

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
    expires_in_seconds: number;
  } | null>(null);

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
            defaultValue: 'Na mesma rede, o acesso autenticado exige HTTPS confiável. Utilize um proxy TLS ou o túnel HTTPS; não é seguro transportar cookies de sessão por HTTP na rede.',
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
                  defaultValue: 'Este convite concede acesso completo ao workspace. Compartilhe somente com alguém de confiança.',
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
                    setError(t('settings.remote.errors.copy', {
                      defaultValue: 'Não foi possível copiar o convite.',
                    })),
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
