import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import {
  NEX_ATLAS_ANIMATION_MS,
  NEX_ATLAS_FRAMES,
  NEX_ATLAS_FRAME_MS,
  NEX_ATLAS_FRAME_PX,
  NEX_ATLAS_HOLD_LAST_ACTIONS,
  NEX_ATLAS_ROW_BY_ACTION,
  NEX_ATLAS_SHA256,
  NEX_ATLAS_URL,
} from './nexIdleAtlas';
import { NEX_IDLE_ACTIONS } from './nexIdleDirector';

describe('NEX hud-32bit idle atlas (locked NEX-ID-v1 asset)', () => {
  it('serves the byte-identical locked atlas from the web public root', () => {
    const assetPath = resolve(process.cwd(), 'public', NEX_ATLAS_URL.replace(/^\//, ''));
    const sha = createHash('sha256').update(readFileSync(assetPath)).digest('hex');
    expect(sha).toBe(NEX_ATLAS_SHA256);
  });

  it('carries exactly the 12 approved idle actions in manifest order', () => {
    expect(Object.keys(NEX_ATLAS_ROW_BY_ACTION)).toEqual([...NEX_IDLE_ACTIONS]);
    expect(Object.keys(NEX_ATLAS_ROW_BY_ACTION)).toHaveLength(12);
    expect(new Set(Object.values(NEX_ATLAS_ROW_BY_ACTION)).size).toBe(12);
    expect(NEX_ATLAS_ROW_BY_ACTION.idle_hover).toBe(0);
    expect(NEX_ATLAS_ROW_BY_ACTION.idle_wake).toBe(11 * NEX_ATLAS_FRAME_PX);
  });

  it('keeps the manifest timing: 8 frames of 125 ms (1000 ms per animation)', () => {
    expect(NEX_ATLAS_FRAMES).toBe(8);
    expect(NEX_ATLAS_FRAME_MS).toBe(125);
    expect(NEX_ATLAS_ANIMATION_MS).toBe(1000);
  });

  it('holds the last frame only for drowse and wake', () => {
    expect([...NEX_ATLAS_HOLD_LAST_ACTIONS]).toEqual(['idle_drowse', 'idle_wake']);
  });
});
