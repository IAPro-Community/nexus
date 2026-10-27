import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Camera,
  Globe,
  Keyboard,
  Lock,
  Monitor,
  RefreshCw,
  Send,
  Smartphone,
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
  owner_runtime_id?: string;
};

export type NexusBrowserAction = {
  type:
    | 'navigate'
    | 'back'
    | 'forward'
    | 'reload'
    | 'click'
    | 'scroll'
    | 'type'
    | 'key'
    | 'click_selector'
    | 'fill_selector'
    | 'wait_selector'
    | 'viewport';
  url?: string;
  text?: string;
  key?: string;
  x?: number;
  y?: number;
  delta_y?: number;
  selector?: string;
  timeout_ms?: number;
  width?: number;
  height?: number;
};

export type BrowserHandoff = {
  status: '' | 'requested' | 'human_active' | 'completed' | 'cancelled';
  reason?: string;
  requested_by?: string;
};
export type AgentGrant = { id: string; label: string; expires_at: string };
export type NexusBrowserElement = {
  selector: string;
  tag: string;
  role: string;
  label: string;
  text: string;
};

export type NexusBrowserSnapshot = {
  url: string;
  title: string;
  untrusted: true;
  text: string;
  elements: NexusBrowserElement[];
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
  list: (projectId: string) =>
    request<NexusBrowserInfo[]>(`${endpoint}?project_id=${encodeURIComponent(projectId)}`),
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
  handoffInfo: (id: string) => request<BrowserHandoff>(`${browserRoute(id)}/handoff`),
  handoff: (id: string, mode: 'begin' | 'complete' | 'cancel') =>
    request<BrowserHandoff>(`${browserRoute(id)}/handoff`, {
      method: 'POST',
      body: JSON.stringify({ mode }),
    }),
  grant: (id: string) =>
    request<{ token: string; mcp_url: string; grant: AgentGrant }>(`${browserRoute(id)}/grant`, {
      method: 'POST',
      body: JSON.stringify({ label: 'Nexus Workspace agent' }),
    }),
  grants: (id: string) => request<AgentGrant[]>(`${browserRoute(id)}/grants`),
  revokeGrant: (id: string, grantId: string) =>
    request<{ revoked: boolean }>(`${browserRoute(id)}/grant`, {
      method: 'DELETE',
      body: JSON.stringify({ id: grantId }),
    }),
  snapshot: (id: string) => request<NexusBrowserSnapshot>(`${browserRoute(id)}/snapshot`),
};

