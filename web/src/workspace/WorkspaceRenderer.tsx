import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Columns2, ExternalLink, GripVertical, Maximize2, Minimize2, Rows2, X } from 'lucide-react';
import { IconButton } from '../design-system';
import { findStackContaining, isSurfaceMatch, listStacks, type WorkspaceNode, type WorkspaceSplit, type WorkspaceStack, type WorkspaceSurface } from './model';
import { useWorkspace } from './WorkspaceProvider';
import { useTranslation } from 'react-i18next';
import { DesktopWorkspace } from './DesktopWorkspace';

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
  popoutSurface?: (surface: WorkspaceSurface) => void;
}> = ({ renderSurface, popoutSurface }) => {
  const workspace = useWorkspace();
  const compact = useCompactViewport();
  const maximized = workspace.model.maximizedSurfaceId;
  const maximizedStack = maximized ? findStackContaining(workspace.model.root, maximized) : null;

  const presentationSwitch = <div className="nx-workspace-presentation" role="group" aria-label="Workspace presentation">
    <button type="button" data-active={workspace.presentation.mode === 'TABS' ? 'true' : 'false'} onClick={() => workspace.setPresentationMode('TABS')}>Abas</button>
    <button type="button" data-active={workspace.presentation.mode === 'DESKTOP' ? 'true' : 'false'} onClick={() => workspace.setPresentationMode('DESKTOP')}>Janelas</button>
  </div>;

  if (workspace.presentation.mode === 'DESKTOP' && !maximized) {
    return <div className="nx-workspace nx-workspace--desktop" data-tour="workspace">{presentationSwitch}<DesktopWorkspace renderSurface={renderSurface} /></div>;
  }

  if (maximized && maximizedStack) {
    return <div className="nx-workspace nx-workspace--maximized" data-tour="workspace">{presentationSwitch}<WorkspaceStackView stack={{ ...maximizedStack, activeId: maximized }} renderSurface={renderSurface} popoutSurface={popoutSurface} /></div>;
  }

  if (compact) {
    const stacks = listStacks(workspace.model.root);
    const activeStack = stacks.find((stack) => stack.activeId) ?? stacks[0];
    return <div className="nx-workspace nx-workspace--compact" data-tour="workspace">{presentationSwitch}{activeStack && <WorkspaceStackView stack={activeStack} renderSurface={renderSurface} popoutSurface={popoutSurface} />}</div>;
  }

  return <div className="nx-workspace" data-tour="workspace">{presentationSwitch}<NodeView node={workspace.model.root} renderSurface={renderSurface} popoutSurface={popoutSurface} /></div>;
};

const NodeView: React.FC<{ node: WorkspaceNode; renderSurface: (surface: WorkspaceSurface) => React.ReactNode; popoutSurface?: (surface: WorkspaceSurface) => void }> = ({ node, renderSurface, popoutSurface }) => {
  if (node.kind === 'stack') return <WorkspaceStackView stack={node} renderSurface={renderSurface} popoutSurface={popoutSurface} />;
  return <WorkspaceSplitView split={node} renderSurface={renderSurface} popoutSurface={popoutSurface} />;
};

const WorkspaceSplitView: React.FC<{ split: WorkspaceSplit; renderSurface: (surface: WorkspaceSurface) => React.ReactNode; popoutSurface?: (surface: WorkspaceSurface) => void }> = ({ split, renderSurface, popoutSurface }) => {
  const { resize } = useWorkspace();
  const containerRef = useRef<HTMLDivElement>(null);
  const horizontal = split.direction === 'horizontal';
  const startDrag = (event: React.PointerEvent<HTMLDivElement>) => {
    event.currentTarget.setPointerCapture(event.pointerId);
    const container = containerRef.current;
    if (!container) return;
    const rect = container.getBoundingClientRect();
    const onMove = (move: PointerEvent) => {
      const ratio = horizontal ? (move.clientX - rect.left) / rect.width : (move.clientY - rect.top) / rect.height;
      resize(split.id, ratio);
    };
    const onUp = () => { window.removeEventListener('pointermove', onMove); window.removeEventListener('pointerup', onUp); };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp, { once: true });
  };
  const keyboardResize = (event: React.KeyboardEvent<HTMLDivElement>) => {
    const delta = event.shiftKey ? 0.1 : 0.03;
    if ((horizontal && event.key === 'ArrowLeft') || (!horizontal && event.key === 'ArrowUp')) { event.preventDefault(); resize(split.id, split.ratio - delta); }
    if ((horizontal && event.key === 'ArrowRight') || (!horizontal && event.key === 'ArrowDown')) { event.preventDefault(); resize(split.id, split.ratio + delta); }
  };
  return <div ref={containerRef} className="nx-workspace-split" data-direction={split.direction}>
    <div className="nx-workspace-split__pane" style={{ flexBasis: `${split.ratio * 100}%` }}><NodeView node={split.first} renderSurface={renderSurface} popoutSurface={popoutSurface} /></div>
    <div className="nx-workspace-splitter" role="separator" aria-orientation={horizontal ? 'vertical' : 'horizontal'} aria-valuemin={20} aria-valuemax={80} aria-valuenow={Math.round(split.ratio * 100)} tabIndex={0} onPointerDown={startDrag} onKeyDown={keyboardResize}><GripVertical size={12} /></div>
    <div className="nx-workspace-split__pane" style={{ flexBasis: `${(1 - split.ratio) * 100}%` }}><NodeView node={split.second} renderSurface={renderSurface} popoutSurface={popoutSurface} /></div>
  </div>;
};

