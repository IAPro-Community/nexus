import React, { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Plus, Sparkles, TerminalSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export type ProjectCreateMenuProps = {
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
  size?: 'sm' | 'xs';
  variant?: 'topbar' | 'compact' | 'ghost';
  label?: string;
  className?: string;
};

export const ProjectCreateMenu: React.FC<ProjectCreateMenuProps> = ({
  onNewAgent,
  onNewAISession,
  onProjectShell,
  size = 'sm',
  variant = 'topbar',
  label,
  className,
}) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const [menuPos, setMenuPos] = useState<{ top: number; left?: number; right?: number } | null>(
    null,
  );

  useEffect(() => {
    if (!open) {
      setMenuPos(null);
      return;
    }

    const compute = () => {
      const el = rootRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const fitsRight = rect.left + 240 < window.innerWidth;
      if (fitsRight) {
        setMenuPos({ top: rect.bottom + 4, left: Math.max(8, rect.left) });
      } else {
        setMenuPos({ top: rect.bottom + 4, right: Math.max(8, window.innerWidth - rect.right) });
      }
    };

    compute();
    const onResize = () => compute();
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if ((target as Element).closest?.('.nx-create-menu__panel')) return;
      setOpen(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };

    window.addEventListener('resize', onResize);
    window.addEventListener('scroll', onResize, true);
    window.addEventListener('pointerdown', onPointerDown, true);
    window.addEventListener('keydown', onKey);

    return () => {
      window.removeEventListener('resize', onResize);
      window.removeEventListener('scroll', onResize, true);
      window.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('keydown', onKey);
    };
  }, [open]);

  if (!onNewAgent && !onNewAISession && !onProjectShell) return null;

  const defaultLabel = label ?? t('overview.create');

  const panel =
    open && menuPos
      ? createPortal(
          <div
            className="nx-create-menu__panel"
            role="menu"
            aria-label={t('overview.createMenuTooltip')}
            style={{
              top: menuPos.top,
              ...(menuPos.left !== undefined ? { left: menuPos.left } : { right: menuPos.right }),
            }}
            onPointerDown={(e) => e.stopPropagation()}
          >
            {onNewAgent && (
              <button
                type="button"
                role="menuitem"
                className="nx-create-menu__item"
                onClick={() => {
                  setOpen(false);
                  onNewAgent();
                }}
              >
                <div className="nx-create-menu__icon nx-create-menu__icon--agent">
                  <Plus size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.newAgent')}</strong>
                  <small>{t('overview.newAgentHint')}</small>
                </div>
              </button>
            )}

            {onNewAISession && (
              <button
                type="button"
                role="menuitem"
                className="nx-create-menu__item"
                onClick={() => {
                  setOpen(false);
                  onNewAISession();
                }}
              >
                <div className="nx-create-menu__icon nx-create-menu__icon--session">
                  <Sparkles size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.newAISession')}</strong>
                  <small>{t('overview.newAISessionHint')}</small>
                </div>
              </button>
            )}

            {onProjectShell && (
              <button
                type="button"
                role="menuitem"
                className="nx-create-menu__item"
                onClick={() => {
                  setOpen(false);
                  onProjectShell();
                }}
              >
                <div className="nx-create-menu__icon nx-create-menu__icon--terminal">
                  <TerminalSquare size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.projectShell')}</strong>
                  <small>{t('overview.projectShellHint')}</small>
                </div>
              </button>
            )}
          </div>,
          document.body,
        )
      : null;

  return (
    <div
      ref={rootRef}
      className={`nx-create-menu ${className ? className : ''}`}
      data-variant={variant}
      data-size={size}
    >
      <button
        type="button"
        data-testid="topbar-create-menu-btn"
        className="nx-create-menu__trigger nx-button"
        data-size={size}
        data-tone={variant === 'ghost' ? undefined : 'brand'}
        aria-expanded={open}
        aria-haspopup="menu"
        title={t('overview.createMenuTooltip')}
        onClick={() => setOpen((val) => !val)}
      >
        <Plus size={size === 'xs' ? 12 : 13} />
        <span className="nx-create-menu__label">{defaultLabel}</span>
        <ChevronDown size={11} className="nx-create-menu__chevron" />
      </button>
      {panel}
    </div>
  );
};
