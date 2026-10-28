// Compatibility entrypoint: all Composer/Flow tour steps are defined once.
// New work should update featureTourDefinitions.ts, not duplicate the map.
export {
  featureTourSteps, featureTourStorageKey, parseFeatureTourProgress,
} from './featureTourDefinitions';
export type { FeatureTourSurface } from './featureTourDefinitions';
