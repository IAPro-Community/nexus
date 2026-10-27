import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Camera,
  Globe,
  Keyboard,
  Lock,
  RefreshCw,
  Send,
  Unlock,
} from 'lucide-react';
import { getDesktopAuthToken, getWebSocketEndpoint, request } from '../../api';
import { useTranslation } from 'react-i18next';
import styles from './BrowserSurface.module.scss';

export type NexusBrowserInfo = {
  id: string;
  project_id: string;
  url: string;
  width: number;
  height: number;
};

export type NexusBrowserAction = {
  type: 'navigate' | 'back' | 'forward' | 'reload' | 'click' | 'scroll' | 'type' | 'key';
  url?: string;
  text?: string;
  key?: string;
  x?: number;
  y?: number;
  delta_y?: number;
};

const endpoint = '/api/v1/browser/sessions';
const browserRoute = (id: string) => `${endpoint}/${encodeURIComponent(id)}`;

export const browserAPI = {
  create: (projectId: string, url: string) =>
    request<NexusBrowserInfo>(endpoint, {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, url }),
    }),
  get: (id: string) => request<NexusBrowserInfo>(browserRoute(id)),
  action: (id: string, action: NexusBrowserAction) =>
    request<NexusBrowserInfo>(`${browserRoute(id)}/action`, {
      method: 'POST',
      body: JSON.stringify(action),
    }),
  control: (id: string, mode: 'acquire' | 'renew' | 'release') =>
    request<{ controller: boolean; lease_seconds?: number }>(`${browserRoute(id)}/control`, {
      method: 'POST',
      body: JSON.stringify({ mode }),
    }),
  snapshot: (id: string) =>
    request<{ url: string; title: string; text: string }>(`${browserRoute(id)}/snapshot`),
};

