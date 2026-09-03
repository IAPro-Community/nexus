/**
 * Notification transport abstraction.
 *
 * Supports multiple notification delivery methods:
 * - IN_APP: Toast notifications within the Nexus UI
 * - BROWSER: Browser Notification API
 * - WEB_PUSH: Service Worker Push API
 * - NATIVE_OS: Native OS notifications (future)
 */

export type NotificationTransportType = 'IN_APP' | 'BROWSER' | 'WEB_PUSH' | 'NATIVE_OS';

export interface NotificationPayload {
  title: string;
  body: string;
  icon?: string;
  tag?: string;
  data?: Record<string, unknown>;
}

export interface NotificationTransport {
  type: NotificationTransportType;
  isAvailable(): boolean;
  send(payload: NotificationPayload): Promise<void>;
}

/**
 * IN_APP transport - shows toast notifications within the Nexus UI.
 */
export class InAppTransport implements NotificationTransport {
  type: NotificationTransportType = 'IN_APP';

  isAvailable(): boolean {
    return true;
  }

  async send(payload: NotificationPayload): Promise<void> {
    // This will be integrated with the existing InAppNotificationCenter
    console.log('[InApp]', payload.title, payload.body);
  }
}

/**
 * BROWSER transport - uses the Browser Notification API.
 */
export class BrowserTransport implements NotificationTransport {
  type: NotificationTransportType = 'BROWSER';

  isAvailable(): boolean {
    return typeof window !== 'undefined' && 'Notification' in window;
  }

  async send(payload: NotificationPayload): Promise<void> {
    if (!this.isAvailable()) return;

    const permission = await Notification.requestPermission();
    if (permission !== 'granted') return;

    new Notification(payload.title, {
      body: payload.body,
      icon: payload.icon || '/logo.png',
      tag: payload.tag,
      data: payload.data,
    });
  }
}

/**
 * WEB_PUSH transport - uses Service Worker Push API.
 */
export class WebPushTransport implements NotificationTransport {
  type: NotificationTransportType = 'WEB_PUSH';

  isAvailable(): boolean {
    return (
      typeof window !== 'undefined' &&
      'serviceWorker' in navigator &&
      'PushManager' in window
    );
  }

  async send(_payload: NotificationPayload): Promise<void> {
    if (!this.isAvailable()) return;

    const _registration = await navigator.serviceWorker.ready;
    // The actual push is handled by the service worker
    // This transport just ensures the service worker is ready
    console.log('[WebPush] Ready to deliver via service worker');
  }
}

/**
 * Select the best available transport based on page visibility.
 */
export function selectTransport(
  visibilityState: DocumentVisibilityState,
  hasFocus: boolean,
): NotificationTransport {
  if (visibilityState === 'visible' && hasFocus) {
    return new InAppTransport();
  }

  if (visibilityState === 'hidden' || !hasFocus) {
    if (new WebPushTransport().isAvailable()) {
      return new WebPushTransport();
    }
    if (new BrowserTransport().isAvailable()) {
      return new BrowserTransport();
    }
  }

  return new InAppTransport();
}
