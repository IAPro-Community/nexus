import React, { Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { NexAvatarView } from './NexAvatarView';
import { NexCompactBar } from './NexCompactBar';
import { NexNotificationBubble } from './NexNotificationBubble';
import { NexSettingsDrawer } from './NexSettingsDrawer';
import type { NexOperationalMood } from './motion/nexBehaviorState';
import {
  ackCapacityAlert,
  defaultPlatformCapabilities,
  defaultPresenceState,
  fetchNexV2Hud,
  loadNexV2Presence,
  presetSizePixels,
  snoozeCapacityAlert,
  updateNexV2AlwaysOnTop,
  updateNexV2CompactBar,
  updateNexV2DoNotDisturb,
  updateNexV2PanelOpen,
  updateNexV2Paused,
  updateNexV2Preset,
  updateNexV2ReducedMotion,
  updateNexV2ShowAvatar,
} from './nexV2Client';
import type {
  NexV2HudAlert,
  NexV2HudState,
  NexV2PlatformCapabilities,
  NexV2PresenceMetricsSnapshot,
  NexV2PresenceState,
  NexV2PresetId,
  NexV2TabId,
} from './nexV2Types';
import styles from './NexPresenceApp.module.scss';

const LazyNexInteractionPanel = React.lazy(() => import('./NexInteractionPanel'));

declare global {
  interface Window {
    __nexV2PresenceMetrics?: NexV2PresenceMetricsSnapshot;
  }
}

export function deriveNexV2Mood(
  hudState: NexV2HudState | null,
  offline: boolean,
): NexOperationalMood {
  if (offline) return 'offline';
  if (!hudState) return 'idle';
  if (hudState.alerts.some((a) => !a.acked && (a.severity || '').toUpperCase() === 'CRITICAL')) {
    return 'critical';
  }
  if (
    hudState.alerts.some((a) => !a.acked) ||
    hudState.runtime_attention.length > 0 ||
    hudState.missions.some((m) => m.needs_user)
  ) {
    return 'attention';
  }
  if (
    hudState.agents.some((a) => ['WORKING', 'RUNNING'].includes((a.status || '').toUpperCase())) ||
    hudState.missions.some((m) => ['RUNNING', 'EXECUTING'].includes((m.state || '').toUpperCase()))
  ) {
    return 'working';
  }
  return 'idle';
}

export const NexPresenceApp: React.FC = () => {
  const { t } = useTranslation();
  const [presence, setPresence] = useState<NexV2PresenceState>(() => defaultPresenceState());
  const [capabilities, setCapabilities] = useState<NexV2PlatformCapabilities>(() =>
    defaultPlatformCapabilities(),
  );
  const [hudState, setHudState] = useState<NexV2HudState | null>(null);
  const [offline, setOffline] = useState(false);
  const [panelOpen, setPanelOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<NexV2TabId>('chat');
  const [dismissedAlerts, setDismissedAlerts] = useState<Record<string, boolean>>({});
  const [chunkLoaded, setChunkLoaded] = useState(false);

  const coldOpenMsRef = useRef<number | null>(null);
  const warmOpenMsRef = useRef<number | null>(null);
  const openStartRef = useRef<number | null>(null);
  const toggleCountRef = useRef(0);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const pollInFlightRef = useRef(false);

  const pollHud = useCallback(async () => {
    if (pollInFlightRef.current) return;
    pollInFlightRef.current = true;
    try {
      const result = await fetchNexV2Hud();
      setOffline(result.offline);
      setHudState(result.state);
    } finally {
      pollInFlightRef.current = false;
    }
  }, []);

  useEffect(() => {
    let active = true;
    void loadNexV2Presence().then(({ state, capabilities: caps }) => {
      if (!active) return;
      setPresence(state);
      setCapabilities(caps);
    });
    void pollHud();
    return () => {
      active = false;
    };
  }, [pollHud]);

  useEffect(() => {
    if (presence.paused) return;
    const intervalMs = hudState?.system.cache_ttl_millis || 15_000;
    const timer = window.setInterval(() => {
      if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return;
      void pollHud();
    }, intervalMs);
    return () => window.clearInterval(timer);
  }, [hudState?.system.cache_ttl_millis, pollHud, presence.paused]);

  const setPanelVisibility = useCallback((nextOpen: boolean, tab?: NexV2TabId) => {
    if (tab) {
      setActiveTab(tab);
    }
    setPanelOpen((prev) => {
      if (prev === nextOpen) return prev;
      toggleCountRef.current += 1;
      if (nextOpen && typeof performance !== 'undefined') {
        openStartRef.current = performance.now();
        previousFocusRef.current =
          typeof document !== 'undefined' ? (document.activeElement as HTMLElement | null) : null;
      } else if (!nextOpen && previousFocusRef.current) {
        const returnTo = previousFocusRef.current;
        previousFocusRef.current = null;
        window.setTimeout(() => returnTo.focus(), 0);
      }
      return nextOpen;
    });
    void updateNexV2PanelOpen(nextOpen);
  }, []);

  useEffect(() => {
    if (panelOpen && openStartRef.current !== null && typeof performance !== 'undefined') {
      const elapsed = Number((performance.now() - openStartRef.current).toFixed(2));
      openStartRef.current = null;
      setChunkLoaded(true);
      if (coldOpenMsRef.current === null) {
        coldOpenMsRef.current = elapsed;
      } else {
        warmOpenMsRef.current = elapsed;
      }
    }
  }, [panelOpen]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.ctrlKey && event.shiftKey && event.key.toLowerCase() === 'n') {
        event.preventDefault();
        setPanelVisibility(!panelOpen);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [panelOpen, setPanelVisibility]);

  const mood = useMemo(() => deriveNexV2Mood(hudState, offline), [hudState, offline]);
  const presetPx = presetSizePixels(presence.placement.presetId);
  const topAlert: NexV2HudAlert | null = useMemo(() => {
    if (!hudState) return null;
    return hudState.alerts.find((a) => !a.acked && !dismissedAlerts[a.fingerprint]) ?? null;
  }, [dismissedAlerts, hudState]);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.__nexV2PresenceMetrics = {
        panelOpen,
        panelOpenColdMs: coldOpenMsRef.current,
        panelOpenWarmMs: warmOpenMsRef.current,
        panelToggleCount: toggleCountRef.current,
        activeTab,
        settingsOpen,
        avatarVisible: presence.showAvatar,
        compactBar: presence.compactBar || !presence.showAvatar,
        doNotDisturb: presence.doNotDisturb,
        offline,
        mood,
        chunkLoaded,
      };
    }
  }, [
    activeTab,
    chunkLoaded,
    mood,
    offline,
    panelOpen,
    presence.compactBar,
    presence.doNotDisturb,
    presence.showAvatar,
    settingsOpen,
  ]);

  const showCompactBar = presence.compactBar || !presence.showAvatar;

  return (
    <main className={styles.shell} aria-label={t('nexV2.presenceShellLabel')} data-nex-presence>
      {offline && (
        <section className={styles.offlineBanner} role="alert" data-nex-offline-banner>
          <strong>{t('nexV2.offlineBannerTitle')}</strong>
          <p className={styles.offlineBody}>{t('nexV2.offlineBannerBody')}</p>
        </section>
      )}

      {panelOpen && (
        <Suspense fallback={<div className={styles.loadingFallback}>...</div>}>
          <LazyNexInteractionPanel
            activeTab={activeTab}
            hudState={hudState}
            offline={offline}
            onSelectTab={setActiveTab}
            onClose={() => setPanelVisibility(false)}
            onOpenSettings={() => setSettingsOpen((prev) => !prev)}
            onAckAlert={(fp) => void ackCapacityAlert(fp).then(() => pollHud())}
            onSnoozeAlert={(fp) => void snoozeCapacityAlert(fp, 30).then(() => pollHud())}
          />
        </Suspense>
      )}

      <NexSettingsDrawer
        open={settingsOpen}
        state={presence}
        capabilities={capabilities}
        onClose={() => setSettingsOpen(false)}
        onPresetChange={(presetId: NexV2PresetId) =>
          void updateNexV2Preset(presetId, presence).then(setPresence)
        }
        onToggleShowAvatar={(visible) =>
          void updateNexV2ShowAvatar(visible, presence).then(setPresence)
        }
        onToggleCompactBar={(enabled) =>
          void updateNexV2CompactBar(enabled, presence).then(setPresence)
        }
        onToggleAlwaysOnTop={(enabled) =>
          void updateNexV2AlwaysOnTop(enabled, presence).then(setPresence)
        }
        onToggleDoNotDisturb={(enabled) =>
          void updateNexV2DoNotDisturb(enabled, presence).then(setPresence)
        }
        onTogglePaused={(paused) => void updateNexV2Paused(paused, presence).then(setPresence)}
        onToggleReducedMotion={(enabled) =>
          void updateNexV2ReducedMotion(enabled, presence).then(setPresence)
        }
      />

      <NexNotificationBubble
        alert={topAlert}
        doNotDisturb={presence.doNotDisturb}
        panelOpen={panelOpen}
        offline={offline}
        onDismiss={(fp) => setDismissedAlerts((prev) => ({ ...prev, [fp]: true }))}
        onOpenAttention={() => setPanelVisibility(true, 'attention')}
      />

      {showCompactBar && (
        <NexCompactBar
          mood={mood}
          hudState={hudState}
          offline={offline}
          doNotDisturb={presence.doNotDisturb}
          panelOpen={panelOpen}
          onTogglePanel={() => setPanelVisibility(!panelOpen)}
          onOpenSettings={() => setSettingsOpen((prev) => !prev)}
        />
      )}

      {presence.showAvatar && (
        <div className={styles.avatarRow}>
          <NexAvatarView
            mood={mood}
            motionEnabled={!presence.paused && !presence.reducedMotion}
            presetSizePx={presetPx}
            onActivate={() => setPanelVisibility(!panelOpen)}
          />
          {!showCompactBar && (
            <button
              type="button"
              className={styles.quickSettingsBtn}
              aria-label={t('nexV2.openSettingsLabel')}
              data-nex-quick-settings
              onClick={() => setSettingsOpen((prev) => !prev)}
            >
              ...
            </button>
          )}
        </div>
      )}
    </main>
  );
};
