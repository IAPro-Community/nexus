import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ArrowLeft, ArrowRight, Check, Compass, X } from 'lucide-react';
import { Button, IconButton } from '../../design-system';
import {
  availableTourSteps,
  nextTourIndex,
  previousTourIndex,
  productTourSteps,
  type TourStep,
} from './tour';
import { useTranslation } from 'react-i18next';
import styles from './ProductTour.module.scss';

export const ProductTour: React.FC<{
  open: boolean;
  onClose: () => void;
  steps?: TourStep[];
  initialIndex?: number;
  onStepChange?: (index: number) => void;
  onComplete?: () => void;
}> = ({
  open,
  onClose,
  steps: proposedSteps = productTourSteps,
  initialIndex = 0,
  onStepChange,
  onComplete,
}) => {
  const { t } = useTranslation();
  const [index, setIndex] = useState(initialIndex);
  const [rect, setRect] = useState<DOMRect | null>(null);
  const modalRef = useRef<HTMLDivElement>(null);
  const returnFocus = useRef<HTMLElement | null>(null);

  // DOM targets must be checked when the tour is opened, not when the
  // component first mounts: Composer/Flow render different project states.
  const steps = useMemo(() => {
    if (!open || typeof document === 'undefined') return [];
    return availableTourSteps(proposedSteps, (selector) => {
      const el = document.querySelector(selector) as HTMLElement | null;
      if (!el) return false;
      const r = el.getBoundingClientRect();
      return r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0;
    });
  }, [open, proposedSteps]);

  const step = steps[index];

  useEffect(() => {
    if (!open) return;
    returnFocus.current = document.activeElement as HTMLElement | null;
    setIndex(Math.min(Math.max(0, initialIndex), Math.max(steps.length - 1, 0)));
    const raf = window.requestAnimationFrame(() => modalRef.current?.focus());
    return () => {
      window.cancelAnimationFrame(raf);
      returnFocus.current?.focus();
    };
  }, [open, initialIndex, steps.length]);

  useEffect(() => {
    if (!open || !step) {
      setRect(null);
      return;
    }
    const target = document.querySelector(step.target) as HTMLElement | null;
    if (!target) {
      setRect(null);
      return;
    }
    target.scrollIntoView({
      behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth',
      block: 'nearest',
    });
    const update = () => setRect(target.getBoundingClientRect());
    update();
    window.addEventListener('resize', update);
    window.addEventListener('scroll', update, true);
    return () => {
      window.removeEventListener('resize', update);
      window.removeEventListener('scroll', update, true);
    };
  }, [open, step]);

  const changeStep = (next: number) => {
    const clamped = Math.min(Math.max(0, next), Math.max(0, steps.length - 1));
    setIndex(clamped);
    onStepChange?.(clamped);
  };

  const onTourKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      onClose();
      return;
    }
    if (event.key !== 'Tab' || !modalRef.current) return;
    const buttons = Array.from(
      modalRef.current.querySelectorAll<HTMLElement>('button:not([disabled])'),
    );
    if (!buttons.length) return;
    const first = buttons[0],
      last = buttons[buttons.length - 1];
    if (
      event.shiftKey &&
      (document.activeElement === first || document.activeElement === modalRef.current)
    ) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  };

  if (!open || !step || !rect) return null;

  const viewportHeight = window.innerHeight;
  const cardEstimate = Math.min(300, Math.max(120, viewportHeight - 24));
  const below = rect.bottom + cardEstimate + 12 <= viewportHeight;
  const position = below ? rect.bottom + 12 : rect.top - cardEstimate - 12;
  const top = Math.min(Math.max(12, position), Math.max(12, viewportHeight - cardEstimate - 12));
  const left = Math.min(Math.max(12, rect.left), Math.max(12, window.innerWidth - 380));

  return (
    <div
      ref={modalRef}
      className="nx-tour-layer"
      role="dialog"
      aria-modal="true"
      aria-label={t('tour.aria')}
      tabIndex={-1}
      onKeyDown={onTourKeyDown}
    >
      <div
        className="nx-tour-highlight"
        style={{
          top: Math.max(0, rect.top - 4),
          left: Math.max(0, rect.left - 4),
          width: rect.width + 8,
          height: rect.height + 8,
        }}
      />
      <div className={`nx-tour-card ${styles.tourCard}`} style={{ top, left }}>
        <header>
          <span>
            <Compass size={15} /> {t('tour.label', { current: index + 1, total: steps.length })}
          </span>
          <IconButton label={t('tour.close')} onClick={onClose}>
            <X size={14} />
          </IconButton>
        </header>
        <strong>{t(step.title)}</strong>
        <p>{t(step.body)}</p>
        <div className="nx-tour-footer">
          <Button
            size="sm"
            tone="ghost"
            disabled={index === 0}
            onClick={() => changeStep(previousTourIndex(index))}
          >
            <ArrowLeft size={13} /> {t('tour.back')}
          </Button>
          {index === steps.length - 1 ? (
            <Button size="sm" tone="brand" onClick={onComplete || onClose}>
              <Check size={13} /> {t('tour.finish')}
            </Button>
          ) : (
            <Button
              size="sm"
              tone="brand"
              onClick={() => changeStep(nextTourIndex(index, steps.length))}
            >
              {t('tour.next')} <ArrowRight size={13} />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};
