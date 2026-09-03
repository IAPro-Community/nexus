import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./PlanBuilderSurface.tsx', import.meta.url), 'utf8');

describe('Mission scheduling wiring', () => {
  it('allows AFTER_RUN scheduling while the dependency Mission is still running', () => {
    expect(source).toContain(">Run after current Mission</Button>");
    expect(source).not.toContain("disabled={activeRun.state !== 'COMPLETED_VERIFIED'}");
  });
});
