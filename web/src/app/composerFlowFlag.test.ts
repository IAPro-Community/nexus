import { describe, expect, it } from 'vitest';
import { createWorkspace, openSurface, splitWithSurface } from '../workspace/model';
import type { WorkspaceSurface } from '../workspace/model';
import {
  COMPOSER_FLOW_ENABLED,
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
  it('is enabled by default in normal test/build', () => {
    expect(COMPOSER_FLOW_ENABLED).toBe(true);
    expect(isExperimentalComposerFlowSurface('work')).toBe(true);
    expect(isExperimentalComposerFlowSurface('missions')).toBe(true);
    expect(isExperimentalComposerFlowSurface('flow-run')).toBe(true);
    expect(isExperimentalComposerFlowSurface('terminals')).toBe(false);
  });

  it('keeps mission decision routes available while hiding experimental planning surfaces', () => {
    expect(isExperimentalRouteBlocked('missions', 'mission-123')).toBe(false);
    expect(isExperimentalRouteBlocked('missions')).toBe(true);
    expect(isExperimentalRouteBlocked('work')).toBe(true);
    expect(isExperimentalRouteBlocked('flow-run', 'mission-123')).toBe(true);
  });

  it('allows only a mission-specific Flow detail surface in the workspace', () => {
    expect(isExperimentalWorkspaceSurfaceBlocked(surface('flow-run'))).toBe(true);
    expect(
      isExperimentalWorkspaceSurfaceBlocked({
        ...surface('flow-run'),
        data: { runId: 'mission-123' },
      }),
    ).toBe(false);
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
