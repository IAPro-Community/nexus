import { describe, expect, it } from 'vitest';
import {
  shouldRenderHudCompactControl,
  shouldRenderHudMascot,
  shouldRenderHudPopover,
} from './hudPresentation';

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

  it('uses the full native fallback window for expanded details without a second mascot row', () => {
    expect(shouldRenderHudMascot(true, 'capsule', 'MASCOT', false, false)).toBe(true);
    expect(shouldRenderHudMascot(true, 'capsule', 'MASCOT', true, false)).toBe(false);
    expect(shouldRenderHudMascot(true, 'capsule', 'MASCOT', true, true)).toBe(true);
    expect(shouldRenderHudMascot(true, 'popover', 'MASCOT', true, true)).toBe(false);
    expect(shouldRenderHudMascot(true, 'capsule', 'BAR', false, false)).toBe(false);
  });

  it('keeps compact controls only when their own native surface remains visible', () => {
    expect(shouldRenderHudCompactControl(true, 'capsule', false, false)).toBe(true);
    expect(shouldRenderHudCompactControl(true, 'capsule', true, false)).toBe(false);
    expect(shouldRenderHudCompactControl(true, 'capsule', true, true)).toBe(true);
    expect(shouldRenderHudCompactControl(true, 'popover', true, true)).toBe(false);
    expect(shouldRenderHudCompactControl(false, 'capsule', true, false)).toBe(true);
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
