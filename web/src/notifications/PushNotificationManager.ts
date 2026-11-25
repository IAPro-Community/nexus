import { formatAttentionPushBody, shouldSendBrowserAttentionPush } from './attentionPushCopy';
import { getPlatformBridge } from '../platform';
import {
  clearNotificationEpisode,
  isNotificationCategoryEnabled,
  markNotificationEpisodeDelivered,
} from './attentionNotificationPolicy';
import { loadNotificationPrefs } from './notificationPrefs';

export interface PushNotificationPayload {
  runtimeId: string;
  projectName?: string;
  agentName?: string;
  reason: 'QUESTION' | 'APPROVAL' | 'TASK_COMPLETED' | 'WORKING' | 'IDLE' | 'ERROR' | string;
  context: string;
  dynamicTitle?: string;
  fingerprint?: string;
  promptKind?: string;
  onClick?: () => void;
}

class PushNotificationService {
  private permission: NotificationPermission =
    typeof window !== 'undefined' && 'Notification' in window ? Notification.permission : 'default';

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
      return perm;
    } catch {
      return this.permission;
    }
  }

  public confirmEnabled(projectName?: string): void {
    if (!this.isSupported() || this.getPermission() !== 'granted') return;
    try {
      new Notification(`Nexus · ${projectName || 'Projeto'}`, {
        body: 'Notificações do navegador ativadas. Com a aba em segundo plano, o Nexus avisa quando um agente espera por você.',
        icon: './nexus-icon.png',
        tag: 'nexus-permission-ok',
      });
    } catch (e) {
      console.warn('[PushNotificationService] Failed to show browser notification:', e);
    }
  }

  public sendPush(payload: PushNotificationPayload): void {
    if (!this.isSupported()) return;

    const documentHidden = typeof document === 'undefined' ? true : document.hidden;
    const prefs = loadNotificationPrefs();

    if (
      !shouldSendBrowserAttentionPush({
        permission: this.getPermission(),
        documentHidden,
        reason: payload.reason,
        context: payload.context,
        promptKind: payload.promptKind,
        notificationsEnabled: prefs.notificationsEnabled,
      })
    ) {
      return;
    }
    if (!isNotificationCategoryEnabled(payload.reason, prefs.categories)) return;

    const fingerprint =
      payload.fingerprint ||
      `${payload.runtimeId}|${payload.promptKind || ''}|${payload.context.toLowerCase().replace(/\s+/g, ' ')}`;
    if (!markNotificationEpisodeDelivered(fingerprint)) return;

    const proj = payload.projectName || 'Projeto';
    const title = payload.reason === 'ERROR' ? `Nexus · ${proj} · erro` : `Nexus · ${proj}`;
    const body = formatAttentionPushBody({
      reason: payload.reason,
      context: payload.context,
      promptKind: payload.promptKind,
      agentName: payload.agentName,
      projectName: payload.projectName,
    });
    if (!body) return;

    try {
      const bridge = getPlatformBridge();
      if (bridge.getCapabilities().native) {
        void bridge.showNotification({
          title,
          body,
          icon: './nexus-icon.png',
          silent: false,
        });
        return;
      }

      const notif = new Notification(title, {
        body,
        icon: './nexus-icon.png',
        tag: `nexus-${fingerprint}`,
        requireInteraction: false,
        silent: false,
      });

      // Auto-dismiss after a short window so the OS tray does not keep a stale y/N forever.
      window.setTimeout(() => {
        try {
          notif.close();
        } catch {
          // ignore
        }
      }, 20_000);

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

  /** Test helper / recovery when the wait clears. */
  public clearFingerprint(fingerprint: string): void {
    clearNotificationEpisode(fingerprint);
  }
}

export const pushNotifications = new PushNotificationService();
