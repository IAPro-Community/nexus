import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ThemeProvider } from '../design-system';
import { initSession } from '../api';
import { NexusHud } from '../features/hud/NexusHud';
import { getPlatformBridge, hudV3, isDesktopApp } from '../platform';
import styles from './DesktopHudApp.module.scss';

export const isDesktopHudApp = () => {
  if (typeof window === 'undefined' || !isDesktopApp()) return false;
  return new URLSearchParams(window.location.search).get('nativeHud') === '1';
};

export const DesktopHudApp: React.FC = () => {
  const { t } = useTranslation();
  const [ready, setReady] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [autoStartSupported, setAutoStartSupported] = useState(false);
  const [autoStartEnabled, setAutoStartEnabled] = useState(false);
  const [autoStartPending, setAutoStartPending] = useState(false);

  useEffect(() => {
    let active = true;
    void initSession()
      .then((session) => {
        if (!active) return;
        setAuthenticated(session.authenticated);
      })
      .finally(() => {
        if (active) setReady(true);
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const capture = () => {
      void hudV3.captureWindowState().catch(() => undefined);
    };
    window.addEventListener('blur', capture);
    window.addEventListener('beforeunload', capture);
    return () => {
      window.removeEventListener('blur', capture);
      window.removeEventListener('beforeunload', capture);
    };
  }, []);

  useEffect(() => {
    let active = true;
    void hudV3
      .getCapabilities()
      .then(async (caps) => {
        if (!active) return;
        setAutoStartSupported(Boolean(caps.autoStart));
        if (caps.autoStart) {
          const enabled = await hudV3.getAutoStartEnabled();
          if (active) setAutoStartEnabled(enabled);
        }
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  const toggleAutoStart = () => {
    if (autoStartPending) return;
    const next = !autoStartEnabled;
    setAutoStartPending(true);
    void hudV3
      .setAutoStartEnabled(next)
      .then(() => setAutoStartEnabled(next))
      .finally(() => setAutoStartPending(false));
  };

  const openNexus = () => {
    void (async () => {
      const bridge = getPlatformBridge();
      const bootstrap = await bridge.getBootstrapInfo?.();
      if (bootstrap?.serverUrl) await bridge.openExternal(bootstrap.serverUrl);
    })();
  };

  return (
    <ThemeProvider>
      <main className={styles.root}>
        {!ready ? (
          <div className={styles.status} role="status">
            {t('common.loading')} Nexus HUD
          </div>
        ) : !authenticated ? (
          <div className={styles.status} role="alert">
            {t('common.hudUnavailable')}
          </div>
        ) : (
          <NexusHud
            native
            onOpenNexus={openNexus}
            nativeAutoStart={{
              supported: autoStartSupported,
              enabled: autoStartEnabled,
              pending: autoStartPending,
              onToggle: toggleAutoStart,
            }}
          />
        )}
      </main>
    </ThemeProvider>
  );
};
