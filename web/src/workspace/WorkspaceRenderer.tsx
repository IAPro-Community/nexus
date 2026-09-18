import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  AppWindow,
  Gauge,
  History,
  Home,
  Layers,
  LayoutGrid,
  Maximize2,
  Minimize2,
  Minus,
  Paintbrush,
  PanelsTopLeft,
  Plus,
  RotateCcw,
  Settings,
  Sparkles,
  TerminalSquare,
  Workflow,
  X,
} from 'lucide-react';
import {
  IconButton,
  ContextMenu,
  contextMenuFromEvent,
  type ContextMenuItem,
  type ContextMenuPoint,
} from '../design-system';
import {
  isSurfaceMatch,
  listStacks,
  listSurfaces,
  surfaceViewId,
  type WorkspaceStack,
  type WorkspaceSurface,
} from './model';
import { useWorkspace } from './WorkspaceProvider';
import { useWorkspacePresentation } from './WorkspacePresentationProvider';
import { useTranslation } from 'react-i18next';
import { isPtySurface } from '../app/surfaces';
import { findTileSplitters } from './arrange';
import { isWindowedPresentationMode, mosaicDropTargetViewId } from './presentation';
import { ptyWindowHeading } from './ptyLiveChrome';
import { liveChromeFor, usePtyLiveChromeOptional } from './PtyLiveChromeContext';
import { ARRANGE_MENU_PRESETS, type ArrangeMenuPreset } from './arrangePresets';

const WINDOW_ACCENTS = ['#38bdf8', '#22c55e', '#f59e0b', '#f472b6', '#a78bfa', '#fb7185'];
const WINDOW_ICONS = ['⌘', '⚡', '◆', '●', '★', '◎'];

const PRODUCT_TAB_ICONS: Record<
  string,
  React.ComponentType<{ size?: number; strokeWidth?: number }>
> = {
  overview: Home,
  terminals: TerminalSquare,
  work: Sparkles,
  missions: Workflow,
  agents: PanelsTopLeft,
  settings: Settings,
  resources: Gauge,
  sessions: History,
  projects: LayoutGrid,
};

const isPinnedProductTab = (surface: WorkspaceSurface) =>
  surface.closable === false ||
  surface.type === 'overview' ||
  surface.type === 'terminals' ||
  surface.type === 'work';

function surfaceIdentity(surface: WorkspaceSurface, customTitle?: string): string {
  return (
    customTitle || surface.data?.agentName || surface.title.replace(/\s·\s.*$/, '') || surface.title
  );
}

const WindowChromeMenu: React.FC<{
  viewId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customTitle: string;
  accent: string;
  icon: string;
  onPatch: (chrome: { customTitle?: string; accent?: string; icon?: string }) => void;
  position?: ContextMenuPoint | null;
}> = ({ open, onOpenChange, customTitle, accent, icon, onPatch, position }) => {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement>(null);
  const [draft, setDraft] = useState(customTitle);
  const [menuPos, setMenuPos] = useState<{ top: number; right?: number; left?: number } | null>(
    null,
  );

  useEffect(() => {
    setDraft(customTitle);
  }, [customTitle]);

  useEffect(() => {
    if (!open) {
      setMenuPos(null);
      return;
    }
    const compute = () => {
      if (position) {
        setMenuPos({ top: position.y, left: position.x });
        return;
      }
      const anchor = rootRef.current?.querySelector('button');
      if (!anchor) return;
      const rect = anchor.getBoundingClientRect();
      setMenuPos({
        top: rect.bottom + 6,
        right: Math.max(8, window.innerWidth - rect.right),
      });
    };
    compute();
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if ((target as Element).closest?.('.nx-desktop-window__chrome-menu')) return;
      onOpenChange(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onOpenChange(false);
    };
    window.addEventListener('pointerdown', onPointerDown, true);
    window.addEventListener('keydown', onKey);
    const onResize = () => compute();
    window.addEventListener('resize', onResize);
    window.addEventListener('scroll', onResize, true);
    return () => {
      window.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('keydown', onKey);
      window.removeEventListener('resize', onResize);
      window.removeEventListener('scroll', onResize, true);
    };
  }, [open, onOpenChange, position]);

  const commitTitle = () => {
    const trimmed = draft.trim();
    if (trimmed !== customTitle) {
      onPatch({ customTitle: trimmed });
    }
  };

  const menu =
    open && menuPos
      ? createPortal(
          <div
            className="nx-desktop-window__chrome-menu"
            role="dialog"
            aria-label={t('workspace.customizeWindow', 'Personalizar janela')}
            style={{
              top: menuPos.top,
              ...(menuPos.left != null ? { left: menuPos.left } : { right: menuPos.right }),
            }}
            onPointerDown={(event) => event.stopPropagation()}
          >
            <label>
              {t('common.name', 'Nome')}
              <input
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                onBlur={commitTitle}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    commitTitle();
                    onOpenChange(false);
                  }
                }}
                placeholder={t('workspace.windowTitlePlaceholder', 'Identidade da janela')}
                autoFocus
              />
            </label>
            <div
              className="nx-desktop-window__chrome-row"
              aria-label={t('workspace.titleColor', 'Cor do título')}
            >
              <button
                type="button"
                className="nx-desktop-window__swatch nx-desktop-window__swatch--clear"
                data-active={!accent ? 'true' : 'false'}
                title={t('workspace.noColor', 'Sem cor')}
                onClick={() => {
                  onPatch({ accent: '' });
                  onOpenChange(false);
                }}
              />
              {WINDOW_ACCENTS.map((color) => (
                <button
                  key={color}
                  type="button"
                  className="nx-desktop-window__swatch"
                  data-active={accent === color ? 'true' : 'false'}
                  style={{ background: color }}
                  title={color}
                  onClick={() => {
                    onPatch({ accent: color });
                    onOpenChange(false);
                  }}
                />
              ))}
            </div>
            <div
              className="nx-desktop-window__chrome-row"
              aria-label={t('workspace.icon', 'Ícone')}
            >
              <button
                type="button"
                className="nx-desktop-window__icon-pick"
                data-active={!icon ? 'true' : 'false'}
                title={t('workspace.noIcon', 'Sem ícone')}
                onClick={() => {
                  onPatch({ icon: '' });
                  onOpenChange(false);
                }}
              >
                —
              </button>
              {WINDOW_ICONS.map((glyph) => (
                <button
                  key={glyph}
                  type="button"
                  className="nx-desktop-window__icon-pick"
                  data-active={icon === glyph ? 'true' : 'false'}
                  onClick={() => {
                    onPatch({ icon: glyph });
                    onOpenChange(false);
                  }}
                >
                  {glyph}
                </button>
              ))}
            </div>
          </div>,
          document.body,
        )
      : null;

  return (
    <div className="nx-desktop-window__chrome" ref={rootRef}>
      <IconButton
        label={t('workspace.customizeWindow', 'Personalizar janela')}
        aria-expanded={open}
        onClick={(event) => {
          event.stopPropagation();
          onOpenChange(!open);
        }}
      >
        <Paintbrush size={13} />
      </IconButton>
      {menu}
    </div>
  );
};

