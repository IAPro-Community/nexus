import { afterEach, describe, expect, it } from 'vitest';
import {
  cancelFeatureReveal,
  consumeFeatureReveal,
  hasPendingFeatureReveal,
  requestFeatureReveal,
} from './featureReveal';

afterEach(() => cancelFeatureReveal());

describe('global feature announcement handoff', () => {
  it('rejects unknown features and blank projects', () => {
    expect(requestFeatureReveal('nonexistent', 'project-A', 1000)).toBe(false);
    expect(requestFeatureReveal('composer-idea-workspace', '', 1000)).toBe(false);
  });

  it('never exposes an announcement in another project or another surface', () => {
    expect(requestFeatureReveal('composer-idea-workspace', 'project-A', 1000)).toBe(true);
    expect(consumeFeatureReveal('project-B', 'composer-start', () => true, 1001)).toBeNull();
    expect(consumeFeatureReveal('project-A', 'flow-library', () => true, 1001)).toBeNull();
    expect(hasPendingFeatureReveal('project-A', 'composer-start', 1002)).toBe(true);
  });

  it('waits until its real target has mounted, then consumes only once', () => {
    requestFeatureReveal('flow-pipeline-graph', 'project-A', 1000);
    expect(consumeFeatureReveal('project-A', 'flow-editor', () => false, 1001)).toBeNull();
    const item = consumeFeatureReveal(
      'project-A',
      'flow-editor',
      (target) => target === '[data-tour="flow-canvas"]',
      1002,
    );
    expect(item?.id).toBe('flow-pipeline-graph');
    expect(consumeFeatureReveal('project-A', 'flow-editor', () => true, 1003)).toBeNull();
  });

  it('reveals the Agent follow control only in the selected project terminal', () => {
    requestFeatureReveal('agent-terminal-follow', 'project-A', 1000);
    expect(consumeFeatureReveal('project-B', 'agent-terminal', () => true, 1001)).toBeNull();
    expect(consumeFeatureReveal('project-A', 'resources', () => true, 1002)).toBeNull();
    const item = consumeFeatureReveal(
      'project-A',
      'agent-terminal',
      (target) => target === '[data-tour="agent-terminal-follow"]',
      1003,
    );
    expect(item?.id).toBe('agent-terminal-follow');
  });

  it('reveals the Browser inspect tour only within the selected project Browser surface', () => {
    requestFeatureReveal('browser-design-inspect', 'project-A', 1000);
    expect(consumeFeatureReveal('project-B', 'browser-inspector', () => true, 1001)).toBeNull();
    expect(consumeFeatureReveal('project-A', 'browser-inspector', () => false, 1002)).toBeNull();
    const item = consumeFeatureReveal(
      'project-A',
      'browser-inspector',
      (target) => target === '[data-tour="browser-inspector-entry"]',
      1003,
    );
    expect(item?.id).toBe('browser-design-inspect');
  });

  it('reveals Agent groups only within the selected project Agents surface', () => {
    requestFeatureReveal('agent-groups-task-worktree-feature', 'project-A', 1000);
    expect(consumeFeatureReveal('project-B', 'agents', () => true, 1001)).toBeNull();
    expect(consumeFeatureReveal('project-A', 'agent-terminal', () => true, 1002)).toBeNull();
    const item = consumeFeatureReveal(
      'project-A',
      'agents',
      (target) => target === '[data-tour="agent-groups"]',
      1003,
    );
    expect(item?.id).toBe('agent-groups-task-worktree-feature');
  });

  it('expires old requests so tours cannot surprise users later', () => {
    requestFeatureReveal('composer-idea-workspace', 'project-A', 1000);
    expect(hasPendingFeatureReveal('project-A', 'composer-start', 32_000)).toBe(false);
    expect(consumeFeatureReveal('project-A', 'composer-start', () => true, 32_001)).toBeNull();
  });

  it('does not reveal a project feature merely because another panel has the same marker', () => {
    requestFeatureReveal('flow-pipeline-graph', 'project-A', 1000);
    const anotherPanelHasTarget = true;
    expect(anotherPanelHasTarget).toBe(true);
    expect(consumeFeatureReveal('project-A', 'flow-editor', () => false, 1001)).toBeNull();
    expect(hasPendingFeatureReveal('project-A', 'flow-editor', 1002)).toBe(true);
    expect(consumeFeatureReveal('project-A', 'flow-editor', () => true, 1003)?.id).toBe(
      'flow-pipeline-graph',
    );
  });

  it('replaces a pending request when the user chooses another feature', () => {
    requestFeatureReveal('composer-idea-workspace', 'project-A', 1000);
    requestFeatureReveal('flow-review-journey', 'project-A', 1001);
    expect(hasPendingFeatureReveal('project-A', 'composer-start', 1002)).toBe(false);
    expect(consumeFeatureReveal('project-A', 'flow-library', () => true, 1002)?.id).toBe(
      'flow-review-journey',
    );
  });
});
