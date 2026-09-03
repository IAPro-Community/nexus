import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { describe, expect, it, vi } from 'vitest';

type WorkerListener = (event: any) => void;
type WorkerClient = {
  url: string;
  focus?: () => Promise<unknown>;
  navigate?: (url: string) => Promise<unknown>;
};

type NotificationOptions = {
  body?: string;
  data?: {
    attentionEventId?: string;
    projectId?: string;
    reason?: string;
    runtimeId?: string;
    view?: string;
  };
};

function loadWorker() {
  const listeners = new Map<string, WorkerListener>();
  const showNotification = vi.fn<(title: string, options: NotificationOptions) => Promise<void>>(async () => undefined);
  const openWindow = vi.fn<(url: string) => Promise<void>>(async () => undefined);
  const matchAll = vi.fn<() => Promise<WorkerClient[]>>(async () => []);
  const consoleInfo = vi.fn();
  const consoleWarn = vi.fn();
  const source = readFileSync(new URL('../../public/nexus-service-worker.js', import.meta.url), 'utf8');

  const self = {
    location: { origin: 'https://nexus.local', pathname: '/nexus-service-worker.js' },
    registration: { showNotification },
    clients: { matchAll, openWindow },
    addEventListener: (type: string, listener: WorkerListener) => {
      listeners.set(type, listener);
    },
    console: { info: consoleInfo, warn: consoleWarn },
  };

  vm.runInNewContext(source, {
    URL,
    self,
    clients: self.clients,
    registration: self.registration,
    location: self.location,
    console: self.console,
  });

  return { listeners, showNotification, openWindow, matchAll, consoleInfo, consoleWarn };
}

describe('nexus-service-worker', () => {
  it('focuses and navigates an existing Nexus client instead of opening a new one', async () => {
    const worker = loadWorker();
    const focus = vi.fn(async () => undefined);
    const navigate = vi.fn(async () => undefined);
    const close = vi.fn();
    worker.matchAll.mockResolvedValue([
      {
        url: 'https://nexus.local/?project=old&view=overview',
        focus,
        navigate,
      },
    ]);

    let pending: Promise<unknown> | undefined;
    worker.listeners.get('notificationclick')?.({
      notification: {
        close,
        data: {
          projectId: 'prj-7',
          view: 'work',
        },
      },
      waitUntil(value: Promise<unknown>) {
        pending = value;
      },
    });

    await pending;

    expect(close).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith('https://nexus.local/?project=prj-7&view=work');
    expect(focus).toHaveBeenCalledTimes(1);
    expect(worker.openWindow).not.toHaveBeenCalled();
  });

  it('opens a new Nexus window only when no client exists', async () => {
    const worker = loadWorker();
    const close = vi.fn();

    let pending: Promise<unknown> | undefined;
    worker.listeners.get('notificationclick')?.({
      notification: {
        close,
        data: {
          projectId: 'prj-9',
          view: 'overview',
        },
      },
      waitUntil(value: Promise<unknown>) {
        pending = value;
      },
    });

    await pending;

    expect(close).toHaveBeenCalledTimes(1);
    expect(worker.openWindow).toHaveBeenCalledWith('https://nexus.local/?project=prj-9&view=overview');
  });

  it('shows a web push notification only when there are no active clients', async () => {
    const worker = loadWorker();
    worker.matchAll.mockResolvedValue([]);

    let pending: Promise<unknown> | undefined;
    worker.listeners.get('push')?.({
      data: {
        json: () => ({
          runtimeId: 'rt-1',
          projectId: 'prj-1',
          projectName: 'Payments',
          attentionEventId: 'evt-1',
          reason: 'QUESTION',
          context: 'Need approval',
          view: 'work',
        }),
      },
      waitUntil(value: Promise<unknown>) {
        pending = value;
      },
    });

    await pending;

    expect(worker.showNotification).toHaveBeenCalledTimes(1);
    const firstCall = worker.showNotification.mock.calls[0];
    expect(firstCall).toBeDefined();
    const [title, options] = firstCall!;
    expect(title).toContain('Payments');
    expect(options.body).toBe('Need approval');
    expect(options.data).toEqual({
      attentionEventId: 'evt-1',
      projectId: 'prj-1',
      reason: 'QUESTION',
      runtimeId: 'rt-1',
      view: 'work',
    });
  });

  it('falls back to a semantic title when the dynamic title looks secret-bearing', async () => {
    const worker = loadWorker();
    worker.matchAll.mockResolvedValue([]);

    let pending: Promise<unknown> | undefined;
    worker.listeners.get('push')?.({
      data: {
        json: () => ({
          runtimeId: 'rt-2',
          projectId: 'prj-2',
          projectName: 'Payments',
          reason: 'QUESTION',
          context: 'transcript: user pasted credentials',
          summary: 'Need approval',
          dynamicTitle: 'Bearer sk-live-secret',
          view: 'work',
        }),
      },
      waitUntil(value: Promise<unknown>) {
        pending = value;
      },
    });

    await pending;

    const [title, options] = worker.showNotification.mock.calls[0]!;
    expect(title).toBe('[Nexus | Payments] ❓ Pergunta Requer Atenção');
    expect(options.body).toBe('Need approval');
  });

  it('does not log subscription secrets on pushsubscriptionchange', async () => {
    const worker = loadWorker();

    let pending: Promise<unknown> | undefined;
    worker.listeners.get('pushsubscriptionchange')?.({
      oldSubscription: {
        endpoint: 'https://push.example/subscriptions/secret-endpoint',
        toJSON: () => ({ keys: { p256dh: 'secret-public-key', auth: 'secret-auth-key' } }),
      },
      waitUntil(value: Promise<unknown>) {
        pending = value;
      },
    });

    await pending;

    const emitted = worker.consoleInfo.mock.calls.flat().join(' ');
    expect(emitted).toContain('pushsubscriptionchange');
    expect(emitted).not.toContain('secret-endpoint');
    expect(emitted).not.toContain('secret-public-key');
    expect(emitted).not.toContain('secret-auth-key');
  });
});
