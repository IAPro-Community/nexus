import { describe, expect, it } from 'vitest';
import { shouldRenderHudPopover } from './hudPresentation';

describe('HUD presentation surface contract', () => {
  it('preserves the web HUD collapsed and expanded states', () => {
    expect(shouldRenderHudPopover(false, false, 'capsule', false)).toBe(false);
    expect(shouldRenderHudPopover(true, false, 'capsule', false)).toBe(true);
    // Desktop-only detached state must never affect the browser HUD.
    expect(shouldRenderHudPopover(true, false, 'capsule', true)).toBe(true);
  });

  it('preserves the single-window native HUD fallback', () => {
    expect(shouldRenderHudPopover(false, true, 'capsule', false)).toBe(false);
    expect(shouldRenderHudPopover(true, true, 'capsule', false)).toBe(true);
  });

  it('never draws the detached details inside the capsule', () => {
    expect(shouldRenderHudPopover(false, true, 'capsule', true)).toBe(false);
    expect(shouldRenderHudPopover(true, true, 'capsule', true)).toBe(false);
  });

  it('pre-renders independent native popup content even before controller polling', () => {
    expect(shouldRenderHudPopover(false, true, 'popover', true)).toBe(true);
    expect(shouldRenderHudPopover(true, true, 'popover', true)).toBe(true);
  });
});
