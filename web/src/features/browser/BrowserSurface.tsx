import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Bot,
  Camera,
  Copy,
  ExternalLink,
  Globe,
  Keyboard,
  Lock,
  Maximize2,
  Monitor,
  PanelBottom,
  Plus,
  RefreshCw,
  Scaling,
  Search,
  Send,
  Smartphone,
  Terminal,
  Unlock,
  X,
} from 'lucide-react';
import { getDesktopAuthToken, getWebSocketEndpoint, request } from '../../api';
import { useTranslation } from 'react-i18next';
import { resolveTerminalInfo } from './terminalResolution';
import type { Agent, RuntimeSession } from '../../types';
import styles from './BrowserSurface.module.scss';

export function normalizeBrowserURL(
  input: string,
  searchEngine = 'https://www.google.com/search?q=',
): string {
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
  if (/\s/.test(trimmed) || (!host.includes('.') && !host.includes(':'))) {
    return `${searchEngine}${encodeURIComponent(trimmed)}`;
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

export function shouldUserControlTab(
  candidate: NexusBrowserInfo,
  userControlledSessionIds: ReadonlySet<string>,
): boolean {
  if (userControlledSessionIds.has(candidate.id)) return true;
  return !candidate.owner_runtime_id;
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
  const [viewMode, setViewMode] = useState<'fit' | 'actual'>('fit');
  const [devToolsOpen, setDevToolsOpen] = useState(false);
  const [drawerTab, setDrawerTab] = useState<'inspector' | 'input' | 'agents'>('inspector');
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
  const userControlledSessions = useRef<Set<string>>(new Set());
  const lastHumanInteraction = useRef<number>(Date.now());
  const lastManualTabSelection = useRef<string | null>(null);
  const lastManualTabSelectionTime = useRef<number>(0);
  const controllerRef = useRef(controller);
  controllerRef.current = controller;
  const wantsControlRef = useRef(wantsControl);
  wantsControlRef.current = wantsControl;
  const rootRef = useRef<HTMLElement>(null);
  const addressInputRef = useRef<HTMLInputElement>(null);
  const isAddressFocused = useRef(false);
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
          const isUserOwned = shouldUserControlTab(next, userControlledSessions.current);
          setWantsControl(isUserOwned);
          if (isUserOwned) {
            userControlledSessions.current.add(next.id);
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
    };
  }, [initialSessionId, projectId, initialUrl]);

  const selectBrowser = useCallback(
    (id: string, isManual = true) => {
      if (!id) return;
      if (isManual) {
        lastManualTabSelection.current = id;
        lastManualTabSelectionTime.current = Date.now();
        lastHumanInteraction.current = Date.now();
      }
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
          const isUserOwned = shouldUserControlTab(next, userControlledSessions.current);
          setWantsControl(isUserOwned);
          setController(isUserOwned);
          if (isUserOwned) {
            userControlledSessions.current.add(next.id);
            void browserAPI
              .control(next.id, 'acquire')
              .then((res) => {
                setController(res.controller);
              })
              .catch(() => {
                setController(false);
              });
          }
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
        userControlledSessions.current.delete(id);
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
            selectBrowser(remaining[0].id, true);
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
      userControlledSessions.current.add(next.id);
      setAvailableBrowsers((prev) => [...prev, next]);
      selectBrowser(next.id, true);
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

                const userIsInteracting = Date.now() - lastHumanInteraction.current < 5000;
                const userManuallySwitchedRecently =
                  Date.now() - lastManualTabSelectionTime.current < 10000;
                const userIsControllingCurrent = controllerRef.current || wantsControlRef.current;

                const shouldFocus =
                  isAgentActing &&
                  !agentTabToFocus &&
                  !userIsControllingCurrent &&
                  !userIsInteracting &&
                  !userManuallySwitchedRecently;

                if (shouldFocus) {
                  agentTabToFocus = candidate.id;
                }
              }
            } else if (candidate.owner_runtime_id && candidate.id !== session.id) {
              // Brand new agent-created tab detected for the first time
              const userIsInteracting = Date.now() - lastHumanInteraction.current < 5000;
              const userIsControllingCurrent = controllerRef.current || wantsControlRef.current;
              if (!userIsControllingCurrent && !userIsInteracting && !agentTabToFocus) {
                agentTabToFocus = candidate.id;
              }
            }

            previousBrowserState.current.set(candidate.id, {
              lastActiveAt: candidate.last_active_at || '',
              url: candidate.url || '',
              activeActor: candidate.active_actor || '',
            });
          }

          if (agentTabToFocus && session.id !== agentTabToFocus) {
            if (!isAddressFocused.current) {
              selectBrowser(agentTabToFocus, false);
            }
          }

          if (browserInfo.url && !isAddressFocused.current) {
            setAddress(browserInfo.url);
          }
          if (status.status === 'human_active') setPrivateInput(true);
          if (status.status === 'completed' || status.status === 'cancelled')
            setPrivateInput(false);
        }
      } catch {
        // background poll failure is non-fatal
      }
    };
    void poll();
    const timer = window.setInterval(poll, 1500);
    return () => {
      disposed = true;
      window.clearInterval(timer);
    };
  }, [session, projectId, selectBrowser]);

  const refreshImage = useCallback(async () => {
    if (!session || !visible) return;
    try {
      const headers: Record<string, string> = {};
      const token = getDesktopAuthToken();
      if (token) headers.Authorization = `Bearer ${token}`;
      const response = await fetch(`${browserRoute(session.id)}/frame`, {
        headers,
        cache: 'no-store',
      });
      if (!response.ok) return;
      const blob = await response.blob();
      const nextURL = URL.createObjectURL(blob);
      if (frameObjectURL.current) URL.revokeObjectURL(frameObjectURL.current);
      frameObjectURL.current = nextURL;
      setFrame(nextURL);
    } catch {
      // non-fatal image poll
    }
  }, [session, visible]);

  const sessionId = session?.id;

  useEffect(() => {
    if (!sessionId || !visible) return;
    let stopped = false;
    let ws: WebSocket | null = null;
    let reconnect: number | null = null;

    const connect = () => {
      try {
        const streamBase = getWebSocketEndpoint(browserRoute(sessionId) + '/stream');
        const token = getDesktopAuthToken();
        const fullStreamUrl = token
          ? `${streamBase}${streamBase.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}`
          : streamBase;

        ws = new WebSocket(fullStreamUrl);
        ws.binaryType = 'blob';
        ws.onopen = () => {
          if (!stopped) setLive(true);
        };
        ws.onclose = () => {
          if (stopped) return;
          setLive(false);
          reconnect = window.setTimeout(connect, 1500);
        };
        ws.onerror = () => {
          ws?.close();
        };
        ws.onmessage = (event) => {
          if (stopped) return;
          lastFrameAt.current = Date.now();
          if (event.data instanceof Blob) {
            const nextURL = URL.createObjectURL(event.data);
            if (frameObjectURL.current) URL.revokeObjectURL(frameObjectURL.current);
            frameObjectURL.current = nextURL;
            setFrame(nextURL);
          } else if (typeof event.data === 'string') {
            try {
              const parsed = JSON.parse(event.data) as {
                type?: string;
                data?: string;
                selection?: string;
                title?: string;
              };
              if (parsed.type === 'frame' && parsed.data) {
                if (frameObjectURL.current) {
                  URL.revokeObjectURL(frameObjectURL.current);
                  frameObjectURL.current = '';
                }
                setFrame(`data:image/jpeg;base64,${parsed.data}`);
              }
              if (parsed.selection !== undefined) {
                setSelectionText(parsed.selection);
              }
              if (parsed.title !== undefined && session) {
                setSession((prev) => (prev ? { ...prev, title: parsed.title } : prev));
              }
            } catch {
              // ignore invalid message
            }
          }
        };
      } catch {
        if (!stopped) reconnect = window.setTimeout(connect, 2000);
      }
    };
    connect();
    void refreshImage().catch(() => undefined);

    const fallback = window.setInterval(() => {
      if (Date.now() - lastFrameAt.current > 3000) {
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
  }, [sessionId, visible, refreshImage, session]);

  useEffect(
    () => () => {
      if (frameObjectURL.current) URL.revokeObjectURL(frameObjectURL.current);
    },
    [],
  );

  useEffect(() => {
    if (!sessionId || !wantsControl) {
      setController(false);
      return;
    }
    let stopped = false;
    const acquire = async () => {
      try {
        const value = await browserAPI.control(sessionId, 'acquire');
        if (!stopped) setController(value.controller);
      } catch {
        if (!stopped) setController(false);
      }
    };
    void acquire();

    const interval = window.setInterval(() => {
      if (document.hidden) return;
      void browserAPI
        .control(sessionId, 'renew')
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
    };
  }, [sessionId, wantsControl]);

  useEffect(() => {
    const controlled = userControlledSessions.current;
    return () => {
      if (session?.id && controlled.has(session.id)) {
        void browserAPI.control(session.id, 'release').catch(() => undefined);
      }
    };
  }, [session?.id]);

  const processQueue = useCallback(async () => {
    if (processingRef.current || !session) return;
    processingRef.current = true;
    lastHumanInteraction.current = Date.now();
    setBusy(true);
    try {
      while (queueRef.current.length > 0) {
        const action = queueRef.current.shift();
        if (!action) break;
        if (!controller) {
          try {
            const lease = await browserAPI.control(session.id, 'acquire');
            if (lease.controller) {
              setController(true);
              setWantsControl(true);
              userControlledSessions.current.add(session.id);
            }
          } catch {
            // best effort acquire
          }
        }
        try {
          await browserAPI.action(session.id, action);
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : String(err);
          if (msg.includes('lease unavailable')) {
            const lease = await browserAPI.control(session.id, 'acquire');
            if (lease.controller) {
              setController(true);
              setWantsControl(true);
              userControlledSessions.current.add(session.id);
              await browserAPI.action(session.id, action);
            } else {
              throw err;
            }
          } else {
            throw err;
          }
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      processingRef.current = false;
      setBusy(false);
      void refreshImage();
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
    lastHumanInteraction.current = Date.now();
    if (event.button !== 0 && event.button !== 2) return;
    if (!session) return;
    const img = event.currentTarget.querySelector('img');
    if (img) {
      const rect = img.getBoundingClientRect();
      if (
        event.clientX < rect.left ||
        event.clientX > rect.right ||
        event.clientY < rect.top ||
        event.clientY > rect.bottom
      ) {
        return;
      }
    }
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
    lastHumanInteraction.current = Date.now();
    if (ignoreClick.current) {
      ignoreClick.current = false;
      return;
    }
    if (isDragging.current) {
      return;
    }
    if (event.detail > 1) {
      return;
    }
    const img = event.currentTarget.querySelector('img');
    if (img) {
      const rect = img.getBoundingClientRect();
      if (
        event.clientX < rect.left ||
        event.clientX > rect.right ||
        event.clientY < rect.top ||
        event.clientY > rect.bottom
      ) {
        return;
      }
    }
    const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
    if (!coords) return;
    perform({
      type: 'click',
      x: coords.x,
      y: coords.y,
      click_count: 1,
      button: event.button === 2 ? 'right' : 'left',
    });
  };

  const handleContextMenu = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    if (!session) return;
    const coords = toViewportCoords(event.clientX, event.clientY, event.currentTarget);
    if (!coords) return;
    perform({
      type: 'click',
      x: coords.x,
      y: coords.y,
      click_count: 1,
      button: 'right',
    });
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (!session) return;
    if ((event.ctrlKey || event.metaKey) && (event.key === 'c' || event.key === 'C')) {
      void copySelection();
      return;
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
    setBusy(true);
    void browserAPI
      .snapshot(session.id)
      .then((value) => {
        setSnapshot(value.text);
        setElements(value.elements || []);
        if (value.url) setAddress(value.url);
        setDevToolsOpen(true);
        setDrawerTab('inspector');
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setBusy(false));
  };

  const copySelection = async () => {
    if (!session) return;
    let textToCopy = selectionText;
    if (!textToCopy) {
      try {
        const snap = await browserAPI.snapshot(session.id);
        if (snap.selection) {
          textToCopy = snap.selection;
          setSelectionText(snap.selection);
        }
      } catch {
        // ignore
      }
    }
    if (textToCopy && typeof navigator !== 'undefined' && navigator.clipboard) {
      await navigator.clipboard.writeText(textToCopy);
    }
  };

  const toggleControl = () => {
    if (!session) return;
    lastHumanInteraction.current = Date.now();
    if (controller) {
      setWantsControl(false);
      setController(false);
      userControlledSessions.current.delete(session.id);
      void browserAPI.control(session.id, 'release').catch(() => undefined);
      if (handoff.status === 'human_active') {
        void browserAPI
          .handoff(session.id, 'complete')
          .then((state) => {
            setHandoff(state);
            setPrivateInput(false);
          })
          .catch(() => undefined);
      }
    } else {
      setWantsControl(true);
      userControlledSessions.current.add(session.id);
      void browserAPI
        .control(session.id, 'acquire')
        .then((result) => setController(result.controller))
        .catch(async () => {
          try {
            const state = await browserAPI.handoff(session.id, 'begin');
            setHandoff(state);
            setPrivateInput(true);
            setController(true);
            userControlledSessions.current.add(session.id);
          } catch (err) {
            setController(false);
            setWantsControl(false);
            userControlledSessions.current.delete(session.id);
            setError(err instanceof Error ? err.message : String(err));
          }
        });
    }
  };

  const navigateTo = (rawUrl: string) => {
    if (!session) return;
    const target = normalizeBrowserURL(rawUrl);
    if (!target) return;
    setAddress(target);
    perform({ type: 'navigate', url: target });
  };

  const createAgentConnection = async () => {
    if (!session) return;
    try {
      const result = await browserAPI.grant(session.id);
      setGrants((items) => [...items, result.grant]);
      const command = `nexus mcp add nexus-browser --url '${result.mcp_url}' --bearer-token-env-var NEXUS_BROWSER_TOKEN\nexport NEXUS_BROWSER_TOKEN='${result.token}'`;
      setGrantFallback(command);
      if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(command);
        setConnectionInfo(
          t('browser.grantCopied', 'Configuração copiada para a área de transferência.'),
        );
      } else {
        setConnectionInfo(t('browser.grantReady', 'Configuração gerada.'));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const respondToHuman = async (mode: 'begin' | 'complete' | 'cancel') => {
    if (!session) return;
    lastHumanInteraction.current = Date.now();
    try {
      const state = await browserAPI.handoff(session.id, mode);
      setHandoff(state);
      if (mode === 'begin') {
        setController(true);
        setWantsControl(true);
        userControlledSessions.current.add(session.id);
        setPrivateInput(true);
      } else {
        setController(false);
        setWantsControl(false);
        userControlledSessions.current.delete(session.id);
        setPrivateInput(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      const root = rootRef.current;
      if (!root || !root.contains(document.activeElement)) return;

      if ((e.ctrlKey || e.metaKey) && (e.key === 'l' || e.key === 'L')) {
        e.preventDefault();
        addressInputRef.current?.focus();
        addressInputRef.current?.select();
        return;
      }

      if (
        (e.key === 'F5' || ((e.ctrlKey || e.metaKey) && (e.key === 'r' || e.key === 'R'))) &&
        document.activeElement !== addressInputRef.current
      ) {
        e.preventDefault();
        perform({ type: 'reload' });
        return;
      }

      if (e.key === 'Escape' && devToolsOpen) {
        e.preventDefault();
        setDevToolsOpen(false);
      }
    };

    window.addEventListener('keydown', handleGlobalKeyDown);
    return () => window.removeEventListener('keydown', handleGlobalKeyDown);
  }, [devToolsOpen, perform]);

  return (
    <section
      ref={rootRef}
      className={styles.root}
      aria-label={t('browser.title', 'Navegador Nexus')}
    >
      {/* Linha 1: Tab Bar Nativa */}
      {availableBrowsers.length > 0 && (
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
                        <span className={styles.badgeText}>
                          {t('browser.operating', 'Operando')}
                        </span>
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

      {/* Linha 2: Toolbar Unificada com Omnibox Inteligente e Controles */}
      <div className={styles.toolbar}>
        <div className={styles.navGroup}>
          <button
            type="button"
            disabled={!session || busy}
            onClick={() => void perform({ type: 'back' })}
            aria-label="Voltar"
            title="Voltar"
          >
            <ArrowLeft size={15} />
          </button>
          <button
            type="button"
            disabled={!session || busy}
            onClick={() => void perform({ type: 'forward' })}
            aria-label="Avançar"
            title="Avançar"
          >
            <ArrowRight size={15} />
          </button>
          <button
            type="button"
            disabled={!session || busy}
            onClick={() => void perform({ type: 'reload' })}
            aria-label="Recarregar página"
            title="Recarregar"
          >
            <RefreshCw size={15} />
          </button>
        </div>

        <form
          className={styles.addressForm}
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            lastHumanInteraction.current = Date.now();
            navigateTo(address);
          }}
        >
          <div className={styles.addressWrapper}>
            <span
              className={`${styles.liveIndicator} ${live ? styles.isLive : ''}`}
              title={live ? 'Conexão CDP ao vivo' : 'Capturas / Reconectando'}
              aria-label={live ? 'Ao vivo' : 'Reconectando'}
            />
            <Search size={13} className={styles.searchIcon} aria-hidden="true" />
            <input
              ref={addressInputRef}
              type="text"
              inputMode="url"
              className={styles.address}
              aria-label="Barra de endereço ou busca"
              placeholder="Digite uma URL ou termo de busca..."
              value={address}
              onChange={(e) => {
                lastHumanInteraction.current = Date.now();
                setAddress(e.target.value);
              }}
              onFocus={() => {
                isAddressFocused.current = true;
                lastHumanInteraction.current = Date.now();
              }}
              onBlur={() => {
                isAddressFocused.current = false;
              }}
              required
            />
            <button
              type="submit"
              className={styles.goButton}
              disabled={!session || busy || !address.trim()}
              aria-label="Navegar"
              title="Ir / Pesquisar"
            >
              <ArrowRight size={13} />
            </button>
          </div>
        </form>

        <div className={styles.actionsGroup}>
          {session && (
            <button
              type="button"
              className={styles.leasePill}
              data-controller={controller}
              onClick={toggleControl}
              title={
                controller
                  ? 'Você está no controle exclusivo. Clique para liberar.'
                  : 'Somente visualização. Clique para assumir o controle.'
              }
            >
              {controller ? <Lock size={12} /> : <Unlock size={12} />}
              <span>{controller ? 'Você' : 'Assumir'}</span>
            </button>
          )}

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
            aria-label={
              session?.width === 390
                ? 'Alternar para desktop (1280x800)'
                : 'Alternar para celular (390x844)'
            }
            title={session?.width === 390 ? 'Desktop (1280x800)' : 'Celular (390x844)'}
          >
            {session?.width === 390 ? <Smartphone size={14} /> : <Monitor size={14} />}
          </button>

          <button
            type="button"
            onClick={() => setViewMode((mode) => (mode === 'fit' ? 'actual' : 'fit'))}
            title={
              viewMode === 'fit'
                ? 'Visualização ajustada: clique para tamanho 100% real'
                : 'Tamanho 100% real: clique para ajustar à tela'
            }
            aria-label={viewMode === 'fit' ? 'Ajustado à tela' : 'Tamanho real 100%'}
          >
            {viewMode === 'fit' ? <Scaling size={14} /> : <Maximize2 size={14} />}
            <span className={styles.modeText}>{viewMode === 'fit' ? 'Ajustar' : `${zoom}%`}</span>
          </button>
          {viewMode === 'actual' && (
            <>
              <button
                type="button"
                onClick={() => setZoom((val) => Math.max(50, val - 25))}
                disabled={zoom <= 50}
                aria-label="Diminuir zoom"
                title="Diminuir zoom"
              >
                −
              </button>
              <button
                type="button"
                onClick={() => setZoom((val) => Math.min(250, val + 25))}
                disabled={zoom >= 250}
                aria-label="Aumentar zoom"
                title="Aumentar zoom"
              >
                +
              </button>
            </>
          )}

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
            <Copy size={13} />
            {selectionText && (
              <span className={styles.selectionCount}>({selectionText.length})</span>
            )}
          </button>

          <button
            type="button"
            className={styles.devToolsToggle}
            data-active={devToolsOpen}
            onClick={() => setDevToolsOpen((open) => !open)}
            aria-label="Alternar console de inspeção e conexões de agentes"
            title="Console de inspeção e agentes"
          >
            <PanelBottom size={14} />
            <span className={styles.modeText}>Inspetor</span>
            {(elements.length > 0 || grants.length > 0) && (
              <span className={styles.badgeCount}>{elements.length || grants.length}</span>
            )}
          </button>
        </div>
      </div>

      {/* Alerta de Intervenção Humana (Handoff) */}
      {(handoff.status === 'requested' || handoff.status === 'human_active') && (
        <div className={styles.handoffBanner} role="status">
          <div className={styles.handoffText}>
            <strong>
              {handoff.status === 'requested'
                ? 'Agente solicita intervenção:'
                : 'Você está no controle:'}
            </strong>
            <span>{handoff.reason || 'Conclua a autenticação na página.'}</span>
          </div>
          <div className={styles.handoffActions}>
            {handoff.status === 'requested' ? (
              <button type="button" onClick={() => void respondToHuman('begin')}>
                Assumir para autenticar
              </button>
            ) : (
              <>
                <button type="button" onClick={() => void respondToHuman('complete')}>
                  Concluído · devolver ao agente
                </button>
                <button type="button" onClick={() => void respondToHuman('cancel')}>
                  Cancelar
                </button>
              </>
            )}
          </div>
        </div>
      )}

      {/* Viewport Principal (100% Altura Flexível Restante) */}
      <div
        className={`${styles.viewportScroll} ${viewMode === 'fit' ? styles.isFitContainer : ''}`}
      >
        {frame ? (
          <button
            type="button"
            className={`${styles.viewportButton} ${viewMode === 'fit' ? styles.isFit : ''} ${dragging ? styles.isDragging : ''}`}
            style={viewMode === 'actual' ? { width: `${zoom}%` } : undefined}
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

      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}

      {/* Gaveta Retrátil de DevTools / Inspeção (Bottom) */}
      {devToolsOpen && (
        <div
          className={styles.devToolsDrawer}
          role="region"
          aria-label="Console de inspeção e agentes"
        >
          <div className={styles.drawerHeader}>
            <div className={styles.drawerTabs} role="tablist">
              <button
                type="button"
                role="tab"
                aria-selected={drawerTab === 'inspector'}
                className={styles.drawerTab}
                data-active={drawerTab === 'inspector'}
                onClick={() => setDrawerTab('inspector')}
              >
                Seletores & DOM ({elements.length})
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={drawerTab === 'input'}
                className={styles.drawerTab}
                data-active={drawerTab === 'input'}
                onClick={() => setDrawerTab('input')}
              >
                Entrada Manual & Teclas
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={drawerTab === 'agents'}
                className={styles.drawerTab}
                data-active={drawerTab === 'agents'}
                onClick={() => setDrawerTab('agents')}
              >
                Agentes MCP ({grants.length})
              </button>
            </div>
            <div className={styles.drawerActions}>
              {drawerTab === 'inspector' && (
                <button
                  type="button"
                  className={styles.drawerActionButton}
                  onClick={inspect}
                  disabled={!session || busy}
                  title="Inspecionar elementos e capturar snapshot do DOM"
                >
                  <Camera size={13} aria-hidden="true" />
                  <span>Capturar DOM</span>
                </button>
              )}
              <button
                type="button"
                className={styles.drawerCloseButton}
                onClick={() => setDevToolsOpen(false)}
                aria-label="Fechar painel de inspeção"
                title="Fechar painel de inspeção (Esc)"
              >
                <X size={14} aria-hidden="true" />
              </button>
            </div>
          </div>
          <div className={styles.drawerBody}>
            {drawerTab === 'inspector' && (
              <div className={styles.inspectorContent}>
                {elements.length > 0 && (
                  <div className={styles.inspectionActions}>
                    {elements.map((item, index) => (
                      <button
                        key={`${item.selector}:${index}`}
                        type="button"
                        disabled={!session || busy}
                        title={item.selector}
                        onClick={() =>
                          void perform({ type: 'click_selector', selector: item.selector })
                        }
                      >
                        {item.label || item.text || item.tag}
                      </button>
                    ))}
                  </div>
                )}
                {snapshot ? (
                  <pre className={styles.snapshotText}>{snapshot}</pre>
                ) : (
                  <p className={styles.emptyNotice}>
                    Nenhum snapshot textual carregado. Clique em &quot;Capturar DOM&quot; para
                    inspecionar os elementos da página.
                  </p>
                )}
              </div>
            )}
            {drawerTab === 'input' && (
              <form
                className={styles.drawerInputBar}
                onSubmit={(event) => {
                  event.preventDefault();
                  if (text) {
                    void perform({ type: 'type', text });
                    setText('');
                  }
                }}
              >
                <Keyboard size={15} aria-hidden="true" />
                <input
                  type={privateInput ? 'password' : 'text'}
                  autoComplete="off"
                  aria-label={
                    privateInput ? 'Texto sensível para login' : 'Texto para o elemento selecionado'
                  }
                  placeholder="Digitar texto para enviar ao elemento remoto..."
                  value={text}
                  onChange={(event) => setText(event.target.value)}
                />
                <button
                  type="button"
                  onClick={() => setPrivateInput((value) => !value)}
                  title="Ocultar caracteres digitados neste campo"
                >
                  {privateInput ? 'Privado' : 'Ocultar'}
                </button>
                <button
                  type="submit"
                  disabled={!session || !text || busy}
                  aria-label="Enviar texto"
                >
                  <Send size={14} />
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
            )}
            {drawerTab === 'agents' && (
              <div className={styles.agentsContent}>
                <div className={styles.agentActionRow}>
                  <span>
                    Agentes do Nexus recebem o Browser automaticamente. Crie acessos temporários
                    para conexões MCP manuais.
                  </span>
                  <button type="button" onClick={() => void createAgentConnection()}>
                    Gerar acesso temporário
                  </button>
                </div>
                {connectionInfo && <p role="status">{connectionInfo}</p>}
                {grantFallback && (
                  <textarea
                    readOnly
                    value={grantFallback}
                    aria-label="Configuração MCP de uso único"
                  />
                )}
                {grants.map((grant) => (
                  <div className={styles.grantRow} key={grant.id}>
                    <span>
                      {grant.label} · expira às {new Date(grant.expires_at).toLocaleTimeString()}
                    </span>
                    <button
                      type="button"
                      onClick={() => {
                        if (!session) return;
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
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  );
};
