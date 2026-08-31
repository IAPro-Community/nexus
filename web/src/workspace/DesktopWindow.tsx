import React, { useRef } from 'react';
import { Maximize2, Minimize2, X } from 'lucide-react';
import type { DesktopWindowState } from './presentation';
import type { WorkspaceSurface } from './model';

export const DesktopWindow: React.FC<{
  surface: WorkspaceSurface;
  windowState: DesktopWindowState;
  children: React.ReactNode;
  onFocus: () => void;
  onMove: (x: number, y: number) => void;
  onMaximize: () => void;
  onClose: () => void;
}> = ({ surface, windowState, children, onFocus, onMove, onMaximize, onClose }) => {
  const drag = useRef<{ offsetX: number; offsetY: number } | null>(null);
  const startDrag = (event: React.PointerEvent<HTMLDivElement>) => {
    if (windowState.maximized) return;
    const target = event.currentTarget.parentElement;
    if (!target) return;
    const rect = target.getBoundingClientRect();
    drag.current = { offsetX: event.clientX - rect.left, offsetY: event.clientY - rect.top };
    event.currentTarget.setPointerCapture(event.pointerId);
    const onMoveEvent = (move: PointerEvent) => {
      if (drag.current) onMove(Math.max(0, move.clientX - drag.current.offsetX), Math.max(0, move.clientY - drag.current.offsetY));
    };
    const onUp = () => {
      drag.current = null;
      window.removeEventListener('pointermove', onMoveEvent);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMoveEvent);
    window.addEventListener('pointerup', onUp, { once: true });
  };

  return <section
    className="nx-desktop-window"
    data-maximized={windowState.maximized ? 'true' : 'false'}
    style={windowState.maximized ? { zIndex: windowState.zIndex } : { left: windowState.x, top: windowState.y, width: windowState.width, height: windowState.height, zIndex: windowState.zIndex }}
    onPointerDown={onFocus}
    aria-label={surface.title}
  >
    <header className="nx-desktop-window__header" onPointerDown={startDrag}>
      <strong>{surface.title}</strong>
      <div className="nx-desktop-window__actions">
        <button type="button" onClick={onMaximize} aria-label={windowState.maximized ? 'Restore window' : 'Maximize window'}>
          {windowState.maximized ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
        </button>
        {surface.closable !== false && <button type="button" onClick={onClose} aria-label="Close view"><X size={13} /></button>}
      </div>
    </header>
    {!windowState.minimized && <div className="nx-desktop-window__body">{children}</div>}
  </section>;
};
