import { afterEach, describe, expect, it } from 'vitest';
import { createWorkspace, openSurface, splitWithSurface } from '../workspace/model';
import type { WorkspaceSurface } from '../workspace/model';
import {
  COMPOSER_FLOW_ENABLED,
  OPTIONAL_FEATURES_ENABLED,
  VOICE_ACTIONS_ENABLED,
  DESIGN_INSPECT_ENABLED,
  VISUAL_DIFF_ENABLED,
  NEX_PROACTIVITY_ENABLED,
  COMPARISON_LAB_ENABLED,
  SMART_LAYOUTS_ENABLED,
  DEVICE_PREVIEW_ENABLED,
  applyOptionalFeatureSnapshot,
  getOptionalFeatures,
  subscribeOptionalFeatures,
  isExperimentalComposerFlowSurface,
  isExperimentalRouteBlocked,
  isExperimentalWorkspaceSurfaceBlocked,
  withoutExperimentalComposerFlow,
} from './composerFlowFlag';

const surface = (type: string, id = type): WorkspaceSurface => ({
  id,
  type,
  title: type,
  closable: false,
});

describe('Composer/Flow release gate', () => {
  afterEach(() => applyOptionalFeatureSnapshot(null));

  it('accepts only explicitly enabled capabilities from the server', () => {
    applyOptionalFeatureSnapshot({
      global: false,
      features: { COMPOSER: true, FLOW: true, MISSIONS: true, NEX_PROACTIVITY: true },
    });
    expect(COMPOSER_FLOW_ENABLED).toBe(false);
    expect(NEX_PROACTIVITY_ENABLED).toBe(false);
    applyOptionalFeatureSnapshot({
      global: true,
      features: {
        COMPOSER: true,
        FLOW: true,
        MISSIONS: true,
        NEX_PROACTIVITY: true,
        VOICE_ACTIONS: 'true',
        DEVICE_PREVIEW: true,
      },
    });
    expect(COMPOSER_FLOW_ENABLED).toBe(true);
    expect(NEX_PROACTIVITY_ENABLED).toBe(true);
    expect(DEVICE_PREVIEW_ENABLED).toBe(true);
    expect(VOICE_ACTIONS_ENABLED).toBe(false);
    expect(COMPARISON_LAB_ENABLED).toBe(false);
    expect(SMART_LAYOUTS_ENABLED).toBe(false);
    applyOptionalFeatureSnapshot(null);
    expect(COMPOSER_FLOW_ENABLED).toBe(false);
  });

  it('publishes immutable snapshots and notifies consumers', () => {
    const previous = getOptionalFeatures();
    const snapshots: ReturnType<typeof getOptionalFeatures>[] = [];
    const unsubscribe = subscribeOptionalFeatures(() => snapshots.push(getOptionalFeatures()));
    applyOptionalFeatureSnapshot({ global: true, features: { SMART_LAYOUTS: true } });
    expect(snapshots).toHaveLength(1);
    expect(Object.isFrozen(snapshots[0])).toBe(true);
    expect(snapshots[0].SMART_LAYOUTS_ENABLED).toBe(true);
    expect(previous.SMART_LAYOUTS_ENABLED).toBe(false);
    unsubscribe();
    applyOptionalFeatureSnapshot(null);
    expect(snapshots).toHaveLength(1);
  });

  it('exposes a frozen FLAGS map covering all 11 optional capabilities individually', () => {
    applyOptionalFeatureSnapshot({
      global: true,
      features: {
        COMPOSER: true,
        FLOW: false,
        PARALLEL_CHALLENGE: true,
      },
    });
    const flags = getOptionalFeatures().FLAGS;
    expect(Object.isFrozen(flags)).toBe(true);
    expect(Object.keys(flags)).toHaveLength(11);
    expect(flags.COMPOSER).toBe(true);
    expect(flags.FLOW).toBe(false);
    expect(flags.MISSIONS).toBe(false);
    expect(flags.PARALLEL_CHALLENGE).toBe(true);
    expect(flags.COMPARISON_LAB).toBe(false);
  });

  it('is disabled by default in normal test/build', () => {
    expect(OPTIONAL_FEATURES_ENABLED).toBe(false);
    expect(COMPOSER_FLOW_ENABLED).toBe(false);
    expect(VOICE_ACTIONS_ENABLED).toBe(false);
    expect(DESIGN_INSPECT_ENABLED).toBe(false);
    expect(VISUAL_DIFF_ENABLED).toBe(false);
    expect(isExperimentalComposerFlowSurface('work')).toBe(true);
    expect(isExperimentalComposerFlowSurface('missions')).toBe(true);
    expect(isExperimentalComposerFlowSurface('flow-run')).toBe(true);
    expect(isExperimentalComposerFlowSurface('terminals')).toBe(false);
  });

  it('blocks legacy mission deep links as well as planning routes', () => {
    expect(isExperimentalRouteBlocked('missions', 'mission-123')).toBe(true);
    expect(isExperimentalRouteBlocked('missions')).toBe(true);
    expect(isExperimentalRouteBlocked('work')).toBe(true);
    expect(isExperimentalRouteBlocked('flow-run', 'mission-123')).toBe(true);
  });

  it('blocks all Flow detail surfaces in the workspace', () => {
    expect(isExperimentalWorkspaceSurfaceBlocked(surface('flow-run'))).toBe(true);
    expect(
      isExperimentalWorkspaceSurfaceBlocked({
        ...surface('flow-run'),
        data: { runId: 'mission-123' },
      }),
    ).toBe(true);
    expect(
      isExperimentalWorkspaceSurfaceBlocked({
        ...surface('missions'),
        data: { runId: 'mission-123' },
      }),
    ).toBe(true);
  });

  it('removes previously pinned Composer/Flow tabs without removing stable views', () => {
    const base = createWorkspace(surface('overview'));
    const withComposer = openSurface(base, surface('work'));
    const withFlow = openSurface(withComposer, surface('missions'));
    const cleaned = withoutExperimentalComposerFlow(withFlow, surface('overview'));
    expect(cleaned.root.kind).toBe('stack');
    if (cleaned.root.kind !== 'stack') return;
    expect(cleaned.root.tabs.map((tab) => tab.type)).toEqual(['overview']);
    expect(cleaned.root.activeId).toBe('overview');
  });

  it('collapses empty experimental splits and keeps the terminal split', () => {
    const base = createWorkspace(surface('overview'));
    const flowSplit = splitWithSurface(base, 'overview', surface('flow-run'), 'horizontal');
    const terminalSplit = splitWithSurface(
      flowSplit,
      'overview',
      surface('terminal'),
      'horizontal',
    );
    const cleaned = withoutExperimentalComposerFlow(terminalSplit, surface('overview'));
    expect(JSON.stringify(cleaned)).not.toContain('"type":"flow-run"');
    expect(JSON.stringify(cleaned)).toContain('"type":"terminal"');
    expect(JSON.stringify(cleaned)).toContain('"type":"overview"');
  });

  it('falls back to Overview when persisted layout only contains Composer', () => {
    const cleaned = withoutExperimentalComposerFlow(
      createWorkspace(surface('work')),
      surface('overview'),
    );
    expect(cleaned.root.kind).toBe('stack');
    if (cleaned.root.kind !== 'stack') return;
    expect(cleaned.root.tabs[0].type).toBe('overview');
    expect(cleaned.root.activeId).toBe('overview');
  });
});
