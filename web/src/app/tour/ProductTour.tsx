import React, { useEffect, useMemo, useState } from 'react';
import { ArrowLeft, ArrowRight, Check, Compass, X } from 'lucide-react';
import { Button, IconButton } from '../../design-system';
import { availableTourSteps, nextTourIndex, previousTourIndex, productTourSteps } from './tour';
import { useTranslation } from 'react-i18next';

export const ProductTour: React.FC<{ open: boolean; onClose: () => void }> = ({
  open,
  onClose,
}) => {
  const { t } = useTranslation();
  const [index, setIndex] = useState(0);
  const [rect, setRect] = useState<DOMRect | null>(null);

  const steps = useMemo(
    () =>
      availableTourSteps(productTourSteps, (selector) => {
        const el = document.querySelector(selector) as HTMLElement | null;
        if (!el) return false;
        const r = el.getBoundingClientRect();
        return r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0;
      }),
    [],
  );

  const step = steps[index];

  useEffect(() => {
    if (open) setIndex(0);
  }, [open]);

  useEffect(() => {
    if (!open || !step) return;
    const target = document.querySelector(step.target) as HTMLElement | null;
    if (!target) return;
    target.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    const update = () => setRect(target.getBoundingClientRect());
    update();
    window.addEventListener('resize', update);
    window.addEventListener('scroll', update, true);
    return () => {
      window.removeEventListener('resize', update);
      window.removeEventListener('scroll', update, true);
    };
  }, [open, step]);

  useEffect(() => {
    if (index >= steps.length) setIndex(Math.max(0, steps.length - 1));
  }, [index, steps.length]);

  if (!open || !step || !rect) return null;

  const below = rect.bottom + 210 < window.innerHeight;
  const top = below
    ? Math.min(window.innerHeight - 210, rect.bottom + 12)
    : Math.max(12, rect.top - 190);
  const left = Math.min(Math.max(12, rect.left), Math.max(12, window.innerWidth - 380));

  return (
    <div className="nx-tour-layer" role="dialog" aria-modal="true" aria-label={t('tour.aria')}>
      <div
        className="nx-tour-highlight"
        style={{
          top: Math.max(0, rect.top - 4),
          left: Math.max(0, rect.left - 4),
          width: rect.width + 8,
          height: rect.height + 8,
        }}
      />
      <div className="nx-tour-card" style={{ top, left }}>
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
            onClick={() => setIndex(previousTourIndex(index))}
          >
            <ArrowLeft size={13} /> {t('tour.back')}
          </Button>
          {index === steps.length - 1 ? (
            <Button size="sm" tone="brand" onClick={onClose}>
              <Check size={13} /> {t('tour.finish')}
            </Button>
          ) : (
            <Button
              size="sm"
              tone="brand"
              onClick={() => setIndex(nextTourIndex(index, steps.length))}
            >
              {t('tour.next')} <ArrowRight size={13} />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};
