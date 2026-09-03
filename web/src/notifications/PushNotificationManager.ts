import {
  prepareNotificationEnvelope,
  type NotificationPolicyInput,
} from './notificationPolicy';

export interface PushNotificationPayload {
  runtimeId: string;
  projectId?: string;
  projectName?: string;
  reason: 'QUESTION' | 'APPROVAL' | 'TASK_COMPLETED' | 'WORKING' | 'IDLE' | 'ERROR' | string;
  context?: string;
  summary?: string;
  dynamicTitle?: string;
  attentionEventId?: string;
  view?: string;
  credentials?: unknown;
  transcript?: unknown;
  onClick?: () => void;
}

class PushNotificationService {
  private permission: NotificationPermission = typeof window !== 'undefined' && 'Notification' in window ? Notification.permission : 'default';
  private lastNotifiedKey = '';
  private lastNotifiedTime = 0;
  private readonly seenAttentionEvents = new Set<string>();

  public isSupported(): boolean {
    return typeof window !== 'undefined' && 'Notification' in window;
  }

  public getPermission(): NotificationPermission {
    if (this.isSupported()) {
      this.permission = Notification.permission;
    }
    return this.permission;
  }

  public async requestPermission(): Promise<NotificationPermission> {
    if (!this.isSupported()) return 'denied';
    try {
      const perm = await Notification.requestPermission();
      this.permission = perm;
      if (perm === 'granted') {
        await this.ensureServiceWorker();
      }
      return perm;
    } catch {
      return this.permission;
    }
  }

  public sendPush(payload: PushNotificationPayload): void {
    if (!this.isSupported() || this.getPermission() !== 'granted') return;
    const hasFocusFn = (window as unknown as { hasFocus?: () => boolean }).hasFocus;
    const envelope = prepareNotificationEnvelope(payload as NotificationPolicyInput, {
      hasActiveClients: true,
      visible: !document.hidden,
      hasFocus: typeof hasFocusFn === 'function' ? hasFocusFn() : true,
    });

    if (envelope.transport === 'IN_APP') {
      return;
    }

    if (payload.attentionEventId) {
      if (this.seenAttentionEvents.has(envelope.dedupeKey)) {
        return;
      }
      this.seenAttentionEvents.add(envelope.dedupeKey);
    }

    const now = Date.now();
    if (!payload.attentionEventId && this.lastNotifiedKey === envelope.dedupeKey && now - this.lastNotifiedTime < 3000) {
      return;
    }
    this.lastNotifiedKey = envelope.dedupeKey;
    this.lastNotifiedTime = now;

    try {
      const notif = new Notification(envelope.title, {
        body: envelope.body,
        icon: '/logo.png',
        tag: envelope.tag,
        data: envelope.data,
        silent: false,
      });

      notif.onclick = () => {
        try {
          window.focus();
        } catch {
          // Browsers may block focus changes triggered by notifications.
        }
        notif.close();
        if (payload.onClick) {
          payload.onClick();
        }
      };
    } catch (e) {
      console.warn('[PushNotificationService] Failed to show browser notification:', e);
    }
  }

  private async ensureServiceWorker(): Promise<void> {
    if (typeof navigator === 'undefined' || !('serviceWorker' in navigator)) return;
    try {
      await navigator.serviceWorker.register('/nexus-service-worker.js');
    } catch {
      // Browser notifications remain a valid fallback when worker registration
      // is unavailable; WEB_PUSH requires a separately registered worker.
    }
  }
}

export const pushNotifications = new PushNotificationService();