export const BrowserSurface: React.FC<{
  projectId: string;
  initialSessionId?: string;
  initialUrl?: string;
  onSessionReady?: (id: string) => void;
}> = ({ projectId, initialSessionId, initialUrl = 'http://localhost:3000', onSessionReady }) => {
  const { t } = useTranslation();
  const [session, setSession] = useState<NexusBrowserInfo | null>(null);
  const [address, setAddress] = useState(initialUrl);
  const [text, setText] = useState('');
  const [frame, setFrame] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [controller, setController] = useState(false);
  const [wantsControl, setWantsControl] = useState(true);
  const [live, setLive] = useState(false);
  const [visible, setVisible] = useState(true);
  const [snapshot, setSnapshot] = useState('');
  const rootRef = useRef<HTMLElement>(null);
  const frameObjectURL = useRef('');
  const started = useRef(false);
  const mounted = useRef(false);
  const onReady = useRef(onSessionReady);
  const touchStart = useRef<number | null>(null);
  const ignoreClick = useRef(false);
  const lastFrameAt = useRef(0);

  useEffect(() => {
    onReady.current = onSessionReady;
  }, [onSessionReady]);

  useEffect(() => {
    const target = rootRef.current;
    if (!target || typeof IntersectionObserver === 'undefined') return;
    const observer = new IntersectionObserver(([entry]) => setVisible(entry.isIntersecting), {
      threshold: 0.01,
    });
    observer.observe(target);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    mounted.current = true;
    if (!started.current) {
      started.current = true;
      void (async () => {
        try {
          let next: NexusBrowserInfo;
          if (initialSessionId) {
            try {
              next = await browserAPI.get(initialSessionId);
              if (next.project_id !== projectId) throw new Error('stale project session');
            } catch {
              next = await browserAPI.create(projectId, initialUrl);
            }
          } else {
            next = await browserAPI.create(projectId, initialUrl);
          }
          if (!mounted.current) {
            // Closing a new surface during Chromium startup must not leak
            // one of the four managed browser slots.
            void request<{ status: string }>(`${browserRoute(next.id)}`, {
              method: 'DELETE',
            }).catch(() => undefined);
            return;
          }
          setSession(next);
          setAddress(next.url);
          onReady.current?.(next.id);
        } catch (err) {
          if (mounted.current) setError(err instanceof Error ? err.message : String(err));
        }
      })();
    }
    return () => {
      mounted.current = false;
      // The Core owns the browser: hidden or remounted panels never kill it.
    };
  }, [initialSessionId, projectId, initialUrl]);

  const refreshImage = useCallback(async () => {
    if (!session || document.hidden || !visible) return;
    const headers: Record<string, string> = {};
    const token = getDesktopAuthToken();
    if (token) headers.Authorization = `Bearer ${token}`;
    const res = await fetch(`${browserRoute(session.id)}/frame`, {
      headers,
      cache: 'no-store',
    });
    if (!res.ok) throw new Error(`Screenshot unavailable (HTTP ${res.status})`);
    const next = URL.createObjectURL(await res.blob());
    if (!mounted.current) {
      URL.revokeObjectURL(next);
      return;
    }
    if (frameObjectURL.current) URL.revokeObjectURL(frameObjectURL.current);
    frameObjectURL.current = next;
    setFrame(next);
  }, [session, visible]);

  // CDP Page.startScreencast sends frames on page changes. The screenshot
  // endpoint is a fallback for older hosts or transient WebSocket failures.
  useEffect(() => {
    if (!session || !visible || document.hidden) return;
    let stopped = false;
    let ws: WebSocket | null = null;
    let reconnect: number | undefined;
    let attempts = 0;

    const connect = () => {
      if (stopped) return;
      ws = new WebSocket(getWebSocketEndpoint(`${browserRoute(session.id)}/stream`));
      ws.onopen = () => {
        attempts = 0;
        setLive(true);
        setError('');
      };
      ws.onmessage = (event: MessageEvent<string>) => {
        try {
          const payload = JSON.parse(event.data) as { type?: string; data?: string };
          if (payload.type !== 'frame' || !payload.data || stopped) return;
          if (frameObjectURL.current) {
            URL.revokeObjectURL(frameObjectURL.current);
            frameObjectURL.current = '';
          }
          lastFrameAt.current = Date.now();
          setFrame(`data:image/jpeg;base64,${payload.data}`);
        } catch {
          // An unexpected message must never crash the Workspace.
        }
      };
      ws.onclose = () => {
        setLive(false);
        if (stopped) return;
        const delay = Math.min(8000, 500 * 2 ** Math.min(attempts++, 4));
        reconnect = window.setTimeout(connect, delay);
      };
      ws.onerror = () => ws?.close();
    };
    connect();
    const fallback = window.setInterval(() => {
      if (ws?.readyState !== WebSocket.OPEN || Date.now() - lastFrameAt.current > 7000) {
        void refreshImage().catch(() => undefined);
      }
    }, 3500);
    return () => {
      stopped = true;
      setLive(false);
      window.clearInterval(fallback);
      if (reconnect) window.clearTimeout(reconnect);
      ws?.close();
    };
  }, [session?.id, visible, refreshImage]);

  useEffect(
    () => () => {
      if (frameObjectURL.current) URL.revokeObjectURL(frameObjectURL.current);
    },
    [],
  );

  // One authenticated Nexus session owns the input lease for 25 seconds.
  // Other desktop/mobile clients can watch, and gain control when released
  // or when its heartbeat expires.
  useEffect(() => {
    if (!session || !visible || !wantsControl) {
      setController(false);
      return;
    }
    let stopped = false;
    const acquire = async () => {
      try {
        const value = await browserAPI.control(session.id, 'acquire');
        if (!stopped) setController(value.controller);
      } catch {
        if (!stopped) setController(false);
      }
    };
    void acquire();
    const interval = window.setInterval(() => {
      if (document.hidden) return;
      void browserAPI
        .control(session.id, 'renew')
        .then((res) => {
          if (!stopped) setController(res.controller);
        })
        .catch(() => {
          if (!stopped) setController(false);
        });
    }, 11000);
    return () => {
      stopped = true;
      window.clearInterval(interval);
      setController(false);
      void browserAPI.control(session.id, 'release').catch(() => undefined);
    };
  }, [session?.id, visible, wantsControl]);

  const perform = useCallback(
    async (action: NexusBrowserAction) => {
      if (!session || !controller || busy) return;
      setBusy(true);
      try {
        const next = await browserAPI.action(session.id, action);
        setSession(next);
        if (action.type === 'navigate') setAddress(next.url);
        setError('');
        window.setTimeout(() => {
          void refreshImage().catch(() => undefined);
        }, 450);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(false);
      }
    },
    [session, busy, controller, refreshImage],
  );

  const clickAt = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (ignoreClick.current) {
      ignoreClick.current = false;
      return;
    }
    if (!session || !controller) return;
    const rect = event.currentTarget.getBoundingClientRect();
    if (!rect.width || !rect.height) return;
    const x = Math.max(
      0,
      Math.min(session.width - 1, ((event.clientX - rect.left) * session.width) / rect.width),
    );
    const y = Math.max(
      0,
      Math.min(session.height - 1, ((event.clientY - rect.top) * session.height) / rect.height),
    );
    void perform({ type: 'click', x, y });
  };

  const inspect = () => {
    if (!session) return;
    void browserAPI
      .snapshot(session.id)
      .then((value) => {
        setSnapshot(value.text);
        if (value.url) setAddress(value.url);
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  };

  return (
    <section
      ref={rootRef}
      className={styles.root}
      aria-label={t('browser.title', 'Navegador Nexus')}
    >
      <form
        className={styles.toolbar}
        onSubmit={(event) => {
          event.preventDefault();
          void perform({ type: 'navigate', url: address.trim() });
        }}
      >
        <Globe size={16} aria-hidden="true" />
        <button
          type="button"
          disabled={!controller || busy}
          onClick={() => void perform({ type: 'back' })}
          aria-label="Voltar"
        >
          <ArrowLeft size={15} />
        </button>
        <button
          type="button"
          disabled={!controller || busy}
          onClick={() => void perform({ type: 'forward' })}
          aria-label="Avançar"
        >
          <ArrowRight size={15} />
        </button>
        <input
          type="url"
          className={styles.address}
          aria-label="Endereço da página"
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          required
        />
        <button type="submit" disabled={!controller || busy} aria-label="Navegar">
          <ArrowRight size={15} />
        </button>
        <button
          type="button"
          disabled={!controller || busy}
          onClick={() => void perform({ type: 'reload' })}
          aria-label="Recarregar página"
        >
          <RefreshCw size={15} />
        </button>
        <button
          type="button"
          disabled={!session}
          onClick={inspect}
          aria-label="Inspecionar conteúdo"
        >
          <Camera size={15} />
        </button>
      </form>
      <div className={styles.controlBar} role="status">
        <span>{live ? 'Ao vivo' : 'Capturas / reconectando'}</span>
        <span>{controller ? 'Você controla' : 'Somente visualização'}</span>
        {session && (
          <button
            type="button"
            onClick={() => {
              if (controller) {
                setWantsControl(false);
                setController(false);
              } else {
                setWantsControl(true);
                void browserAPI
                  .control(session.id, 'acquire')
                  .then((result) => setController(result.controller))
                  .catch((err: unknown) =>
                    setError(err instanceof Error ? err.message : String(err)),
                  );
              }
            }}
          >
            {controller ? <Unlock size={13} /> : <Lock size={13} />}
            {controller ? 'Liberar controle' : 'Assumir controle'}
          </button>
        )}
      </div>
      <div className={styles.viewportScroll}>
        {frame ? (
          <button
            type="button"
            className={styles.viewportButton}
            onClick={clickAt}
            onWheel={(event) => {
              if (!session || !controller) return;
              void perform({
                type: 'scroll',
                x: session.width / 2,
                y: session.height / 2,
                delta_y: event.deltaY,
              });
            }}
            onTouchStart={(event) => {
              touchStart.current = event.touches[0]?.clientY ?? null;
            }}
            onTouchEnd={(event) => {
              if (touchStart.current === null || !session) return;
              const end = event.changedTouches[0]?.clientY;
              if (end !== undefined && Math.abs(end - touchStart.current) > 25) {
                ignoreClick.current = true;
                if (controller)
                  void perform({
                    type: 'scroll',
                    x: session.width / 2,
                    y: session.height / 2,
                    delta_y: (touchStart.current - end) * 2,
                  });
              }
              touchStart.current = null;
            }}
            aria-label="Página do navegador (clique para interagir)"
          >
            <img src={frame} alt="Visualização atual da página" draggable={false} />
          </button>
        ) : (
          <div className={styles.placeholder}>Iniciando Chromium / aguardando imagem…</div>
        )}
      </div>
      <form
        className={styles.inputBar}
        onSubmit={(event) => {
          event.preventDefault();
          if (text) {
            void perform({ type: 'type', text });
            setText('');
          }
        }}
      >
        <Keyboard size={16} aria-hidden="true" />
        <input
          aria-label="Texto para o elemento selecionado"
          placeholder="Digitar no elemento selecionado"
          value={text}
          onChange={(event) => setText(event.target.value)}
        />
        <button type="submit" disabled={!controller || !text || busy} aria-label="Enviar texto">
          <Send size={15} />
        </button>
        <button
          type="button"
          disabled={!controller || busy}
          onClick={() => void perform({ type: 'key', key: 'Enter' })}
        >
          Enter
        </button>
        <button
          type="button"
          disabled={!controller || busy}
          onClick={() => void perform({ type: 'key', key: 'Tab' })}
        >
          Tab
        </button>
        <button
          type="button"
          disabled={!controller || busy}
          onClick={() => void perform({ type: 'key', key: 'Backspace' })}
        >
          ⌫
        </button>
      </form>
      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
      {snapshot && (
        <details className={styles.snapshot}>
          <summary>Conteúdo da página</summary>
          <pre>{snapshot}</pre>
        </details>
      )}
    </section>
  );
};
