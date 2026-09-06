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
});
