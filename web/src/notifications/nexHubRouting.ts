import { useEffect, useState } from 'react';

export interface NexHubStatus {
  active: boolean;
  can_notify: boolean;
  do_not_disturb: boolean;
  reason: string;
  last_seen_at?: string;
  expires_at?: string;
}

const DEFAULT_NEX_HUB_STATUS: NexHubStatus = {
  active: false,
  can_notify: false,
  do_not_disturb: false,
  reason: 'nex_offline',
};

const POLL_INTERVAL_MS = 15_000;

let testOverride: NexHubStatus | null = null;
let currentStatus: NexHubStatus = { ...DEFAULT_NEX_HUB_STATUS };
let pollTimer: number | null = null;
let focusListener: (() => void) | null = null;
let inFlight = false;
const listeners = new Set<(status: NexHubStatus) => void>();

function statusSignature(status: NexHubStatus): string {
  return `${status.active ? '1' : '0'}|${status.can_notify ? '1' : '0'}|${status.do_not_disturb ? '1' : '0'}|${status.reason}`;
}

function getEffectiveStatus(): NexHubStatus {
  return testOverride ?? currentStatus;
}

function emitStatus(): void {
  const snapshot = getEffectiveStatus();
  listeners.forEach((listener) => listener(snapshot));
}

export async function refreshNexHubStatus(): Promise<NexHubStatus> {
  if (testOverride !== null) return testOverride;
  if (typeof fetch !== 'function') return currentStatus;
  if (inFlight) return currentStatus;
  inFlight = true;
  const previousSig = statusSignature(currentStatus);
  try {
    const res = await fetch('/api/v1/nex/hub-status', {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
    });
    if (!res.ok) {
      currentStatus = { ...DEFAULT_NEX_HUB_STATUS };
    } else {
      const payload = (await res.json()) as Partial<NexHubStatus>;
      const active = Boolean(payload.active);
      currentStatus = {
        active,
        can_notify: Boolean(payload.can_notify),
        do_not_disturb: Boolean(payload.do_not_disturb),
        reason: payload.reason || (active ? 'nex_active' : 'nex_offline'),
        last_seen_at: payload.last_seen_at,
        expires_at: payload.expires_at,
      };
    }
  } catch {
    // Fail-safe: any network or auth error restores fallback pop-ups immediately.
    currentStatus = { ...DEFAULT_NEX_HUB_STATUS };
  } finally {
    inFlight = false;
  }
  if (statusSignature(currentStatus) !== previousSig) {
    emitStatus();
  }
  return currentStatus;
}

export function subscribeNexHubStatus(listener: (status: NexHubStatus) => void): () => void {
  listeners.add(listener);
  if (listeners.size === 1 && testOverride === null && typeof window !== 'undefined') {
    void refreshNexHubStatus();
    pollTimer = window.setInterval(() => {
      if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return;
      void refreshNexHubStatus();
    }, POLL_INTERVAL_MS);
    focusListener = () => {
      void refreshNexHubStatus();
    };
    window.addEventListener('focus', focusListener);
  }
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0 && typeof window !== 'undefined') {
      if (pollTimer !== null) {
        window.clearInterval(pollTimer);
        pollTimer = null;
      }
      if (focusListener !== null) {
        window.removeEventListener('focus', focusListener);
        focusListener = null;
      }
    }
  };
}

export function useNexHubStatus(): NexHubStatus {
  const [status, setStatus] = useState<NexHubStatus>(() => getEffectiveStatus());
  useEffect(() => {
    setStatus(getEffectiveStatus());
    return subscribeNexHubStatus(setStatus);
  }, []);
  return status;
}

export function isNexHubActive(): boolean {
  return getEffectiveStatus().active;
}

export function setNexHubStatusForTest(status: NexHubStatus | null): void {
  testOverride = status;
  if (status === null) {
    currentStatus = { ...DEFAULT_NEX_HUB_STATUS };
  }
  emitStatus();
}
