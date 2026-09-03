import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./modals/MaestroControlModal.tsx', import.meta.url), 'utf8');

describe('Maestro UI honesty', () => {
  it('never fabricates a version or skills when Maestro is unavailable', () => {
    expect(source).not.toContain("|| '0.1.25'");
    expect(source).not.toContain('skill-saas-factory');
    expect(source).not.toContain('skill-security-hooks');
    expect(source).toContain('capabilities?.skills || []');
  });
});
