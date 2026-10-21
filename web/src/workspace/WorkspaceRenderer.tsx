import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  AppWindow,
  Gauge,
  History,
  Home,
  Layers,
  LayoutGrid,
  Columns2,
  Rows2,
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
  DropdownMenu,
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
import { resolveTerminalSplitViews } from './presentation';
import splitStyles from './TerminalSplit.module.scss';

const WINDOW_ACCENTS = [
  'var(--nx-info)',
  'var(--nx-success)',
  'var(--nx-warning)',
  'var(--nx-accent)',
  'var(--nx-accent-text)',
  'var(--nx-danger)',
];
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
  allowAiTitle: boolean;
  accent: string;
  icon: string;
  onPatch: (chrome: {
    customTitle?: string;
    allowAiTitle?: boolean;
    accent?: string;
    icon?: string;
  }) => void;
  position?: ContextMenuPoint | null;
}> = ({ open, onOpenChange, customTitle, allowAiTitle, accent, icon, onPatch, position }) => {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement>(null);
  const [draft, setDraft] = useState(customTitle);
  const [menuPos, setMenuPos] = useState<{ top: number; right?: number; left?: number } | null>(
    null,
  );

  useEffect(() => {
    // Escape discards the draft; reset the suppression when the menu reopens.
    if (open) skipBlurCommit.current = false;
    setDraft(customTitle);
  }, [customTitle, open]);

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

  const skipBlurCommit = useRef(false);
  const commitTitle = () => {
    if (skipBlurCommit.current) return;
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
              {t('workspace.customTitle', 'Nome personalizado')}
              <input
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                onBlur={commitTitle}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    commitTitle();
                    onOpenChange(false);
                  } else if (event.key === 'Escape') {
                    event.stopPropagation();
                    skipBlurCommit.current = true;
                    setDraft(customTitle);
                    onOpenChange(false);
                  }
                }}
                placeholder={t('workspace.windowTitlePlaceholder', 'Identidade da janela')}
                autoFocus
              />
            </label>
            <label className="nx-desktop-window__ai-title-toggle">
              <input
                type="checkbox"
                checked={allowAiTitle}
                onChange={(event) => onPatch({ allowAiTitle: event.target.checked })}
              />
              <span>{t('workspace.allowAiTitle', 'Permitir que a IA atualize o título')}</span>
            </label>
            <p className="nx-desktop-window__ai-title-help">
              {t(
                'workspace.allowAiTitleHelp',
                'O nome personalizado permanece salvo. A IA pode atualizar o título principal quando esta opção estiver ativa.',
              )}
            </p>
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

  return (
    <div className="nx-arrange-menu">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            className="nx-arrange-menu__trigger"
            title={t('workspace.arrangeWindows', 'Arranjar janelas')}
            aria-label={t('workspace.arrangeWindows', 'Arranjar janelas')}
          >
            <LayoutGrid size={12} />
            <span>{t('workspace.arrange', 'Arranjar')}</span>
          </button>
        </DropdownMenu.Trigger>

        <DropdownMenu.Portal>
          <DropdownMenu.Content
            className="nx-arrange-menu__panel"
            aria-label={t('workspace.arrangeWindows', 'Arranjar janelas')}
            align="end"
            sideOffset={6}
            collisionPadding={8}
          >
            <DropdownMenu.RadioGroup
              value={activePreset}
              onValueChange={(value) => onSelect(value as ArrangeMenuPreset)}
            >
              {ARRANGE_MENU_PRESETS.map((preset) => (
                <DropdownMenu.RadioItem
                  key={preset.id}
                  value={preset.id}
                  className="nx-arrange-menu__item"
                  data-active={activePreset === preset.id ? 'true' : undefined}
                >
                  <span>{preset.label}</span>
                  <small>{preset.hint}</small>
                </DropdownMenu.RadioItem>
              ))}
            </DropdownMenu.RadioGroup>

            {onReset && (
              <>
                <DropdownMenu.Separator className="nx-arrange-menu__separator" />
                <DropdownMenu.Item className="nx-arrange-menu__item" onSelect={() => onReset()}>
                  <span className="nx-arrange-menu__reset-label">
                    <RotateCcw size={12} />
                    <span>{t('workspace.resetLayout', 'Redefinir layout')}</span>
                  </span>
                  <small>{t('workspace.restoreDefault', 'Restaurar padrão')}</small>
                </DropdownMenu.Item>
              </>
            )}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
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
  // Surface isolation contract: inactive product surfaces unmount. Their React
  // portals unmount with them, so no orphan dialog/backdrop/focus-trap can leak.
  // TerminalsHost is the deliberate keep-alive exception (PTY/runtime continuity).
  const terminalsSurface = productTabs.find((tab) => tab.type === 'terminals');
  const terminalsActive = activeProduct?.type === 'terminals';
  useEffect(() => {
    setTabMenu(null);
    // If focus was inside the panel that just unmounted/hid, move it to the
    // newly active tab instead of stranding it on <body>.
    const active = document.activeElement;
    if (active && active !== document.body && !document.contains(active)) {
      document.getElementById(`nx-tab-${activeId}`)?.focus();
    } else if (active instanceof Element && active.closest?.('.nx-workspace-panel[hidden]')) {
      document.getElementById(`nx-tab-${activeId}`)?.focus();
    }
  }, [activeId]);
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
        <div
          className="nx-workspace-tabs"
          role="tablist"
          aria-label={t('shell.openSurfaces')}
          onKeyDown={(event) => {
            // Manual activation: arrows move focus only, Enter/Space activates.
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
          {productTabs.map((surface) => {
            const TabIcon = PRODUCT_TAB_ICONS[surface.type] || Layers;
            const pinned = isPinnedProductTab(surface);
            return (
              <div
                key={surface.id}
                className="nx-workspace-tab__item"
                data-active={activeId === surface.id ? 'true' : 'false'}
              >
                <button
                  draggable={!pinned}
                  id={`nx-tab-${surface.id}`}
                  aria-controls={activeId === surface.id ? `nx-panel-${surface.id}` : undefined}
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
                  onKeyDown={(event) => {
                    if (event.key === 'Delete' && surface.closable !== false) {
                      event.preventDefault();
                      closeSurface(surface);
                    }
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
                </button>
                {surface.closable !== false && (
                  // Decorative close affordance: role=tablist may own only tabs,
                  // so the × stays out of the accessibility tree. Keyboard users
                  // close a tab with Delete or the tab context menu.
                  <span
                    className="nx-workspace-tab__close"
                    aria-hidden="true"
                    onClick={(event) => {
                      event.stopPropagation();
                      closeSurface(surface);
                    }}
                  >
                    <X size={11} />
                  </span>
                )}
              </div>
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
        {terminalsSurface && (
          <div
            role="tabpanel"
            id={`nx-panel-${terminalsSurface.id}`}
            aria-labelledby={`nx-tab-${terminalsSurface.id}`}
            hidden={!terminalsActive || undefined}
            inert={!terminalsActive || undefined}
            data-active={terminalsActive ? 'true' : 'false'}
            className="nx-workspace-panel nx-workspace-panel--terminals"
          >
            <TerminalsHost
              ptySurfaces={ptyTabs}
              renderSurface={renderSurface}
              onRequestClose={onRequestClose}
              createActions={createActions}
              active={terminalsActive}
            />
          </div>
        )}
        {activeProduct && activeProduct.type !== 'terminals' && (
          <div
            key={activeProduct.id}
            role="tabpanel"
            id={`nx-panel-${activeProduct.id}`}
            aria-labelledby={`nx-tab-${activeProduct.id}`}
            data-active="true"
            className="nx-workspace-panel"
          >
            {renderSurface(activeProduct)}
          </div>
        )}
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
  const compactSplit = useCompactViewport();
  const effectiveSplitOrientation = compactSplit ? 'horizontal' : presentation.state.splitOrientation;
  const splitMode = presentation.state.mode === 'SPLIT' && ptySurfaces.length > 1;
  const splitViews = resolveTerminalSplitViews(
    presentation.state,
    ptySurfaces.map((surface) => surfaceViewId(surface)),
  );
  const splitRef = useRef<HTMLDivElement>(null);
  const beginSplitResize = (event: React.PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    event.preventDefault();
    const target = event.currentTarget;
    target.setPointerCapture(event.pointerId);
  };
  const updateSplitResize = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!event.currentTarget.hasPointerCapture(event.pointerId)) return;
    const rect = splitRef.current?.getBoundingClientRect();
    if (!rect || !rect.width || !rect.height) return;
    const vertical = effectiveSplitOrientation === 'vertical';
    const fraction = vertical
      ? (event.clientX - rect.left) / rect.width
      : (event.clientY - rect.top) / rect.height;
    presentation.setSplitRatio(fraction);
  };
  const minimizedPtys = ptySurfaces.filter(
    (surface) => presentation.state.windows[surfaceViewId(surface)]?.minimized,
  );
  // Surface switch away from terminals: close ownerless menu portals living
  // in document.body (inert on the panel cannot reach them).
  const hostWasActive = useRef(active);
  useEffect(() => {
    if (hostWasActive.current && !active) {
      setTabMenu(null);
      setChromeMenuViewId(null);
      setChromeMenuPoint(null);
    }
    hostWasActive.current = active;
  }, [active]);

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
    if (splitMode && viewId !== splitViews.primary && viewId !== splitViews.secondary) {
      presentation.setSplitPrimary(viewId);
    }
    presentation.setActivePty(viewId);
    if (windowed) {
      presentation.focus(viewId);
    }
  };
  // Restore a minimized PTY in one dispatch, then nudge layout observers (the
  // xterm was 0x0 while hidden). Focus moves to the restored PTY tab: the
  // restore chip unmounts, so focus must not fall back to body.
  const restoreView = (viewId: string) => {
    presentation.restore(viewId);
    const surface = ptySurfaces.find((item) => surfaceViewId(item) === viewId);
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        window.dispatchEvent(new Event('resize'));
        if (surface) document.getElementById(`nx-pty-tab-${surface.id}`)?.focus();
      });
    });
  };

  return (
    <div className="nx-terminals-host" data-active={active ? 'true' : 'false'}>
      <header className="nx-terminals-host__chrome">
        <div className="nx-terminals-host__title">
          <TerminalSquare size={14} />
          <h1 className="nx-sr-only">{t('nav.terminals')}</h1>
          <strong aria-hidden="true">
            {t('nav.terminals')} · {t('nav.ptyCount', { count: ptySurfaces.length })}
          </strong>
        </div>
        {windowed && minimizedPtys.length > 0 ? (
          <div
            className="nx-minimized-chips"
            role="group"
            aria-label={t('workspace.minimizedShelf')}
          >
            <span className="nx-minimized-chips__label">{t('workspace.minimizedShelf')}</span>
            {minimizedPtys.map((surface) => {
              const viewId = surfaceViewId(surface);
              const win = presentation.state.windows[viewId];
              const live = liveChromeFor(liveByViewId, viewId);
              const label = ptyWindowHeading({
                customTitle: win?.customTitle,
                identity: surfaceIdentity(surface, win?.customTitle),
                liveTitle: live.title || surface.data?.dynamicTitle,
                allowAiTitle: win?.allowAiTitle,
              }).heading;
              return (
                <button
                  key={viewId}
                  type="button"
                  id={`nx-restore-${viewId}`}
                  title={t('workspace.restoreNamed', { name: label })}
                  aria-label={t('workspace.restoreNamed', { name: label })}
                  data-unread={surface.data?.unreadAttention === 'true' ? 'true' : undefined}
                  onClick={() => restoreView(viewId)}
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
              disabled={ptySurfaces.length < 2}
              data-active={presentation.state.mode === 'SPLIT' ? 'true' : 'false'}
              onClick={() => presentation.setMode('SPLIT')}
              title={t('workspace.splitTerminals')}
              aria-label={t('workspace.splitTerminals')}
            >
              <Columns2 size={11} />
              <span>{t('workspace.splitShort')}</span>
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
          {splitMode && (
            <div className={splitStyles.orientation} role="group" aria-label={t('workspace.splitOrientation')}>
              <button type="button" aria-pressed={effectiveSplitOrientation === 'vertical'}
                disabled={compactSplit}
                title={t('workspace.splitSideBySide')} onClick={() => presentation.setSplitOrientation('vertical')}>
                <Columns2 size={15} />
              </button>
              <button type="button" aria-pressed={effectiveSplitOrientation === 'horizontal'}
                title={t('workspace.splitStacked')} onClick={() => presentation.setSplitOrientation('horizontal')}>
                <Rows2 size={15} />
              </button>
            </div>
          )}
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
          <div className="nx-terminals-empty" role="status">
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
            active={active}
          />
        ) : (
          <div className="nx-terminals-inner-tabs">
            <div
              className="nx-workspace-tabs nx-pty-tabs"
              role="tablist"
              aria-label={t('workspace.ptyTabs', 'Abas PTY')}
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
                      : (current + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) %
                        tabs.length;
                tabs[next]?.focus();
                tabs[next]?.click();
              }}
            >
              {ptySurfaces.map((surface) => {
                const viewId = surfaceViewId(surface);
                const win = presentation.state.windows[viewId];
                const live = liveChromeFor(liveByViewId, viewId);
                const tabLabel = ptyWindowHeading({
                  customTitle: win?.customTitle,
                  identity: surfaceIdentity(surface, win?.customTitle),
                  liveTitle: live.title || surface.data?.dynamicTitle,
                  allowAiTitle: win?.allowAiTitle,
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
                      id={`nx-pty-tab-${surface.id}`}
                      aria-controls={`nx-pty-panel-${surface.id}`}
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
                        if (!splitMode) focusPty(surface);
                        setChromeMenuViewId(null);
                        setTabMenu({ surface, point: contextMenuFromEvent(event) });
                      }}
                      onKeyDown={(event) => {
                        if (
                          event.key === 'Delete' &&
                          surface.closable !== false &&
                          onRequestClose
                        ) {
                          event.preventDefault();
                          onRequestClose(surface);
                        }
                      }}
                    >
                      {icon ? (
                        <span className="nx-pty-tab__glyph" aria-hidden="true">
                          {icon}
                        </span>
                      ) : null}
                      <span className="nx-workspace-tab__label">{tabLabel}</span>
                    </button>
                    {surface.closable !== false && onRequestClose && (
                      // Decorative close affordance, same contract as the main
                      // strip: aria-hidden ×, Delete key or context menu close.
                      <span
                        className="nx-workspace-tab__close"
                        aria-hidden="true"
                        onClick={(event) => {
                          event.stopPropagation();
                          onRequestClose(surface);
                        }}
                      >
                        <X size={11} />
                      </span>
                    )}
                    <WindowChromeMenu
                      viewId={viewId}
                      open={chromeMenuViewId === viewId}
                      position={chromeMenuViewId === viewId ? chromeMenuPoint : null}
                      onOpenChange={(next) => {
                        setChromeMenuPoint(null);
                        setChromeMenuViewId(next ? viewId : null);
                      }}
                      customTitle={win?.customTitle || ''}
                      allowAiTitle={win?.allowAiTitle !== false}
                      accent={accent}
                      icon={icon}
                      onPatch={(chrome) => presentation.patchChrome(viewId, chrome)}
                    />
                  </div>
                );
              })}
            </div>
            <div
              ref={splitRef}
              className={splitMode ? `nx-terminals-inner-panels ${splitStyles.splitPanels}` : 'nx-terminals-inner-panels'}
              data-orientation={splitMode ? effectiveSplitOrientation : undefined}
              style={splitMode ? ({ '--nx-terminal-split-ratio': `${Math.round(presentation.state.splitRatio * 100)}%` } as React.CSSProperties) : undefined}
            >
              {ptySurfaces.map((surface) => {
                const ptyActive = focusedPtyId === surface.id;
                const isSplitPrimary = splitMode && surfaceViewId(surface) === splitViews.primary;
                const isSplitSecondary = splitMode && surfaceViewId(surface) === splitViews.secondary;
                const visible = splitMode ? isSplitPrimary || isSplitSecondary : ptyActive;
                return (
                  <div
                    key={surface.id}
                    role="tabpanel"
                    id={`nx-pty-panel-${surface.id}`}
                    aria-labelledby={`nx-pty-tab-${surface.id}`}
                    hidden={!visible || undefined}
                    inert={!visible || undefined}
                    className={splitMode ? `nx-workspace-panel ${splitStyles.splitPane}` : 'nx-workspace-panel'}
                    style={splitMode && visible ? { order: isSplitPrimary ? 0 : 2 } : undefined}
                    data-active={visible ? 'true' : 'false'}
                    onPointerDownCapture={() => {
                      if (splitMode && visible && presentation.state.activePtyViewId !== surfaceViewId(surface)) {
                        presentation.setActivePty(surfaceViewId(surface));
                      }
                    }}
                  >
                    {renderSurface(surface)}
                  </div>
                );
              })}
              {splitMode && splitViews.secondary && (
                <div
                  className={splitStyles.divider}
                  role="separator"
                  aria-orientation={effectiveSplitOrientation}
                  aria-label={t('workspace.resizeSplit')}
                  aria-valuemin={20}
                  aria-valuemax={80}
                  aria-valuenow={Math.round(presentation.state.splitRatio * 100)}
                  tabIndex={0}
                  style={{ order: 1 }}
                  onPointerDown={beginSplitResize}
                  onPointerMove={updateSplitResize}
                  onPointerUp={() => window.dispatchEvent(new Event('resize'))}
                  onKeyDown={(event) => {
                    const orientation = effectiveSplitOrientation;
                    const increment = orientation === 'vertical'
                      ? (event.key === 'ArrowRight' ? 0.05 : event.key === 'ArrowLeft' ? -0.05 : 0)
                      : (event.key === 'ArrowDown' ? 0.05 : event.key === 'ArrowUp' ? -0.05 : 0);
                    if (!increment) return;
                    event.preventDefault();
                    presentation.setSplitRatio(presentation.state.splitRatio + increment);
                  }}
                />
              )}
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
                      ...(splitMode && surfaceViewId(tabMenu.surface) !== splitViews.primary
                        ? ([{
                            type: 'item',
                            id: 'split-secondary',
                            label: t('workspace.showInSecondPane'),
                            icon: <Columns2 size={13} />,
                            onSelect: () => presentation.setSplitSecondary(surfaceViewId(tabMenu.surface)),
                          }] satisfies ContextMenuItem[])
                        : []),
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
  active: boolean;
}> = ({ surfaces, renderSurface, onRequestClose, active }) => {
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
  // Body portals owned here must not outlive the terminals panel: when it
  // hides, any open window/chrome menu would overlay the new surface.
  useEffect(() => {
    if (!active) {
      setChromeMenuViewId(null);
      setChromeMenuPoint(null);
      setWindowMenu(null);
    }
  }, [active]);
  const terminalSurfaces = surfaces.filter(isPtySurface);
  const mosaic = presentation.state.mode === 'MOSAIC';
  // Minimize then move keyboard focus to the shelf restore control — but only
  // when focus was inside the minimized window. Double rAF waits for React to
  // commit the hidden subtree and the shelf chip before focusing.
  const minimizeView = (viewId: string) => {
    const hadFocus =
      document.activeElement instanceof Element &&
      Boolean(document.activeElement.closest?.(`[data-view-id="${viewId}"]`));
    setChromeMenuViewId(null);
    setChromeMenuPoint(null);
    setWindowMenu(null);
    presentation.minimize(viewId);
    if (!hadFocus) return;
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        document.getElementById(`nx-restore-${viewId}`)?.focus();
      });
    });
  };
  // Single-dispatch restore + layout nudge for the 0x0-hidden xterm, then move
  // focus into the restored window (the invoking dock chip may unmount).
  const restoreDesktopView = (viewId: string) => {
    presentation.restore(viewId);
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        window.dispatchEvent(new Event('resize'));
        const winEl = document.getElementById(`nx-window-${viewId}`);
        const xtermTextarea = winEl?.querySelector<HTMLTextAreaElement>('.xterm-helper-textarea');
        if (xtermTextarea) {
          xtermTextarea.focus();
        } else {
          winEl?.focus();
        }
      });
    });
  };
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
      // While the terminals panel is hidden the rect is zero; publishing a
      // fallback canvas would clobber the desktop layout for no reason.
      if (rect.width <= 0 || rect.height <= 0) return;
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
        if (!win) return null;
        // Minimized windows stay mounted but hidden (display:none keeps the
        // xterm instance, WebSocket and scrollback alive): minimize must not
        // unmount the PTY. Backend runtime survives regardless; this preserves
        // output continuity and fit state on restore.
        const minimized = win.minimized === true;
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
          liveTitle: live.title || surface.data?.dynamicTitle,
          allowAiTitle: win.allowAiTitle,
        });
        const statusSuffix = surface.data?.statusSuffix || '';
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
            id={`nx-window-${viewId}`}
            tabIndex={-1}
            className="nx-desktop-window"
            data-view-id={viewId}
            hidden={minimized || undefined}
            inert={minimized || undefined}
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
            onPointerDownCapture={(event) => {
              presentation.focus(viewId);
              if (chromeMenuViewId && chromeMenuViewId !== viewId) setChromeMenuViewId(null);
              const target = event.target as HTMLElement | null;
              const isInteractive = target?.closest(
                'button, input, select, textarea, [role="button"], [role="menuitem"], .nx-desktop-window__resize',
              );
              if (!isInteractive) {
                const xtermTextarea =
                  event.currentTarget.querySelector<HTMLTextAreaElement>('.xterm-helper-textarea');
                if (xtermTextarea) {
                  window.requestAnimationFrame(() => xtermTextarea.focus());
                }
              }
            }}
            onFocus={(event) => {
              if (event.target === event.currentTarget) {
                const xtermTextarea =
                  event.currentTarget.querySelector<HTMLTextAreaElement>('.xterm-helper-textarea');
                if (xtermTextarea) {
                  xtermTextarea.focus();
                }
              }
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
                title={[heading.heading, heading.identityHint || providerLabel, statusSuffix]
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
                  {statusSuffix && (
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
                  allowAiTitle={win.allowAiTitle !== false}
                  accent={win.accent || ''}
                  icon={win.icon || ''}
                  onPatch={(chrome) => presentation.patchChrome(viewId, chrome)}
                />
                <IconButton
                  label={t('workspace.minimizeAgent', { agent: agentName })}
                  onClick={() => minimizeView(viewId)}
                >
                  <Minus size={13} />
                </IconButton>
                {!mosaic && (
                  <IconButton
                    label={
                      win.maximized
                        ? t('workspace.restoreAgent', { agent: agentName })
                        : t('workspace.maximizeAgent', { agent: agentName })
                    }
                    onClick={() => presentation.maximize(viewId)}
                  >
                    {win.maximized ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
                  </IconButton>
                )}
                {surface.closable !== false && (
                  <IconButton
                    label={t('workspace.closeAgent', { agent: agentName })}
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
        <div className="nx-mosaic-all-minimized" role="status">
          <Minus size={18} />
          <strong>{t('workspace.minimizedShelf')}</strong>
          <p>{t('workspace.mosaicAllMinimized')}</p>
        </div>
      )}
      <div
        className="nx-desktop-dock"
        role="group"
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
              liveTitle: live.title || surface.data?.dynamicTitle,
              allowAiTitle: win?.allowAiTitle,
            }).heading;
            return (
              <button
                type="button"
                key={viewId}
                id={`nx-restore-dock-${viewId}`}
                title={t('workspace.restoreNamed', { name: label })}
                aria-label={t('workspace.restoreNamed', { name: label })}
                data-unread={unread ? 'true' : undefined}
                onClick={() => restoreDesktopView(viewId)}
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
                restoreDesktopView(windowMenu.viewId);
              },
            });
          } else {
            items.push({
              type: 'item',
              id: 'min',
              label: t('workspace.windowMinimize'),
              icon: <Minus size={13} />,
              onSelect: () => minimizeView(windowMenu.viewId),
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
