import React, { useEffect, useState } from 'react';
import { ThemeProvider } from '../design-system';
import { initSession } from '../api';
import { NexusHud } from '../features/hud/NexusHud';
import { getPlatformBridge } from '../platform';
import styles from './DesktopHudApp.module.scss';

export const isDesktopHudApp = () =>
  typeof window !== 'undefined' && Boolean(window.go?.desktop?.HUDController);

export const DesktopHudApp: React.FC = () => {
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
      void window.go?.desktop?.HUDController?.CaptureHUDWindowState?.();
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
    const controller = window.go?.desktop?.HUDController;
    if (!controller?.GetHUDCapabilities) return () => undefined;
    void controller
      .GetHUDCapabilities()
      .then(async (caps) => {
        if (!active) return;
        setAutoStartSupported(Boolean(caps.autoStart));
        if (caps.autoStart && controller.GetHUDAutoStartEnabled) {
          const enabled = await controller.GetHUDAutoStartEnabled();
          if (active) setAutoStartEnabled(enabled);
        }
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  const toggleAutoStart = () => {
    const controller = window.go?.desktop?.HUDController;
    if (!controller?.SetHUDAutoStartEnabled || autoStartPending) return;
    const next = !autoStartEnabled;
    setAutoStartPending(true);
    void controller
      .SetHUDAutoStartEnabled(next)
      .then(() => setAutoStartEnabled(next))
      .finally(() => setAutoStartPending(false));
  };

  const openNexus = () => {
    void (async () => {
      const bridge = getPlatformBridge();
      const bootstrap = await bridge.getBootstrapInfo?.();
      if (bootstrap?.serverUrl) {
        await bridge.openExternal(bootstrap.serverUrl);
      }
    })();
  };

  if (!ready) {
    return <div className={styles.status}>Nexus HUD</div>;
  }
  if (!authenticated) {
    return (
      <div className={styles.status} role="alert">
        HUD offline · não foi possível anexar à sessão local do Nexus.
      </div>
    );
  }

  return (
    <ThemeProvider>
      <main className={styles.root}>
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
      </main>
    </ThemeProvider>
  );
};
