import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ArrowRight, Camera, Globe, Keyboard, MousePointer2, RefreshCw, Send } from 'lucide-react';
import { getDesktopAuthToken, request } from '../../api';
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
  type: 'navigate' | 'click' | 'scroll' | 'type' | 'key';
  url?: string;
  text?: string;
  key?: string;
  x?: number;
  y?: number;
  delta_y?: number;
};

const endpoint = '/api/v1/browser/sessions';

export const browserAPI = {
  create: (projectId: string, url: string) =>
    request<NexusBrowserInfo>(endpoint, {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, url }),
    }),
  get: (id: string) => request<NexusBrowserInfo>(`${endpoint}/${encodeURIComponent(id)}`),
  action: (id: string, action: NexusBrowserAction) =>
    request<NexusBrowserInfo>(`${endpoint}/${encodeURIComponent(id)}/action`, {
      method: 'POST',
      body: JSON.stringify(action),
    }),
  snapshot: (id: string) =>
    request<{ url: string; title: string; text: string }>(
      `${endpoint}/${encodeURIComponent(id)}/snapshot`,
    ),
};

export const BrowserSurface: React.FC<{
  projectId: string;
  initialSessionId?: string;
  initialUrl?: string;
  onSessionReady?: (id: string) => void;
}> = ({ projectId, initialSessionId, initialUrl = 'http://localhost:3000', onSessionReady }) => {
  const { t } = useTranslation();
  const [session, setSession] = useState<NexusBrowserInfo | null>(null);
  const [url, setURL] = useState(initialUrl);
  const [text, setText] = useState('');
  const [frame, setFrame] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [snapshot, setSnapshot] = useState('');
  const frameURL = useRef('');
  const readyRef = useRef(onSessionReady);
  const started = useRef(false);
  const alive = useRef(false);
  const touchY = useRef<number | null>(null);

  useEffect(() => {
    readyRef.current = onSessionReady;
  }, [onSessionReady]);

  // StrictMode and workspace visibility changes must not create duplicate sessions.
  useEffect(() => {
    alive.current = true;
    if (started.current)
      return () => {
        alive.current = false;
      };
    started.current = true;
    async function start() {
      try {
        let next: NexusBrowserInfo;
        if (initialSessionId) {
          try {
            next = await browserAPI.get(initialSessionId);
            if (next.project_id !== projectId) throw new Error('wrong project');
          } catch {
            next = await browserAPI.create(projectId, initialUrl);
          }
        } else {
          next = await browserAPI.create(projectId, initialUrl);
        }
        if (!alive.current) return;
        setSession(next);
        setURL(next.url);
        readyRef.current?.(next.id);
      } catch (e) {
        if (alive.current) setError(e instanceof Error ? e.message : String(e));
      }
    }
    void start();
    return () => {
      alive.current = false;
      // Backend session deliberately lives beyond tab switches / minimized windows.
    };
  }, [initialSessionId, initialUrl, projectId]);

  const refresh = useCallback(async () => {
    if (!session || document.hidden) return;
    try {
      // Binary frame requests reuse the same authenticated Core origin as JSON APIs.
      const headers: Record<string, string> = {};
      const token = getDesktopAuthToken();
      if (token) headers.Authorization = `Bearer ${token}`;
      const response = await fetch(`${endpoint}/${encodeURIComponent(session.id)}/frame`, {
        headers,
        cache: 'no-store',
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const blob = await response.blob();
      const next = URL.createObjectURL(blob);
      const previous = frameURL.current;
      frameURL.current = next;
      setFrame(next);
      if (previous) URL.revokeObjectURL(previous);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [session]);

  useEffect(() => {
    if (!session) return;
    void refresh();
    const timer = window.setInterval(() => {
      void refresh();
    }, 1800);
    return () => {
      window.clearInterval(timer);
    };
  }, [refresh, session]);

  useEffect(() => {
    return () => {
      if (frameURL.current) URL.revokeObjectURL(frameURL.current);
    };
  }, []);

  const perform = useCallback(
    async (action: NexusBrowserAction) => {
      if (!session || busy) return;
      setBusy(true);
      try {
        const next = await browserAPI.action(session.id, action);
        setSession(next);
        setURL(next.url);
        setError('');
        window.setTimeout(() => {
          void refresh();
        }, 300);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setBusy(false);
      }
    },
    [session, busy, refresh],
  );

  const clickAt = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (!session) return;
    const rect = event.currentTarget.getBoundingClientRect();
    if (!rect.width || !rect.height) return;
    // CSS preserves the viewport aspect ratio; CDP coordinates are CSS pixels.
    const x = Math.max(
      0,
      Math.min(session.width - 1, ((event.clientX - rect.left) / rect.width) * session.width),
    );
    const y = Math.max(
      0,
      Math.min(session.height - 1, ((event.clientY - rect.top) / rect.height) * session.height),
    );
    void perform({ type: 'click', x, y });
  };

  return (
    <section className={styles.root} aria-label={t('browser.title', 'Navegador Nexus')}>
      <form
        className={styles.toolbar}
        onSubmit={(event) => {
          event.preventDefault();
          void perform({ type: 'navigate', url: url.trim() });
        }}
      >
        <Globe size={16} aria-hidden="true" />
        <input
          type="url"
          className={styles.address}
          aria-label={t('browser.address', 'Endereço da página')}
          value={url}
          onChange={(event) => setURL(event.target.value)}
          required
        />
        <button type="submit" disabled={!session || busy} aria-label={t('browser.go', 'Navegar')}>
          <ArrowRight size={15} />
        </button>
        <button
          type="button"
          onClick={() => void refresh()}
          disabled={!session}
          aria-label={t('browser.refresh', 'Atualizar imagem')}
        >
          <RefreshCw size={15} />
        </button>
        <button
          type="button"
          disabled={!session}
          onClick={() => {
            if (!session) return;
            void browserAPI
              .snapshot(session.id)
              .then((value) => {
                setSnapshot(value.text);
                if (value.url) {
                  setURL(value.url);
                  setSession((current) => (current ? { ...current, url: value.url } : current));
                }
              })
              .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)));
          }}
          aria-label={t('browser.inspect', 'Inspecionar página')}
        >
          <Camera size={15} />
        </button>
      </form>
      <div className={styles.viewportScroll}>
        {frame ? (
          <button
            type="button"
            className={styles.viewportButton}
            onClick={clickAt}
            onWheel={(event) => {
              if (!session) return;
              // Centered wheel support for trackpads and mouse scroll.
              void perform({
                type: 'scroll',
                x: session.width / 2,
                y: session.height / 2,
                delta_y: event.deltaY,
              });
            }}
            onTouchStart={(event) => {
              touchY.current = event.touches[0]?.clientY ?? null;
            }}
            onTouchEnd={(event) => {
              if (touchY.current == null || !session) return;
              const end = event.changedTouches[0]?.clientY;
              if (end != null && Math.abs(end - touchY.current) > 30) {
                void perform({
                  type: 'scroll',
                  x: session.width / 2,
                  y: session.height / 2,
                  delta_y: (touchY.current - end) * 2,
                });
              }
              touchY.current = null;
            }}
            aria-label={t('browser.viewport', 'Página interativa do navegador')}
          >
            <img
              src={frame}
              alt={t('browser.viewport', 'Página interativa do navegador')}
              draggable={false}
            />
          </button>
        ) : (
          <div className={styles.placeholder}>
            <MousePointer2 size={22} />
            {session
              ? t('browser.loading', 'Carregando página…')
              : t('browser.starting', 'Iniciando Chromium…')}
          </div>
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
          aria-label={t('browser.type', 'Digitar no elemento selecionado')}
          placeholder={t('browser.type', 'Digitar no elemento selecionado')}
          value={text}
          onChange={(event) => setText(event.target.value)}
        />
        <button
          type="submit"
          disabled={!session || !text || busy}
          aria-label={t('browser.sendText', 'Enviar texto')}
        >
          <Send size={15} />
        </button>
        <button
          type="button"
          disabled={!session || busy}
          onClick={() => void perform({ type: 'key', key: 'Enter' })}
        >
          Enter
        </button>
        <button
          type="button"
          disabled={!session || busy}
          onClick={() => void perform({ type: 'key', key: 'Tab' })}
        >
          Tab
        </button>
      </form>
      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
      {snapshot && (
        <details className={styles.snapshot}>
          <summary>{t('browser.inspection', 'Texto da página')}</summary>
          <pre>{snapshot}</pre>
        </details>
      )}
    </section>
  );
};
