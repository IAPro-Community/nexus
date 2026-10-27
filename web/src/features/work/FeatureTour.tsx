import React, { useEffect, useState } from 'react';
import { Compass } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../design-system';
import { ProductTour } from '../../app/tour/ProductTour';
import {
  featureTourSteps, featureTourStorageKey, parseFeatureTourProgress,
  type FeatureTourSurface,
} from './featureTour';

const readProgress = (key: string, length: number): number => {
  try { return parseFeatureTourProgress(window.localStorage.getItem(key), length); }
  catch { return 0; }
};

/**
 * Explicit, resumable and non-destructive contextual walkthrough. Does not
 * click buttons, mutate data, start agents, switch tabs or change project.
 * The global workspace tour remains independent.
 */
export const FeatureTour: React.FC<{
  projectId: string;
  surface: FeatureTourSurface;
}> = ({ projectId, surface }) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const steps = featureTourSteps[surface];
  const key = featureTourStorageKey(projectId, surface);
  const [initialIndex, setInitialIndex] = useState(0);
  // Switching projects must not leave a tour for the previous project open.
  useEffect(() => {
    setOpen(false);
  }, [projectId, surface]);
  const save = (index: number) => {
    try { window.localStorage.setItem(key, String(index)); }
    catch { /* No persistence available; tour still usable. */ }
  };
  const begin = () => {
    setInitialIndex(readProgress(key, steps.length));
    setOpen(true);
  };
  const finish = () => {
    save(0); // Replaying a completed tour starts from its beginning.
    setOpen(false);
  };
  return (
    <>
      <Button size="sm" onClick={begin} aria-label={t('tour.featureOpen', 'Conhecer esta área')}>
        <Compass size={14} /> {t('tour.featureOpen', 'Conhecer esta área')}
      </Button>
      <ProductTour
        open={open}
        steps={steps}
        initialIndex={initialIndex}
        onStepChange={save}
        onClose={() => setOpen(false)}
        onComplete={finish}
      />
    </>
  );
};
