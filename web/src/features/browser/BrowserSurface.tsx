import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Bot,
  Camera,
  Copy,
  Globe,
  Keyboard,
  Lock,
  Monitor,
  Plus,
  RefreshCw,
  Send,
  Smartphone,
  Unlock,
  X,
} from 'lucide-react';
import { getDesktopAuthToken, getWebSocketEndpoint, request } from '../../api';
import { useTranslation } from 'react-i18next';
import styles from './BrowserSurface.module.scss';

export function normalizeBrowserURL(input: string): string {
  const trimmed = input.trim();
  if (!trimmed) return trimmed;
  if (/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(trimmed)) {
    return trimmed;
  }
  if (trimmed.startsWith('//')) {
    return `https:${trimmed}`;
  }
  const host = trimmed.split(/[/?#]/)[0].toLowerCase();
  if (
    host === 'localhost' ||
    host.startsWith('localhost:') ||
    host === '127.0.0.1' ||
    host.startsWith('127.0.0.1:') ||
    host === '0.0.0.0' ||
    host.startsWith('0.0.0.0:') ||
    host === '[::1]' ||
    host.startsWith('[::1]:')
  ) {
    return `http://${trimmed}`;
  }
  return `https://${trimmed}`;
}

export type NexusBrowserInfo = {
  id: string;
  project_id: string;
  url: string;
  title?: string;
  width: number;
  height: number;
  owner_runtime_id?: string;
  active_actor?: string;
  last_active_at?: string;
};

export function formatTabTitle(candidate: NexusBrowserInfo): string {
  if (candidate.title && candidate.title.trim()) {
    return candidate.title.trim();
  }
  if (candidate.url) {
    try {
      const parsed = new URL(candidate.url);
      const host = parsed.hostname;
      if (host) {
        const path = parsed.pathname !== '/' ? parsed.pathname : '';
        return `${host}${path}`.slice(0, 24);
      }
    } catch {
      return candidate.url.slice(0, 24);
    }
  }
  if (candidate.owner_runtime_id) {
    return `Agente · ${candidate.owner_runtime_id.slice(0, 8)}`;
  }
  return `Aba · ${candidate.id.slice(0, 6)}`;
}

export type NexusBrowserAction = {
  type:
    | 'navigate'
    | 'back'
    | 'forward'
    | 'reload'
    | 'click'
    | 'mouse_down'
    | 'mouse_up'
    | 'mouse_move'
    | 'scroll'
    | 'type'
    | 'key'
    | 'select_all'
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
  click_count?: number;
  button?: 'left' | 'right' | 'middle';
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
  selection?: string;
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
  close: (id: string) =>
    request<{ status: string }>(browserRoute(id), {
      method: 'DELETE',
    }),
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
  const [dragging, setDragging] = useState(false);
  const [selectionText, setSelectionText] = useState('');
  const [updatedTabs, setUpdatedTabs] = useState<Set<string>>(new Set());
  const previousBrowserState = useRef<
    Map<string, { lastActiveAt: string; url: string; activeActor: string }>
  >(new Map());
  const rootRef = useRef<HTMLElement>(null);
  const frameObjectURL = useRef('');
  const started = useRef(false);
  const mounted = useRef(false);
  const onReady = useRef(onSessionReady);
  const touchStart = useRef<number | null>(null);
  const ignoreClick = useRef(false);
  const lastFrameAt = useRef(0);
  const queueRef = useRef<NexusBrowserAction[]>([]);
  const processingRef = useRef(false);
  const isDragging = useRef(false);
  const dragStart = useRef<{ x: number; y: number; clientX: number; clientY: number } | null>(null);
  const lastMoveAt = useRef(0);

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

  const selectBrowser = useCallback(
    (id: string) => {
      if (!id) return;
      setUpdatedTabs((prevSet) => {
        if (!prevSet.has(id)) return prevSet;
        const next = new Set(prevSet);
        next.delete(id);
        return next;
      });
      if (id === session?.id) return;
      void browserAPI
        .get(id)
        .then((next) => {
          if (next.project_id !== projectId) return;
          setWantsControl(false);
          setController(false);
          setSession(next);
          setAddress(next.url);
          setSnapshot('');
          setElements([]);
          setError('');
          onReady.current?.(next.id);
        })
        .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
    },
    [projectId, session?.id],
  );

  const closeBrowser = useCallback(
    async (id: string) => {
      try {
        await browserAPI.close(id);
        setAvailableBrowsers((prev) => prev.filter((b) => b.id !== id));
        setUpdatedTabs((prevSet) => {
          const next = new Set(prevSet);
          next.delete(id);
          return next;
        });
        previousBrowserState.current.delete(id);
        if (session?.id === id) {
          const remaining = availableBrowsers.filter((b) => b.id !== id);
          if (remaining.length > 0) {
            selectBrowser(remaining[0].id);
          }
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    },
    [availableBrowsers, selectBrowser, session?.id],
  );

  const createNewBrowser = useCallback(async () => {
    if (availableBrowsers.length >= 4) return;
    try {
      const next = await browserAPI.create(projectId, 'https://example.org');
      setAvailableBrowsers((prev) => [...prev, next]);
      selectBrowser(next.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [availableBrowsers.length, projectId, selectBrowser]);

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

          let agentTabToFocus: string | null = null;
          for (const candidate of browsers) {
            const prev = previousBrowserState.current.get(candidate.id);
            if (prev) {
              const urlChanged = Boolean(candidate.url && candidate.url !== prev.url);
              const activityChanged = Boolean(
                candidate.last_active_at && candidate.last_active_at !== prev.lastActiveAt,
              );
              const actorChanged = Boolean(
                candidate.active_actor && candidate.active_actor !== prev.activeActor,
              );
              const hasChanged = urlChanged || activityChanged || actorChanged;

              if (hasChanged && candidate.id !== session.id) {
                setUpdatedTabs((prevSet) => {
                  const next = new Set(prevSet);
                  next.add(candidate.id);
                  return next;
                });

                const isAgentActing = Boolean(
                  candidate.active_actor?.startsWith('agent:') ||
                    (candidate.owner_runtime_id && (urlChanged || activityChanged)),
                );
                if (isAgentActing && !agentTabToFocus) {
                  agentTabToFocus = candidate.id;
                }
              }
            }
            previousBrowserState.current.set(candidate.id, {
              lastActiveAt: candidate.last_active_at || '',
              url: candidate.url || '',
              activeActor: candidate.active_actor || '',
            });
          }

          if (agentTabToFocus && session.id !== agentTabToFocus) {
            const activeEl = document.activeElement;
            const userIsTyping =
              activeEl &&
              (activeEl.tagName === 'INPUT' || activeEl.tagName === 'TEXTAREA') &&
              activeEl.getAttribute('aria-label') === 'Endereço da página';
            if (!userIsTyping) {
              selectBrowser(agentTabToFocus);
            }
          }

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
  }, [session?.id, session, projectId, selectBrowser]);

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
    if (!session || !wantsControl) {
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
  }, [session?.id, wantsControl]);

  const processQueue = useCallback(async () => {
    if (processingRef.current || !session) return;
    processingRef.current = true;
    setBusy(true);
    try {
      while (queueRef.current.length > 0) {
        const action = queueRef.current.shift();
        if (!action) break;
        if (!controller) {
          try {
            const res = await browserAPI.control(session.id, 'acquire');
            if (res.controller) {
              setController(true);
              setWantsControl(true);
            } else {
              try {
                const state = await browserAPI.handoff(session.id, 'begin');
                setHandoff(state);
                setPrivateInput(true);
                setController(true);
                setWantsControl(true);
              } catch {
                setError('Não foi possível assumir o controle do navegador.');
                break;
              }
            }
          } catch {
            try {
              const state = await browserAPI.handoff(session.id, 'begin');
              setHandoff(state);
              setPrivateInput(true);
              setController(true);
              setWantsControl(true);
            } catch (acquireErr) {
              setError(acquireErr instanceof Error ? acquireErr.message : String(acquireErr));
              break;
            }
          }
        }
        try {
          const next = await browserAPI.action(session.id, action);
          setSession(next);
          if (action.type === 'navigate') setAddress(next.url);
          setError('');
        } catch (err) {
          setError(err instanceof Error ? err.message : String(err));
        }
      }
    } finally {
      processingRef.current = false;
      setBusy(false);
      window.setTimeout(() => void refreshImage().catch(() => undefined), 100);
      window.setTimeout(() => void refreshImage().catch(() => undefined), 350);
    }
  }, [session, controller, refreshImage]);

  const perform = useCallback(
    (action: NexusBrowserAction) => {
      queueRef.current.push(action);
      void processQueue();
    },
    [processQueue],
  );

  const toViewportCoords = (clientX: number, clientY: number, target: HTMLElement | null) => {
    if (!session || !target) return null;
    const img = (target.querySelector('img') as HTMLImageElement | null) || (target as HTMLElement);
    const rect = img.getBoundingClientRect();
    if (!rect.width || !rect.height) return null;
    const rawX = ((clientX - rect.left) / rect.width) * session.width;
    const rawY = ((clientY - rect.top) / rect.height) * session.height;
    const x = Math.max(0, Math.min(session.width - 1, Math.round(rawX)));
    const y = Math.max(0, Math.min(session.height - 1, Math.round(rawY)));
    return { x, y };
  };

  const handleMouseDown = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (event.button !== 0 && event.button !== 2) return;
    if (!session) return;
    const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
    if (!coords) return;
    dragStart.current = { ...coords, clientX: event.clientX, clientY: event.clientY };
    isDragging.current = false;
  };

  const handleMouseMove = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (!session || !dragStart.current) return;
    const distance = Math.hypot(
      event.clientX - dragStart.current.clientX,
      event.clientY - dragStart.current.clientY,
    );
    if (distance > 5) {
      const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
      if (!coords) return;
      if (!isDragging.current) {
        isDragging.current = true;
        setDragging(true);
        perform({
          type: 'mouse_down',
          x: dragStart.current.x,
          y: dragStart.current.y,
          button: 'left',
        });
      }
      const now = Date.now();
      if (now - lastMoveAt.current > 50) {
        lastMoveAt.current = now;
        perform({ type: 'mouse_move', x: coords.x, y: coords.y });
      }
    }
  };

  const handleMouseUp = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (!session) return;
    const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
    if (isDragging.current) {
      isDragging.current = false;
      setDragging(false);
      dragStart.current = null;
      if (coords) {
        perform({
          type: 'mouse_up',
          x: coords.x,
          y: coords.y,
          button: 'left',
        });
      }
      window.setTimeout(() => {
        void browserAPI
          .snapshot(session.id)
          .then((snap) => {
            if (snap.selection) setSelectionText(snap.selection);
          })
          .catch(() => undefined);
      }, 250);
      return;
    }
    dragStart.current = null;
  };

  const handleMouseLeave = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (isDragging.current && session) {
      isDragging.current = false;
      setDragging(false);
      const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
      if (coords) {
        perform({ type: 'mouse_up', x: coords.x, y: coords.y, button: 'left' });
      }
    }
    dragStart.current = null;
  };

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (ignoreClick.current) {
      ignoreClick.current = false;
      return;
    }
    if (isDragging.current) {
      return;
    }
    if (!session) return;
    const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
    if (!coords) return;
    const clickCount = event.detail || 1;
    perform({
      type: 'click',
      x: coords.x,
      y: coords.y,
      click_count: clickCount,
      button: event.button === 2 ? 'right' : 'left',
    });
    if (clickCount >= 2) {
      window.setTimeout(() => {
        void browserAPI
          .snapshot(session.id)
          .then((snap) => {
            if (snap.selection) setSelectionText(snap.selection);
          })
          .catch(() => undefined);
      }, 250);
    }
  };

  const handleContextMenu = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    if (!session) return;
    const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
    if (coords) {
      perform({ type: 'click', x: coords.x, y: coords.y, click_count: 1, button: 'right' });
    }
  };

  const copySelection = async () => {
    if (!session) return;
    try {
      const snap = await browserAPI.snapshot(session.id);
      if (snap.selection) {
        await navigator.clipboard.writeText(snap.selection);
        setSelectionText(snap.selection);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLElement>) => {
    if (
      document.activeElement &&
      document.activeElement !== event.currentTarget &&
      ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement.tagName)
    ) {
      return;
    }
    if (!session) return;

    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'c') {
      void copySelection();
      return;
    }

    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'a') {
      event.preventDefault();
      perform({ type: 'select_all' });
      return;
    }

    if (
      [
        'ArrowUp',
        'ArrowDown',
        'ArrowLeft',
        'ArrowRight',
        'Space',
        ' ',
        'Tab',
        'Backspace',
        'Enter',
      ].includes(event.key)
    ) {
      event.preventDefault();
    }

    if (
      event.key === 'Backspace' ||
      event.key === 'Delete' ||
      event.key === 'Enter' ||
      event.key === 'Tab' ||
      event.key === 'Escape' ||
      event.key.startsWith('Arrow')
    ) {
      perform({ type: 'key', key: event.key });
    } else if (event.key === ' ') {
      perform({ type: 'key', key: ' ' });
    } else if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
      perform({ type: 'key', key: event.key });
    }
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
      {availableBrowsers.length > 1 && (
        <div
          className={styles.tabBar}
          role="tablist"
          aria-label={t('browser.tabs', 'Abas do navegador')}
          onKeyDown={(event) => {
            if (
              event.key !== 'ArrowRight' &&
              event.key !== 'ArrowLeft' &&
              event.key !== 'Home' &&
              event.key !== 'End'
            )
              return;
            const tabs = Array.from(
              event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="tab"]'),
            );
            const current = tabs.indexOf(document.activeElement as HTMLButtonElement);
            if (current === -1) return;
            event.preventDefault();
            const next =
              event.key === 'Home'
                ? 0
                : event.key === 'End'
                  ? tabs.length - 1
                  : (current + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length;
            tabs[next]?.focus();
          }}
        >
          <div className={styles.tabList}>
            {availableBrowsers.map((candidate) => {
              const isActive = candidate.id === session?.id;
              const isAgent = Boolean(candidate.owner_runtime_id);
              const isOperating = Boolean(
                candidate.active_actor && candidate.active_actor.startsWith('agent:'),
              );
              const isUpdated = updatedTabs.has(candidate.id) && !isActive;
              const title = formatTabTitle(candidate);

              return (
                <div
                  key={candidate.id}
                  className={styles.tabItem}
                  data-active={isActive ? 'true' : 'false'}
                  data-operating={isOperating ? 'true' : 'false'}
                  data-updated={isUpdated ? 'true' : 'false'}
                >
                  <button
                    type="button"
                    role="tab"
                    id={`nx-browser-tab-${candidate.id}`}
                    aria-selected={isActive}
                    className={styles.tabButton}
                    onClick={() => selectBrowser(candidate.id)}
                    title={`${title} (${candidate.url})`}
                  >
                    <span className={styles.tabIcon} aria-hidden="true">
                      {isAgent ? <Bot size={13} /> : <Globe size={13} />}
                    </span>
                    <span className={styles.tabLabel}>{title}</span>
                    {isOperating && (
                      <span
                        className={styles.operatingBadge}
                        title={t('browser.agentOperating', 'Agente operando nesta aba')}
                      >
                        <span className={styles.pulseDot} aria-hidden="true" />
                        <span className={styles.badgeText}>{t('browser.operating', 'Operando')}</span>
                      </span>
                    )}
                    {isUpdated && !isOperating && (
                      <span
                        className={styles.updateDot}
                        title={t('browser.tabUpdated', 'Aba atualizada')}
                        aria-label={t('browser.tabUpdated', 'Aba atualizada')}
                      />
                    )}
                  </button>
                  <button
                    type="button"
                    className={styles.tabCloseButton}
                    aria-label={t('browser.closeTab', 'Fechar aba')}
                    title={t('browser.closeTab', 'Fechar aba')}
                    onClick={(e) => {
                      e.stopPropagation();
                      void closeBrowser(candidate.id);
                    }}
                  >
                    <X size={12} aria-hidden="true" />
                  </button>
                </div>
              );
            })}
          </div>
          {availableBrowsers.length < 4 && (
            <button
              type="button"
              className={styles.addTabButton}
              onClick={() => void createNewBrowser()}
              aria-label={t('browser.newTab', 'Nova aba')}
              title={t('browser.newTab', 'Nova aba')}
            >
              <Plus size={14} aria-hidden="true" />
            </button>
          )}
        </div>
      )}
      <form
        className={styles.toolbar}
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          const target = normalizeBrowserURL(address);
          setAddress(target);
          void perform({ type: 'navigate', url: target });
        }}
      >
        <Globe size={16} aria-hidden="true" />
        <button
          type="button"
          disabled={!session || busy}
          onClick={() => void perform({ type: 'back' })}
          aria-label="Voltar"
        >
          <ArrowLeft size={15} />
        </button>
        <button
          type="button"
          disabled={!session || busy}
          onClick={() => void perform({ type: 'forward' })}
          aria-label="Avançar"
        >
          <ArrowRight size={15} />
        </button>
        <input
          type="text"
          inputMode="url"
          className={styles.address}
          aria-label="Endereço da página"
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          required
        />
        <button type="submit" disabled={!session || busy} aria-label="Navegar">
          <ArrowRight size={15} />
        </button>
        <button
          type="button"
          disabled={!session || busy}
          onClick={() => void perform({ type: 'reload' })}
          aria-label="Recarregar página"
        >
          <RefreshCw size={15} />
        </button>
        <button
          type="button"
          disabled={!session || busy}
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
              : 'Somente visualização (clique na página para controlar)'}
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
          onClick={() => void copySelection()}
          aria-label="Copiar seleção para área de transferência"
          title={
            selectionText
              ? `Copiar "${selectionText.slice(0, 40)}${selectionText.length > 40 ? '…' : ''}" (Ctrl+C)`
              : 'Copiar texto selecionado (Ctrl+C)'
          }
        >
          <Copy size={13} aria-hidden="true" />
          <span>
            {selectionText
              ? `Copiar seleção (${selectionText.length})`
              : 'Copiar seleção'}
          </span>
        </button>
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
            className={`${styles.viewportButton} ${dragging ? styles.isDragging : ''}`}
            style={{ width: `${zoom}%` }}
            onClick={handleClick}
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            onMouseLeave={handleMouseLeave}
            onContextMenu={handleContextMenu}
            onKeyDown={handleKeyDown}
            onWheel={(event) => {
              if (!session) return;
              perform({
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
                perform({
                  type: 'scroll',
                  x: session.width / 2,
                  y: session.height / 2,
                  delta_y: (touchStart.current - end) * 2,
                });
              }
              touchStart.current = null;
            }}
            aria-label="Página do navegador (clique para interagir, arraste para selecionar, digite diretamente)"
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
        <button type="submit" disabled={!session || !text || busy} aria-label="Enviar texto">
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
        <button
          type="button"
          disabled={!session || busy}
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
                disabled={!session || busy}
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