export const BrowserSurface: React.FC<{
  projectId: string;
  initialSessionId?: string;
  initialUrl?: string;
  onSessionReady?: (id: string) => void;
}> = ({ projectId, initialSessionId, initialUrl = 'http://localhost:3000', onSessionReady }) => {
  const { t } = useTranslation();
  const [session, setSession] = useState<NexusBrowserInfo | null>(null);
  const [availableBrowsers, setAvailableBrowsers] = useState<NexusBrowserInfo[]>([]);
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
  const [elements, setElements] = useState<NexusBrowserElement[]>([]);
  const [zoom, setZoom] = useState(100);
  const [handoff, setHandoff] = useState<BrowserHandoff>({ status: '' });
  const [grants, setGrants] = useState<AgentGrant[]>([]);
  const [privateInput, setPrivateInput] = useState(false);
  const [connectionInfo, setConnectionInfo] = useState('');
  const [grantFallback, setGrantFallback] = useState('');
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
          let createdNew = false;
          if (initialSessionId) {
            try {
              next = await browserAPI.get(initialSessionId);
              if (next.project_id !== projectId) throw new Error('stale project session');
            } catch {
              next = await browserAPI.create(projectId, initialUrl);
              createdNew = true;
            }
          } else {
            // Prefer an existing agent-owned browser so opening Browser in
            // the Workspace previews the very page the CLI is using.
            const existing = await browserAPI.list(projectId);
            const agentOwned = existing.find((candidate) => Boolean(candidate.owner_runtime_id));
            if (agentOwned) {
              next = agentOwned;
            } else {
              next = await browserAPI.create(projectId, initialUrl);
              createdNew = true;
            }
          }
          if (!mounted.current) {
            if (createdNew) {
              void request<{ status: string }>(browserRoute(next.id), {
                method: 'DELETE',
              }).catch(() => undefined);
            }
            return;
          }
          if (next.owner_runtime_id) setWantsControl(false);
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

  useEffect(() => {
    if (!session) return;
    let disposed = false;
    const poll = async () => {
      try {
        const [status, activeGrants, browserInfo, browsers] = await Promise.all([
          browserAPI.handoffInfo(session.id),
          browserAPI.grants(session.id),
          browserAPI.get(session.id),
          browserAPI.list(projectId),
        ]);
        if (!disposed) {
          setHandoff(status);
          setGrants(activeGrants);
          setAvailableBrowsers(browsers);
          // Keep the visible address in sync with agent-driven navigation
          // without overwriting a human editing the location bar.
          if (
            browserInfo.url &&
            document.activeElement?.getAttribute('aria-label') !== 'Endereço da página'
          ) {
            setAddress(browserInfo.url);
          }
          if (status.status === 'human_active') setPrivateInput(true);
          if (status.status === 'completed' || status.status === 'cancelled')
            setPrivateInput(false);
        }
      } catch {
        // The browser may have been closed or Core temporarily disconnected.
      }
    };
    void poll();
    const interval = window.setInterval(() => void poll(), 2500);
    return () => {
      disposed = true;
      window.clearInterval(interval);
    };
  }, [session?.id]);

  const createAgentConnection = async () => {
    if (!session) return;
    try {
      const grant = await browserAPI.grant(session.id);
      const text = `# Execute no host que roda o Nexus Core (ajuste a porta se necessário)
export NEXUS_BROWSER_TOKEN='${grant.token}'
codex mcp add nexus-browser --url '${grant.mcp_url}' --bearer-token-env-var NEXUS_BROWSER_TOKEN`;
      try {
        await navigator.clipboard.writeText(text);
        setConnectionInfo(
          'Comandos MCP copiados. O token concede acesso apenas a esta sessão e expira em 1 hora.',
        );
        setGrantFallback('');
      } catch {
        setGrantFallback(text);
        setConnectionInfo('A cópia automática falhou. Copie os comandos apresentados abaixo.');
      }
      setGrants((current) => [...current, grant.grant]);
      setWantsControl(false);
      setController(false);
      void browserAPI.control(session.id, 'release').catch(() => undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const respondToHuman = async (mode: 'begin' | 'complete' | 'cancel') => {
    if (!session) return;
    try {
      const next = await browserAPI.handoff(session.id, mode);
      setHandoff(next);
      const takingOver = mode === 'begin';
      setWantsControl(takingOver);
      setController(takingOver);
      setPrivateInput(takingOver);
      setText('');
      setSnapshot('');
      setElements([]);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

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

  const selectBrowser = (id: string) => {
    if (!id || id === session?.id) return;
    void browserAPI.get(id).then((next) => {
      if (next.project_id !== projectId) return;
      setWantsControl(false);
      setController(false);
      setSession(next);
      setAddress(next.url);
      setSnapshot('');
      setElements([]);
      setError('');
      onReady.current?.(next.id);
    }).catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  };

  const inspect = () => {
    if (!session) return;
    void browserAPI
      .snapshot(session.id)
      .then((value) => {
        setSnapshot(value.text);
        setElements(value.elements || []);
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
        {availableBrowsers.length > 1 && (
          <select
            className={styles.browserSelect}
            aria-label="Alternar navegador do Workspace"
            value={session?.id || ''}
            onChange={(event) => selectBrowser(event.target.value)}
          >
            {availableBrowsers.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.owner_runtime_id
                  ? `Agente · ${candidate.owner_runtime_id.slice(0, 18)}`
                  : `Manual · ${candidate.id.slice(0, 8)}`}
              </option>
            ))}
          </select>
        )}
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
          disabled={!controller || busy}
          onClick={() =>
            void perform(
              session?.width === 390
                ? { type: 'viewport', width: 1280, height: 800 }
                : { type: 'viewport', width: 390, height: 844 },
            )
          }
          aria-label={session?.width === 390 ? 'Viewport desktop' : 'Viewport celular'}
          title={session?.width === 390 ? 'Viewport desktop' : 'Viewport celular'}
        >
          {session?.width === 390 ? <Monitor size={15} /> : <Smartphone size={15} />}
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
        {session?.owner_runtime_id && <span>Navegador do agente · {session.owner_runtime_id}</span>}
        <span>
          {handoff.status === 'human_active'
            ? 'Intervenção humana · privada'
            : controller
              ? 'Você controla'
              : 'Somente visualização'}
        </span>
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
                  .catch(async () => {
                    // If an agent owns the page, use an atomic Core handoff
                    // rather than competing with the agent's input lease.
                    try {
                      const state = await browserAPI.handoff(session.id, 'begin');
                      setHandoff(state);
                      setPrivateInput(true);
                      setController(true);
                    } catch (err) {
                      setController(false);
                      setError(err instanceof Error ? err.message : String(err));
                    }
                  });
              }
            }}
          >
            {controller ? <Unlock size={13} /> : <Lock size={13} />}
            {controller ? 'Liberar controle' : 'Assumir controle'}
          </button>
        )}
      </div>
      {(handoff.status === 'requested' || handoff.status === 'human_active') && (
        <div className={styles.handoffPanel} role="status">
          <strong>
            {handoff.status === 'requested'
              ? 'Agente solicita sua intervenção'
              : 'Você está controlando o navegador'}
          </strong>
          <span>
            {handoff.reason || 'Conclua autenticação ou parâmetros diretamente na página.'}
          </span>
          {handoff.status === 'requested' ? (
            <button type="button" onClick={() => void respondToHuman('begin')}>
              Assumir para autenticar
            </button>
          ) : (
            <div className={styles.handoffButtons}>
              <button type="button" onClick={() => void respondToHuman('complete')}>
                Concluído · devolver ao agente
              </button>
              <button type="button" onClick={() => void respondToHuman('cancel')}>
                Cancelar intervenção
              </button>
            </div>
          )}
          <small>
            O agente permanece pausado e não recebe senhas, capturas ou conteúdo da página durante a
            intervenção.
          </small>
        </div>
      )}
      {session && (
        <details className={styles.agentConnections}>
          <summary>Conexões avançadas com agentes externos</summary>
          <p>
            Agentes iniciados pelo Nexus recebem o Browser automaticamente. Use esta opção apenas
            para uma conexão externa autorizada.
          </p>
          <button type="button" onClick={() => void createAgentConnection()}>
            Gerar acesso temporário e copiar configuração
          </button>
          {connectionInfo && <p role="status">{connectionInfo}</p>}
          {grantFallback && (
            <textarea readOnly value={grantFallback} aria-label="Configuração MCP de uso único" />
          )}
          {grants.map((grant) => (
            <div className={styles.grantRow} key={grant.id}>
              <span>
                {grant.label} · até {new Date(grant.expires_at).toLocaleTimeString()}
              </span>
              <button
                type="button"
                onClick={() => {
                  void browserAPI
                    .revokeGrant(session.id, grant.id)
                    .then(() => {
                      setGrants((items) => items.filter((item) => item.id !== grant.id));
                      setGrantFallback('');
                    })
                    .catch((err: unknown) =>
                      setError(err instanceof Error ? err.message : String(err)),
                    );
                }}
              >
                Revogar
              </button>
            </div>
          ))}
        </details>
      )}
      <div className={styles.zoomControls} role="group" aria-label="Zoom do navegador">
        <button
          type="button"
          onClick={() => setZoom((value) => Math.max(75, value - 25))}
          disabled={zoom <= 75}
          aria-label="Diminuir zoom"
        >
          −
        </button>
        <span>{zoom}%</span>
        <button
          type="button"
          onClick={() => setZoom((value) => Math.min(250, value + 25))}
          disabled={zoom >= 250}
          aria-label="Aumentar zoom"
        >
          +
        </button>
      </div>
      <div className={styles.viewportScroll}>
        {frame ? (
          <button
            type="button"
            className={styles.viewportButton}
            style={{ width: `${zoom}%` }}
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
          type={privateInput ? 'password' : 'text'}
          autoComplete="off"
          aria-label={
            privateInput ? 'Texto sensível para login' : 'Texto para o elemento selecionado'
          }
          placeholder="Digitar no elemento selecionado"
          value={text}
          onChange={(event) => setText(event.target.value)}
        />
        <button
          type="button"
          onClick={() => setPrivateInput((value) => !value)}
          title="Ocultar caracteres digitados neste campo"
        >
          {' '}
          {privateInput ? 'Privado' : 'Ocultar'}{' '}
        </button>
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
      {(snapshot || elements.length > 0) && (
        <details className={styles.snapshot}>
          <summary>Inspeção da página · {elements.length} elementos</summary>
          <div className={styles.inspectionActions}>
            {elements.map((item, index) => (
              <button
                key={`${item.selector}:${index}`}
                type="button"
                disabled={!controller || busy}
                title={item.selector}
                onClick={() => void perform({ type: 'click_selector', selector: item.selector })}
              >
                {item.label || item.text || item.tag}
              </button>
            ))}
          </div>
          <pre>{snapshot}</pre>
          <small>Conteúdo de sites externos não é confiável.</small>
        </details>
      )}
    </section>
  );
};