const WorkspaceStackView: React.FC<{ stack: WorkspaceStack; renderSurface: (surface: WorkspaceSurface) => React.ReactNode; popoutSurface?: (surface: WorkspaceSurface) => void }> = ({ stack, renderSurface, popoutSurface }) => {
  const { t } = useTranslation();
  const { activate, close, move, splitEmpty, maximize, model } = useWorkspace();
  const [draggedSurface, setDraggedSurface] = useState<string | null>(null);
  const active = useMemo(
    () => stack.tabs.find((tab) => isSurfaceMatch(tab, stack.activeId)) ?? stack.tabs[0],
    [stack.tabs, stack.activeId]
  );
  const canClose = active?.closable !== false;
  const legacyTitleKeys: Record<string, string> = { overview: 'nav.overview', work: 'nav.work', missions: 'nav.missions', agents: 'nav.agents', maestro: 'nav.maestro', sessions: 'nav.sessions', settings: 'nav.settings', resources: 'nav.resources', 'legacy-runtimes': 'nav.runtimes', 'legacy-providers': 'nav.providers', 'legacy-events': 'nav.events' };
  const displayTitle = (surface: WorkspaceSurface) => {
    const key = surface.titleKey || legacyTitleKeys[surface.type];
    return key ? t(key, surface.titleParams) : surface.title;
  };
  return <section className="nx-workspace-stack" data-stack-id={stack.id} onDragOver={(event) => { if (event.dataTransfer.types.includes('application/x-nexus-surface')) event.preventDefault(); }} onDrop={(event) => {
    const id = event.dataTransfer.getData('application/x-nexus-surface') || draggedSurface;
    if (id) move(id, stack.id);
    setDraggedSurface(null);
  }}>
    <header className="nx-workspace-stack__tabs">
      <div className="nx-workspace-tabs" role="tablist" aria-label={t('shell.openSurfaces')}>
        {stack.tabs.map((surface) => {
          const isActive = isSurfaceMatch(surface, stack.activeId);
          return (
            <button
              draggable
              key={surface.id}
              type="button"
              role="tab"
              aria-selected={isActive}
              data-active={isActive ? 'true' : 'false'}
              className="nx-workspace-tab"
              onDragStart={(event) => {
                setDraggedSurface(surface.id);
                event.dataTransfer.setData('application/x-nexus-surface', surface.id);
                event.dataTransfer.effectAllowed = 'move';
              }}
              onClick={() => activate(surface.id)}
            >
              <span>{displayTitle(surface)}</span>
              {surface.closable !== false && (
                <span
                  className="nx-workspace-tab__close"
                  role="button"
                  aria-label={t("workspace.closeNamed", { name: displayTitle(surface) })}
                  onClick={(event) => {
                    event.stopPropagation();
                    close(surface.id);
                  }}
                >
                  <X size={11} />
                </span>
              )}
            </button>
          );
        })}
      </div>
      {active && <div className="nx-workspace-stack__actions">
        <IconButton label={t('workspace.splitRight')} onClick={() => splitEmpty(active.id, 'horizontal')}><Columns2 size={13} /></IconButton>
        <IconButton label={t('workspace.splitDown')} onClick={() => splitEmpty(active.id, 'vertical')}><Rows2 size={13} /></IconButton>
        {popoutSurface && <IconButton label={t('workspace.popout')} onClick={() => popoutSurface(active)}><ExternalLink size={13} /></IconButton>}
        <IconButton label={t(model.maximizedSurfaceId === active.id ? 'workspace.restore' : 'workspace.maximize')} onClick={() => maximize(active.id)}>{model.maximizedSurfaceId === active.id ? <Minimize2 size={13} /> : <Maximize2 size={13} />}</IconButton>
        {canClose && <IconButton label={t('workspace.close')} onClick={() => close(active.id)}><X size={13} /></IconButton>}
      </div>}
    </header>
    <div className="nx-workspace-stack__body">
      {stack.tabs.map((surface) => {
        const isActive = isSurfaceMatch(surface, stack.activeId);
        return (
          <div
            key={surface.id}
            role="tabpanel"
            aria-hidden={!isActive}
            data-active={isActive ? 'true' : 'false'}
            className="nx-workspace-panel"
            style={{
              display: isActive ? 'flex' : 'none',
              flexDirection: 'column',
              width: '100%',
              height: '100%',
              position: 'absolute',
              inset: 0,
            }}
          >
            {renderSurface(surface)}
          </div>
        );
      })}
    </div>
  </section>;
};
