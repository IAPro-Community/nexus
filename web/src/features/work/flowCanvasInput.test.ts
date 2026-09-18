import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./FlowCanvas.tsx', import.meta.url), 'utf8');
const styles = readFileSync(new URL('../../app/workspace-os.css', import.meta.url), 'utf8');

describe('Flow canvas wheel interaction', () => {
  it('does not call preventDefault from React wheel handlers', () => {
    expect(source).not.toMatch(/onWheel=\{[\s\S]*?preventDefault\(\)/);
  });

  it('contains scroll chaining at the canvas boundary', () => {
    expect(styles).toMatch(/\.nx-flow-canvas\s*\{[\s\S]*?overscroll-behavior:\s*contain;/);
  });

  it('fits the graph once after init instead of enabling continuous fitView', () => {
    expect(source).not.toMatch(/^\s+fitView\s*$/m);
    expect(source).toContain('instance.fitView({ padding: 0.2 })');
  });
});

const historySource = readFileSync(
  new URL('./FlowRunsHistorySurface.tsx', import.meta.url),
  'utf8',
);
const historyStyles = readFileSync(
  new URL('./FlowRunsHistorySurface.module.scss', import.meta.url),
  'utf8',
);

describe('Flow studio surface layout', () => {
  it('scrolls inside the workspace panel instead of being clipped', () => {
    expect(historySource).toContain('nx-surface-scroll');
    expect(historyStyles).toMatch(/\.canvasHost\s*\{[\s\S]*?height:\s*100%;/);
  });
});