const ArrangeLayoutMenu: React.FC<{
  activePreset: ArrangeMenuPreset;
  onSelect: (preset: ArrangeMenuPreset) => void;
  onReset?: () => void;
}> = ({ activePreset, onSelect, onReset }) => {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [menuPos, setMenuPos] = useState<{ top: number; right: number } | null>(null);

  useEffect(() => {
    if (!open) {
      setMenuPos(null);
      return;
    }
    const anchor = rootRef.current?.querySelector('button');
    if (anchor) {
      const rect = anchor.getBoundingClientRect();
      setMenuPos({
        top: rect.bottom + 6,
        right: Math.max(8, window.innerWidth - rect.right),
      });
    }
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if ((target as Element).closest?.('.nx-arrange-menu__panel')) return;
      setOpen(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    const listen = window.setTimeout(() => {
      window.addEventListener('pointerdown', onPointerDown, true);
    }, 0);
    window.addEventListener('keydown', onKey);
    return () => {
      window.clearTimeout(listen);
      window.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('keydown', onKey);
    };
  }, [open]);

  const panel =
    open && menuPos
      ? createPortal(
          <div
            className="nx-arrange-menu__panel"
            role="menu"
            aria-label={t('workspace.arrangeWindows', 'Arranjar janelas')}
            style={{ top: menuPos.top, right: menuPos.right }}
            onPointerDown={(event) => event.stopPropagation()}
          >
            {ARRANGE_MENU_PRESETS.map((preset) => (
              <button
                key={preset.id}
                type="button"
                role="menuitemradio"
                className="nx-arrange-menu__item"
                aria-checked={activePreset === preset.id}
                data-active={activePreset === preset.id ? 'true' : undefined}
                onClick={() => {
                  onSelect(preset.id);
                  setOpen(false);
                }}
              >
                <span>{preset.label}</span>
                <small>{preset.hint}</small>
              </button>
            ))}
            {onReset && (
              <>
                <div className="nx-arrange-menu__separator" role="separator" />
                <button
                  type="button"
                  role="menuitem"
                  className="nx-arrange-menu__item"
                  onClick={() => {
                    setOpen(false);
                    onReset();
                  }}
                >
                  <span className="nx-arrange-menu__reset-label">
                    <RotateCcw size={12} />
                    <span>{t('workspace.resetLayout', 'Redefinir layout')}</span>
                  </span>
                  <small>{t('workspace.restoreDefault', 'Restaurar padrão')}</small>
                </button>
              </>
            )}
          </div>,
          document.body,
        )
      : null;

  return (
    <div className="nx-arrange-menu" ref={rootRef}>
      <button
        type="button"
        className="nx-arrange-menu__trigger"
        aria-expanded={open}
        aria-haspopup="menu"
        title={t('workspace.arrangeWindows', 'Arranjar janelas')}
        aria-label={t('workspace.arrangeWindows', 'Arranjar janelas')}
        onClick={() => setOpen((value) => !value)}
      >
        <LayoutGrid size={12} />
        <span>{t('workspace.arrange', 'Arranjar')}</span>
      </button>
      {panel}
    </div>
  );
};

export type WorkspaceCreateActions = {
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
};

function useCompactViewport(): boolean {
  const [compact, setCompact] = useState(() => window.matchMedia('(max-width: 720px)').matches);
  useEffect(() => {
    const media = window.matchMedia('(max-width: 720px)');
    const update = () => setCompact(media.matches);
    media.addEventListener?.('change', update);
    return () => media.removeEventListener?.('change', update);
  }, []);
  return compact;
}

export const WorkspaceRenderer: React.FC<{
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
  onActivateSurface?: (surface: WorkspaceSurface) => void;
  createActions?: WorkspaceCreateActions;
}> = ({ renderSurface, onRequestClose, onActivateSurface, createActions }) => {
  const workspace = useWorkspace();
  const presentation = useWorkspacePresentation();
  const compact = useCompactViewport();
  const surfaces = useMemo(() => listSurfaces(workspace.model.root), [workspace.model.root]);
  const floatingSurfaces = useMemo(() => surfaces.filter(isPtySurface), [surfaces]);
  const surfaceSignature = floatingSurfaces.map(surfaceViewId).join('|');
  const windowSignature = Object.keys(presentation.state.windows).sort().join('|');

  useEffect(() => {
    if (surfaceSignature !== windowSignature) presentation.sync(floatingSurfaces);
  }, [surfaceSignature, windowSignature, floatingSurfaces, presentation]);

  const stacks = listStacks(workspace.model.root);
  const activeStack = stacks.find((stack) => stack.activeId) ?? stacks[0];

  return (
    <div
      className={compact ? 'nx-workspace nx-workspace--compact' : 'nx-workspace'}
      data-tour="workspace"
    >
      {activeStack && (
        <WorkspaceStackView
          stack={activeStack}
          renderSurface={renderSurface}
          onRequestClose={onRequestClose}
          onActivateSurface={onActivateSurface}
          createActions={createActions}
        />
      )}
    </div>
  );
};

const WorkspaceStackView: React.FC<{
  stack: WorkspaceStack;
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
  onActivateSurface?: (surface: WorkspaceSurface) => void;
  createActions?: WorkspaceCreateActions;
}> = ({ stack, renderSurface, onRequestClose, onActivateSurface, createActions }) => {
  const { t } = useTranslation();
  const { activate, close, move } = useWorkspace();
  const [draggedSurface, setDraggedSurface] = useState<string | null>(null);
  const [tabMenu, setTabMenu] = useState<{
    surface: WorkspaceSurface;
    point: ContextMenuPoint;
  } | null>(null);
  const productTabs = useMemo(() => {
    const pinnedOrder = ['overview', 'terminals', 'work'];
    return [...stack.tabs.filter((tab) => !isPtySurface(tab))].sort((a, b) => {
      const ai = pinnedOrder.indexOf(a.type);
      const bi = pinnedOrder.indexOf(b.type);
      if (ai === -1 && bi === -1) return 0;
      if (ai === -1) return 1;
      if (bi === -1) return -1;
      return ai - bi;
    });
  }, [stack.tabs]);
  const ptyTabs = useMemo(() => stack.tabs.filter(isPtySurface), [stack.tabs]);
  const activeRaw = stack.tabs.find((tab) => isSurfaceMatch(tab, stack.activeId));
  // If a PTY was left as activeId (legacy layouts), treat the Terminais product tab as active.
  const activeProduct =
    activeRaw && !isPtySurface(activeRaw)
      ? activeRaw
      : productTabs.find((tab) => tab.type === 'terminals') || productTabs[0];
  const activeId = activeProduct?.id || stack.activeId;
  const closeSurface = (surface: WorkspaceSurface) =>
    onRequestClose ? onRequestClose(surface) : close(surface.id);
  const legacyTitleKeys: Record<string, string> = {
    overview: 'nav.overview',
    work: 'nav.work',
    missions: 'nav.missions',
    terminals: 'nav.terminals',
    agents: 'nav.agents',
    maestro: 'nav.maestro',
    sessions: 'nav.sessions',
    settings: 'nav.settings',
    resources: 'nav.resources',
    'legacy-runtimes': 'nav.runtimes',
    'legacy-providers': 'nav.providers',
    'legacy-events': 'nav.events',
  };
  const displayTitle = (surface: WorkspaceSurface) => {
    const key = surface.titleKey || legacyTitleKeys[surface.type];
    return key ? t(key, surface.titleParams) : surface.title;
  };

  return (
    <section
      className="nx-workspace-stack"
      data-stack-id={stack.id}
      onDragOver={(event) => {
        if (event.dataTransfer.types.includes('application/x-nexus-surface'))
          event.preventDefault();
      }}
      onDrop={(event) => {
        const id = event.dataTransfer.getData('application/x-nexus-surface') || draggedSurface;
        if (id) move(id, stack.id);
        setDraggedSurface(null);
      }}
    >
      <header className="nx-workspace-stack__tabs">
        <div className="nx-workspace-tabs" role="tablist" aria-label={t('shell.openSurfaces')}>
          {productTabs.map((surface) => {
            const TabIcon = PRODUCT_TAB_ICONS[surface.type] || Layers;
            const pinned = isPinnedProductTab(surface);
            return (
              <button
                draggable={!pinned}
                key={surface.id}
                type="button"
                role="tab"
                aria-selected={activeId === surface.id}
                data-active={activeId === surface.id ? 'true' : 'false'}
                data-pinned={pinned ? 'true' : undefined}
                data-kind={surface.type}
                data-attention={surface.data?.hasAttention === 'true' ? 'true' : undefined}
                data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
                className="nx-workspace-tab"
                onDragStart={(event) => {
                  if (pinned) {
                    event.preventDefault();
                    return;
                  }
                  setDraggedSurface(surface.id);
                  event.dataTransfer.setData('application/x-nexus-surface', surface.id);
                  event.dataTransfer.effectAllowed = 'move';
                }}
                onClick={() =>
                  onActivateSurface ? onActivateSurface(surface) : activate(surface.id)
                }
                onContextMenu={(event) => {
                  if (surface.closable === false) return;
                  setTabMenu({ surface, point: contextMenuFromEvent(event) });
                }}
              >
                <span className="nx-workspace-tab__icon" aria-hidden="true">
                  <TabIcon size={16} strokeWidth={2.25} />
                </span>
                {(surface.data?.hasAttention === 'true' ||
                  surface.data?.unreadAttention === 'true') && (
                  <span
                    className="nx-workspace-tab__attention-dot"
                    data-kind={surface.data?.attentionKind || undefined}
                    data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
                    role="img"
                    aria-label={t('workspace.needsAttention', 'needs attention')}
                  />
                )}
                <span className="nx-workspace-tab__label">{displayTitle(surface)}</span>
                {surface.closable !== false && (
                  <span
                    className="nx-workspace-tab__close"
                    role="button"
                    aria-label={t('workspace.closeNamed', { name: displayTitle(surface) })}
                    onClick={(event) => {
                      event.stopPropagation();
                      closeSurface(surface);
                    }}
                  >
                    <X size={11} />
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </header>
      <ContextMenu
        open={tabMenu?.point ?? null}
        onClose={() => setTabMenu(null)}
        label={t('workspace.menu')}
        items={
          tabMenu && tabMenu.surface.closable !== false
            ? [
                {
                  type: 'item',
                  id: 'close',
                  label: t('workspace.closeNamed', { name: displayTitle(tabMenu.surface) }),
                  danger: true,
                  icon: <X size={13} />,
                  onSelect: () => closeSurface(tabMenu.surface),
                },
              ]
            : []
        }
      />
      <div className="nx-workspace-stack__body">
        {productTabs.map((surface) => (
          <div
            key={surface.id}
            role="tabpanel"
            aria-hidden={activeId !== surface.id}
            data-active={activeId === surface.id ? 'true' : 'false'}
            className="nx-workspace-panel"
          >
            {surface.type === 'terminals' ? (
              <TerminalsHost
                ptySurfaces={ptyTabs}
                renderSurface={renderSurface}
                onRequestClose={onRequestClose}
                createActions={createActions}
                active={activeId === surface.id}
              />
            ) : (
              renderSurface(surface)
            )}
          </div>
        ))}
      </div>
    </section>
  );
};

/** Owns all PTYs so they stay mounted when other product tabs are active. */
const TerminalsHost: React.FC<{
  ptySurfaces: WorkspaceSurface[];
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
  createActions?: WorkspaceCreateActions;
  active: boolean;
}> = ({ ptySurfaces, renderSurface, onRequestClose, createActions, active }) => {
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const presentation = useWorkspacePresentation();
  const liveChrome = usePtyLiveChromeOptional();
  const liveByViewId = liveChrome?.byViewId || {};
  const [tabMenu, setTabMenu] = useState<{
    surface: WorkspaceSurface;
    point: ContextMenuPoint;
  } | null>(null);
  const [chromeMenuViewId, setChromeMenuViewId] = useState<string | null>(null);
  const [chromeMenuPoint, setChromeMenuPoint] = useState<ContextMenuPoint | null>(null);
  const focusedPtyId =
    ptySurfaces.find((s) => surfaceViewId(s) === presentation.state.activePtyViewId)?.id ||
    ptySurfaces[0]?.id ||
    '';
  const windowed = isWindowedPresentationMode(presentation.state.mode);
  const minimizedPtys = ptySurfaces.filter(
    (surface) => presentation.state.windows[surfaceViewId(surface)]?.minimized,
  );

  useEffect(() => {
    const activePtyViewId = presentation.state.activePtyViewId;
    const setActivePty = presentation.setActivePty;
    if (ptySurfaces.length === 0) {
      if (activePtyViewId) setActivePty('');
      return;
    }
    const viewIds = ptySurfaces.map((s) => surfaceViewId(s));
    if (!activePtyViewId || !viewIds.includes(activePtyViewId)) {
      setActivePty(viewIds[0] || '');
    }
  }, [ptySurfaces, presentation.state.activePtyViewId, presentation.setActivePty]);

  const focusPty = (surface: WorkspaceSurface) => {
    const viewId = surfaceViewId(surface);
    presentation.setActivePty(viewId);
    if (windowed) {
      presentation.focus(viewId);
    }
  };

  return (
    <div className="nx-terminals-host" data-active={active ? 'true' : 'false'}>
      <header className="nx-terminals-host__chrome">
        <div className="nx-terminals-host__title">
          <TerminalSquare size={14} />
          <strong>
            {t('nav.terminals')} · {ptySurfaces.length} PTY{ptySurfaces.length === 1 ? '' : 's'}
          </strong>
        </div>
        {windowed && minimizedPtys.length > 0 ? (
          <div className="nx-minimized-chips" aria-label={t('workspace.minimizedShelf')}>
            <span className="nx-minimized-chips__label">{t('workspace.minimizedShelf')}</span>
            {minimizedPtys.map((surface) => {
              const viewId = surfaceViewId(surface);
              const win = presentation.state.windows[viewId];
              const live = liveChromeFor(liveByViewId, viewId);
              const label = ptyWindowHeading({
                customTitle: win?.customTitle,
                identity: surfaceIdentity(surface, win?.customTitle),
                liveTitle: live.title,
              }).heading;
              return (
                <button
                  key={viewId}
                  type="button"
                  title={t('workspace.restoreNamed', { name: label })}
                  data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
                  onClick={() => {
                    presentation.minimize(viewId);
                    presentation.focus(viewId);
                  }}
                >
                  {win?.icon ? <span aria-hidden="true">{win.icon}</span> : <Minus size={11} />}
                  {label}
                </button>
              );
            })}
          </div>
        ) : (
          <div className="nx-minimized-chips" aria-hidden="true" />
        )}
        <div className="nx-terminals-host__chrome-cluster">
          <div
            className="nx-presentation-toggle"
            role="group"
            aria-label={t('workspace.terminalPresentation', 'Apresentação de terminais')}
          >
            <button
              type="button"
              data-active={presentation.state.mode === 'TABS' ? 'true' : 'false'}
              onClick={() => presentation.setMode('TABS')}
              title={t('workspace.ptyTabs', 'Abas internas de PTY')}
              aria-label={t('workspace.tabsMode', 'Modo de abas')}
            >
              <PanelsTopLeft size={11} />
              <span>{t('workspace.tabs', 'Abas')}</span>
            </button>
            <button
              type="button"
              data-active={presentation.state.mode === 'DESKTOP' ? 'true' : 'false'}
              onClick={() => presentation.setMode('DESKTOP')}
              title={t('workspace.floatingWindows', 'Janelas flutuantes')}
              aria-label={t('workspace.windowsMode', 'Modo de janelas')}
            >
              <AppWindow size={11} />
              <span>{t('workspace.windows', 'Janelas')}</span>
            </button>
            <button
              type="button"
              data-active={presentation.state.mode === 'MOSAIC' ? 'true' : 'false'}
              onClick={() => presentation.setMode('MOSAIC')}
              title={t('workspace.mosaicLayout', 'Mosaico lado a lado')}
              aria-label={t('workspace.mosaicMode', 'Modo de mosaico')}
            >
              <LayoutGrid size={11} />
              <span>{t('workspace.mosaic', 'Mosaico')}</span>
            </button>
          </div>
          {windowed && (
            <ArrangeLayoutMenu
              activePreset={presentation.state.lastArrangePreset}
              onSelect={(preset) => presentation.rearrangePreset(preset)}
              onReset={workspace.reset}
            />
          )}
        </div>
      </header>
      <div className="nx-terminals-host__body">
        {ptySurfaces.length === 0 ? (
          <div className="nx-terminals-empty">
            <TerminalSquare size={26} />
            <strong>{t('workspace.noTerminalsOpen', 'Nenhum terminal aberto')}</strong>
            <p>
              {t(
                'workspace.noTerminalsDesc',
                'Inicie um shell de comando, abra uma nova sessão de IA rápida ou configure um agente autônomo neste projeto.',
              )}
            </p>
            {createActions && (
              <div className="nx-terminals-empty__actions">
                {createActions.onProjectShell && (
                  <button
                    type="button"
                    className="nx-button nx-button--terminal"
                    data-size="sm"
                    onClick={createActions.onProjectShell}
                  >
                    <TerminalSquare size={13} />
                    <span>{t('workspace.openTerminal', 'Abrir Terminal')}</span>
                  </button>
                )}
                {createActions.onNewAISession && (
                  <button
                    type="button"
                    className="nx-button"
                    data-size="sm"
                    onClick={createActions.onNewAISession}
                  >
                    <Sparkles size={13} />
                    <span>{t('workspace.newAISession', 'Nova Sessão IA')}</span>
                  </button>
                )}
                {createActions.onNewAgent && (
                  <button
                    type="button"
                    className="nx-button"
                    data-tone="brand"
                    data-size="sm"
                    onClick={createActions.onNewAgent}
                  >
                    <Plus size={13} />
                    <span>{t('workspace.newAgent', 'Novo Agente')}</span>
                  </button>
                )}
              </div>
            )}
          </div>
        ) : windowed ? (
          <DesktopWorkspace
            surfaces={ptySurfaces}
            renderSurface={renderSurface}
            onRequestClose={onRequestClose}
          />
        ) : (
          <div className="nx-terminals-inner-tabs">
            <div
              className="nx-workspace-tabs nx-pty-tabs"
              role="tablist"
              aria-label={t('workspace.ptyTabs', 'Abas PTY')}
            >
              {ptySurfaces.map((surface) => {
                const viewId = surfaceViewId(surface);
                const win = presentation.state.windows[viewId];
                const live = liveChromeFor(liveByViewId, viewId);
                const tabLabel = ptyWindowHeading({
                  customTitle: win?.customTitle,
                  identity: surfaceIdentity(surface, win?.customTitle),
                  liveTitle: live.title,
                }).heading;
                const accent = win?.accent || '';
                const icon = win?.icon || '';
                return (
                  <div
                    key={surface.id}
                    className="nx-pty-tab"
                    data-active={focusedPtyId === surface.id ? 'true' : 'false'}
                    data-accented={accent ? 'true' : undefined}
                    style={
                      accent
                        ? ({ ['--nx-window-accent' as string]: accent } as React.CSSProperties)
                        : undefined
                    }
                  >
                    <button
                      type="button"
                      role="tab"
                      className="nx-workspace-tab"
                      data-active={focusedPtyId === surface.id ? 'true' : 'false'}
                      data-attention={
                        surface.data?.hasAttention === 'true' || live.questionnaire
                          ? 'true'
                          : undefined
                      }
                      data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
                      aria-selected={focusedPtyId === surface.id}
                      onClick={() => {
                        setChromeMenuViewId(null);
                        focusPty(surface);
                      }}
                      onContextMenu={(event) => {
                        focusPty(surface);
                        setChromeMenuViewId(null);
                        setTabMenu({ surface, point: contextMenuFromEvent(event) });
                      }}
                    >
                      {icon ? (
                        <span className="nx-pty-tab__glyph" aria-hidden="true">
                          {icon}
                        </span>
                      ) : null}
                      <span className="nx-workspace-tab__label">{tabLabel}</span>
                    </button>
                    <WindowChromeMenu
                      viewId={viewId}
                      open={chromeMenuViewId === viewId}
                      position={chromeMenuViewId === viewId ? chromeMenuPoint : null}
                      onOpenChange={(next) => {
                        setChromeMenuPoint(null);
                        setChromeMenuViewId(next ? viewId : null);
                      }}
                      customTitle={win?.customTitle || ''}
                      accent={accent}
                      icon={icon}
                      onPatch={(chrome) => presentation.patchChrome(viewId, chrome)}
                    />
                    {surface.closable !== false && onRequestClose && (
                      <span
                        className="nx-workspace-tab__close"
                        role="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          onRequestClose(surface);
                        }}
                      >
                        <X size={11} />
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
            <div className="nx-terminals-inner-panels">
              {ptySurfaces.map((surface) => (
                <div
                  key={surface.id}
                  className="nx-workspace-panel"
                  data-active={focusedPtyId === surface.id ? 'true' : 'false'}
                  aria-hidden={focusedPtyId !== surface.id}
                >
                  {renderSurface(surface)}
                </div>
              ))}
            </div>
            <ContextMenu
              open={tabMenu?.point ?? null}
              onClose={() => setTabMenu(null)}
              label={t('workspace.menu')}
              items={
                tabMenu
                  ? ([
                      {
                        type: 'item',
                        id: 'customize',
                        label: t('workspace.windowCustomize'),
                        icon: <Paintbrush size={13} />,
                        onSelect: () => {
                          const viewId = surfaceViewId(tabMenu.surface);
                          setChromeMenuPoint(tabMenu.point);
                          setChromeMenuViewId(viewId);
                        },
                      },
                      ...(tabMenu.surface.closable !== false && onRequestClose
                        ? ([
                            { type: 'separator', id: 'sep-close' },
                            {
                              type: 'item',
                              id: 'close',
                              label: t('workspace.closeNamed', {
                                name:
                                  presentation.state.windows[surfaceViewId(tabMenu.surface)]
                                    ?.customTitle ||
                                  tabMenu.surface.data?.agentName ||
                                  tabMenu.surface.title,
                              }),
                              danger: true,
                              icon: <X size={13} />,
                              onSelect: () => onRequestClose(tabMenu.surface),
                            },
                          ] satisfies ContextMenuItem[])
                        : []),
                    ] satisfies ContextMenuItem[])
                  : []
              }
            />
          </div>
        )}
      </div>
    </div>
  );
};

const DesktopWorkspace: React.FC<{
  surfaces: WorkspaceSurface[];
  renderSurface: (surface: WorkspaceSurface) => React.ReactNode;
  onRequestClose?: (surface: WorkspaceSurface) => void;
}> = ({ surfaces, renderSurface, onRequestClose }) => {
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const presentation = useWorkspacePresentation();
  const liveChrome = usePtyLiveChromeOptional();
  const liveByViewId = liveChrome?.byViewId || {};
  const hostRef = useRef<HTMLDivElement>(null);
  const [chromeMenuViewId, setChromeMenuViewId] = useState<string | null>(null);
  const [chromeMenuPoint, setChromeMenuPoint] = useState<ContextMenuPoint | null>(null);
  const [windowMenu, setWindowMenu] = useState<{
    viewId: string;
    surface: WorkspaceSurface;
    point: ContextMenuPoint;
  } | null>(null);
  const [mosaicDrag, setMosaicDrag] = useState<{
    viewId: string;
    origin: { x: number; y: number; width: number; height: number };
    ghost: { x: number; y: number };
    pointer: { x: number; y: number };
  } | null>(null);
  const closeSurface = (surface: WorkspaceSurface) =>
    onRequestClose ? onRequestClose(surface) : workspace.close(surface.id);
  const terminalSurfaces = surfaces.filter(isPtySurface);
  const mosaic = presentation.state.mode === 'MOSAIC';
  const activeViewId = presentation.state.activePtyViewId;
  const [layoutMotion, setLayoutMotion] = useState(false);
  const previousMode = useRef(presentation.state.mode);

  useEffect(() => {
    if (previousMode.current === presentation.state.mode) return;
    previousMode.current = presentation.state.mode;
    setLayoutMotion(true);
    const timer = window.setTimeout(() => setLayoutMotion(false), 340);
    return () => window.clearTimeout(timer);
  }, [presentation.state.mode]);

  const windows = terminalSurfaces
    .map((surface) => ({ surface, win: presentation.state.windows[surfaceViewId(surface)] }))
    .filter((item) => Boolean(item.win))
    .sort((a, b) => surfaceViewId(a.surface).localeCompare(surfaceViewId(b.surface)));
  const mosaicDropViewId =
    mosaic && mosaicDrag
      ? mosaicDropTargetViewId(presentation.state, mosaicDrag.viewId, mosaicDrag.pointer)
      : '';

  const splitters = useMemo(() => {
    if (!mosaic) return [];
    const tiles = windows
      .filter(({ win }) => win && !win.minimized && !win.maximized)
      .map(({ win }) => win!);
    return findTileSplitters(tiles);
  }, [mosaic, windows]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const publish = () => {
      const rect = host.getBoundingClientRect();
      presentation.setCanvas({
        x: 8,
        y: 8,
        width: Math.max(320, Math.floor(rect.width - 16)),
        height: Math.max(240, Math.floor(rect.height - 16)),
      });
    };
    publish();
    const observer =
      typeof ResizeObserver !== 'undefined' ? new ResizeObserver(() => publish()) : null;
    observer?.observe(host);
    return () => observer?.disconnect();
  }, [presentation]);

  const startMove = (viewId: string, event: React.PointerEvent) => {
    if (event.button !== 0) return;
    const win = presentation.state.windows[viewId];
    if (!win || win.maximized || win.minimized) return;
    event.preventDefault();
    event.stopPropagation();
    presentation.focus(viewId);
    const startX = event.clientX;
    const startY = event.clientY;
    const origin = { x: win.x, y: win.y, width: win.width, height: win.height };
    let dragged = false;
    const pointInWorkspace = (move: PointerEvent) => {
      const host = hostRef.current;
      if (!host) return { x: move.clientX, y: move.clientY };
      const rect = host.getBoundingClientRect();
      return { x: move.clientX - rect.left, y: move.clientY - rect.top };
    };
    const onMove = (move: PointerEvent) => {
      if (!dragged && Math.abs(move.clientX - startX) + Math.abs(move.clientY - startY) < 4) return;
      dragged = true;
      if (mosaic) {
        setMosaicDrag({
          viewId,
          origin,
          ghost: { x: origin.x + (move.clientX - startX), y: origin.y + (move.clientY - startY) },
          pointer: pointInWorkspace(move),
        });
        return;
      }
      presentation.move(
        viewId,
        origin.x + (move.clientX - startX),
        origin.y + (move.clientY - startY),
      );
    };
    const onUp = (up: PointerEvent) => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
      setMosaicDrag(null);
      if (mosaic && dragged) presentation.commitMove(viewId, origin, pointInWorkspace(up));
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  const startResize = (viewId: string, event: React.PointerEvent) => {
    if (mosaic || event.button !== 0) return;
    const win = presentation.state.windows[viewId];
    if (!win || win.maximized || win.minimized) return;
    event.preventDefault();
    event.stopPropagation();
    presentation.focus(viewId);
    const startX = event.clientX;
    const startY = event.clientY;
    const originW = win.width;
    const originH = win.height;
    const onMove = (move: PointerEvent) => {
      presentation.resize(
        viewId,
        originW + (move.clientX - startX),
        originH + (move.clientY - startY),
      );
    };
    const onUp = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  const startSplitter = (
    splitter: { firstId: string; secondId: string; orientation: 'vertical' | 'horizontal' },
    event: React.PointerEvent,
  ) => {
    if (event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    let last = splitter.orientation === 'vertical' ? event.clientX : event.clientY;
    const onMove = (move: PointerEvent) => {
      const current = splitter.orientation === 'vertical' ? move.clientX : move.clientY;
      const delta = current - last;
      last = current;
      if (delta === 0) return;
      presentation.resizeAdjacent(splitter.firstId, splitter.secondId, splitter.orientation, delta);
    };
    const onUp = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  return (
    <div
      ref={hostRef}
      className="nx-desktop-workspace"
      data-tour="terminals-windows"
      data-presentation={mosaic ? 'mosaic' : 'desktop'}
      data-layout-motion={layoutMotion ? 'true' : undefined}
    >
      {windows.map(({ surface, win }) => {
        if (!win || win.minimized) return null;
        const viewId = surfaceViewId(surface);
        const live = liveChromeFor(liveByViewId, viewId);
        const questionnaire = live.questionnaire;
        const attention = surface.data?.hasAttention === 'true' || questionnaire;
        const unread = surface.data?.unreadAttention === 'true';
        const attentionKind = questionnaire ? 'needs_user' : surface.data?.attentionKind || '';
        const agentName = surfaceIdentity(surface, win.customTitle);
        const heading = ptyWindowHeading({
          customTitle: win.customTitle,
          identity: agentName,
          liveTitle: live.title,
        });
        const statusSuffix = surface.data?.statusSuffix || '';
        const dynamicTitle = live.title ? '' : surface.data?.dynamicTitle || '';
        const providerLabel = surface.data?.providerLabel || '';
        const accent = win.accent || '';
        const icon = win.icon || '';
        const style: React.CSSProperties = {
          ...(win.maximized && !mosaic
            ? { inset: 8, zIndex: win.zIndex }
            : {
                left: win.x,
                top: win.y,
                width: win.width,
                height: win.height,
                zIndex: mosaic ? 1 : mosaicDrag?.viewId === viewId ? 10_000 : win.zIndex,
              }),
          ...(accent ? ({ ['--nx-window-accent' as string]: accent } as React.CSSProperties) : {}),
        };
        return (
          <section
            key={viewId}
            className="nx-desktop-window"
            data-view-id={viewId}
            data-maximized={win.maximized ? 'true' : 'false'}
            data-mosaic={mosaic ? 'true' : undefined}
            data-active={activeViewId === viewId ? 'true' : undefined}
            data-dragging={mosaicDrag?.viewId === viewId ? 'true' : undefined}
            data-drop-target={mosaicDropViewId === viewId ? 'true' : undefined}
            data-accented={accent ? 'true' : undefined}
            data-attention={attention ? 'true' : undefined}
            data-unread={unread ? 'true' : undefined}
            data-attention-kind={attentionKind || undefined}
            data-questionnaire={questionnaire ? 'true' : undefined}
            data-icon={icon || undefined}
            style={style}
            onPointerDownCapture={() => {
              presentation.focus(viewId);
              if (chromeMenuViewId && chromeMenuViewId !== viewId) setChromeMenuViewId(null);
            }}
          >
            <header
              className="nx-desktop-window__titlebar"
              data-accented={accent ? 'true' : undefined}
              onPointerDown={(event) => {
                if (event.button === 0) setChromeMenuViewId(null);
                startMove(viewId, event);
              }}
              onContextMenu={(event) => {
                presentation.focus(viewId);
                setChromeMenuViewId(null);
                setWindowMenu({ viewId, surface, point: contextMenuFromEvent(event) });
              }}
            >
              <div
                className="nx-desktop-window__title"
                title={[
                  heading.heading,
                  heading.identityHint || providerLabel,
                  dynamicTitle || statusSuffix,
                ]
                  .filter(Boolean)
                  .join(' · ')}
              >
                <div className="nx-desktop-window__title-primary">
                  {icon && (
                    <span className="nx-desktop-window__icon" aria-hidden="true">
                      {icon}
                    </span>
                  )}
                  {(attention || unread) && (
                    <span
                      className="nx-desktop-window__attention-dot"
                      data-kind={attentionKind || 'needs_user'}
                      data-unread={unread ? 'true' : undefined}
                      role="img"
                      aria-label={t(
                        unread
                          ? 'workspace.unreadAttention'
                          : questionnaire
                            ? 'workspace.questionnaire'
                            : 'workspace.attention',
                        unread ? 'Unread attention' : questionnaire ? 'Questionnaire' : 'Attention',
                      )}
                    />
                  )}
                  <strong>{heading.heading}</strong>
                </div>
                <div className="nx-desktop-window__title-meta" aria-hidden="true">
                  {heading.identityHint && (
                    <small className="nx-desktop-window__provider">{heading.identityHint}</small>
                  )}
                  {!heading.identityHint && providerLabel && (
                    <small className="nx-desktop-window__provider">{providerLabel}</small>
                  )}
                  {dynamicTitle && (
                    <small className="nx-desktop-window__dynamic">{dynamicTitle}</small>
                  )}
                  {!dynamicTitle && statusSuffix && (
                    <small
                      className="nx-desktop-window__status"
                      data-kind={attentionKind || undefined}
                    >
                      {statusSuffix}
                    </small>
                  )}
                </div>
              </div>
              <div
                className="nx-desktop-window__actions"
                onPointerDown={(event) => event.stopPropagation()}
              >
                <WindowChromeMenu
                  viewId={viewId}
                  open={chromeMenuViewId === viewId}
                  position={chromeMenuViewId === viewId ? chromeMenuPoint : null}
                  onOpenChange={(next) => {
                    setChromeMenuPoint(null);
                    setChromeMenuViewId(next ? viewId : null);
                  }}
                  customTitle={win.customTitle || ''}
                  accent={win.accent || ''}
                  icon={win.icon || ''}
                  onPatch={(chrome) => presentation.patchChrome(viewId, chrome)}
                />
                <IconButton
                  label={t('workspace.minimizeAgent', {
                    agent: agentName,
                    defaultValue: `Minimizar ${agentName}`,
                  })}
                  onClick={() => presentation.minimize(viewId)}
                >
                  <Minus size={13} />
                </IconButton>
                {!mosaic && (
                  <IconButton
                    label={
                      win.maximized
                        ? t('workspace.restoreAgent', {
                            agent: agentName,
                            defaultValue: `Restaurar ${agentName}`,
                          })
                        : t('workspace.maximizeAgent', {
                            agent: agentName,
                            defaultValue: `Maximizar ${agentName}`,
                          })
                    }
                    onClick={() => presentation.maximize(viewId)}
                  >
                    {win.maximized ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
                  </IconButton>
                )}
                {surface.closable !== false && (
                  <IconButton
                    label={t('workspace.closeAgent', {
                      agent: agentName,
                      defaultValue: `Fechar ${agentName}`,
                    })}
                    onClick={() => closeSurface(surface)}
                  >
                    <X size={13} />
                  </IconButton>
                )}
              </div>
            </header>
            <div className="nx-desktop-window__body">{renderSurface(surface)}</div>
            {!mosaic && !win.maximized && (
              <div
                className="nx-desktop-window__resize"
                aria-hidden="true"
                onPointerDown={(event) => startResize(viewId, event)}
              />
            )}
          </section>
        );
      })}
      {mosaic && mosaicDrag && (
        <>
          <div
            className="nx-desktop-mosaic-slot"
            data-kind="origin"
            aria-hidden="true"
            style={{
              left: mosaicDrag.origin.x,
              top: mosaicDrag.origin.y,
              width: mosaicDrag.origin.width,
              height: mosaicDrag.origin.height,
            }}
          />
          {mosaicDropViewId && presentation.state.windows[mosaicDropViewId] && (
            <div
              className="nx-desktop-mosaic-slot"
              data-kind="drop"
              aria-hidden="true"
              style={{
                left: presentation.state.windows[mosaicDropViewId].x,
                top: presentation.state.windows[mosaicDropViewId].y,
                width: presentation.state.windows[mosaicDropViewId].width,
                height: presentation.state.windows[mosaicDropViewId].height,
              }}
            />
          )}
          <div
            className="nx-desktop-mosaic-ghost"
            aria-hidden="true"
            style={{
              left: mosaicDrag.ghost.x,
              top: mosaicDrag.ghost.y,
              width: mosaicDrag.origin.width,
              height: mosaicDrag.origin.height,
            }}
          />
        </>
      )}
      {mosaic &&
        splitters.map((splitter) => {
          const style: React.CSSProperties =
            splitter.orientation === 'vertical'
              ? {
                  left: splitter.x - 3,
                  top: splitter.y,
                  width: 6,
                  height: splitter.length,
                  cursor: 'col-resize',
                }
              : {
                  left: splitter.x,
                  top: splitter.y - 3,
                  width: splitter.length,
                  height: 6,
                  cursor: 'row-resize',
                };
          return (
            <div
              key={splitter.id}
              className="nx-desktop-mosaic-splitter"
              data-orientation={splitter.orientation}
              style={style}
              onPointerDown={(event) => startSplitter(splitter, event)}
            />
          );
        })}
      {mosaic && windows.length > 0 && windows.every(({ win }) => win?.minimized) && (
        <div className="nx-mosaic-all-minimized">
          <Minus size={18} />
          <strong>{t('workspace.minimizedShelf')}</strong>
          <p>{t('workspace.mosaicAllMinimized')}</p>
        </div>
      )}
      <div
        className="nx-desktop-dock"
        data-kind={mosaic ? 'shelf' : 'float'}
        hidden={windows.every(({ win }) => !win?.minimized)}
        aria-label={t('workspace.minimizedShelf')}
      >
        {mosaic && <span className="nx-desktop-dock__label">{t('workspace.minimizedShelf')}</span>}
        {windows
          .filter(({ win }) => win?.minimized)
          .map(({ surface, win }) => {
            const viewId = surfaceViewId(surface);
            const unread = surface.data?.unreadAttention === 'true';
            const live = liveChromeFor(liveByViewId, viewId);
            const label = ptyWindowHeading({
              customTitle: win?.customTitle,
              identity: surfaceIdentity(surface, win?.customTitle),
              liveTitle: live.title,
            }).heading;
            return (
              <button
                type="button"
                key={viewId}
                title={t('workspace.restoreNamed', { name: label })}
                data-unread={unread ? 'true' : undefined}
                onClick={() => {
                  presentation.minimize(viewId);
                  presentation.focus(viewId);
                }}
                onContextMenu={(event) => {
                  setWindowMenu({ viewId, surface, point: contextMenuFromEvent(event) });
                }}
              >
                {unread && <span className="nx-desktop-window__attention-dot" data-unread="true" />}
                {win?.icon ? <span aria-hidden="true">{win.icon}</span> : null}
                {label}
              </button>
            );
          })}
      </div>
      <ContextMenu
        open={windowMenu?.point ?? null}
        onClose={() => setWindowMenu(null)}
        label={t('workspace.menu')}
        items={(() => {
          if (!windowMenu) return [];
          const win = presentation.state.windows[windowMenu.viewId];
          const items: ContextMenuItem[] = [
            {
              type: 'item',
              id: 'customize',
              label: t('workspace.windowCustomize'),
              icon: <Paintbrush size={13} />,
              onSelect: () => {
                setChromeMenuPoint(windowMenu.point);
                setChromeMenuViewId(windowMenu.viewId);
              },
            },
          ];
          if (win?.minimized) {
            items.push({
              type: 'item',
              id: 'restore',
              label: t('workspace.windowRestore'),
              icon: <Minimize2 size={13} />,
              onSelect: () => {
                presentation.minimize(windowMenu.viewId);
                presentation.focus(windowMenu.viewId);
              },
            });
          } else {
            items.push({
              type: 'item',
              id: 'min',
              label: t('workspace.windowMinimize'),
              icon: <Minus size={13} />,
              onSelect: () => presentation.minimize(windowMenu.viewId),
            });
            if (!mosaic) {
              items.push({
                type: 'item',
                id: 'max',
                label: win?.maximized
                  ? t('workspace.windowRestore')
                  : t('workspace.windowMaximize'),
                icon: win?.maximized ? <Minimize2 size={13} /> : <Maximize2 size={13} />,
                onSelect: () => presentation.maximize(windowMenu.viewId),
              });
            }
          }
          if (windowMenu.surface.closable !== false) {
            items.push({ type: 'separator', id: 'sep-close' });
            items.push({
              type: 'item',
              id: 'close',
              label: t('workspace.windowClose'),
              danger: true,
              icon: <X size={13} />,
              onSelect: () => closeSurface(windowMenu.surface),
            });
          }
          return items;
        })()}
      />
    </div>
  );
};
