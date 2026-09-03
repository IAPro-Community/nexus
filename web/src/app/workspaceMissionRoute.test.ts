import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./WorkspaceSurfaceHost.tsx', import.meta.url), 'utf8');

describe('Workspace OS mission surface wiring', () => {
  it('routes the primary Missions surface to the WorkPlan/Mission Runner builder', () => {
    expect(source).toContain("import { PlanBuilderSurface } from '../features/work/PlanBuilderSurface';");
    expect(source).toMatch(/surface\.type === 'missions'[\s\S]*?<PlanBuilderSurface project=\{project\} agents=\{agents\}/);
    expect(source).not.toMatch(/surface\.type === 'missions'[\s\S]*?<MissionsPage projectId=/);
  });
});
