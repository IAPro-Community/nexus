import { describe, expect, it } from 'vitest';
import { findTourTarget } from './tourTargets';

const target = (visible: boolean) =>
  ({
    getBoundingClientRect: () => ({
      width: visible ? 100 : 0,
      height: 30,
      right: 100,
      bottom: 30,
    }),
  }) as unknown as HTMLElement;

const scope = (...elements: HTMLElement[]) =>
  ({
    querySelectorAll: (_selector: string) => elements,
  }) as unknown as ParentNode;

describe('contextual tour target resolution', () => {
  it('never falls back to another project or document', () => {
    const otherProjectTarget = target(true);
    expect(findTourTarget(null, '[data-tour="composer-goal"]')).toBeNull();
    expect(findTourTarget(scope(), '[data-tour="composer-goal"]')).toBeNull();
    expect(findTourTarget(scope(otherProjectTarget), '[data-tour="composer-goal"]')).toBe(
      otherProjectTarget,
    );
  });

  it('skips hidden duplicates inside one workspace', () => {
    const hidden = target(false);
    const visible = target(true);
    expect(findTourTarget(scope(hidden, visible), '[data-tour="flow-canvas"]')).toBe(visible);
  });

  it('accepts targets that are below the viewport, ready for scrollIntoView', () => {
    const belowFold = {
      getBoundingClientRect: () => ({
        width: 150,
        height: 48,
        left: 20,
        top: 1800,
        right: 170,
        bottom: 1848,
      }),
    } as unknown as HTMLElement;
    expect(findTourTarget(scope(belowFold), '[data-tour="flow-run-status"]')).toBe(belowFold);
  });
});
