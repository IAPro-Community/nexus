import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./WorkspaceSurfaceHost.tsx', import.meta.url), 'utf8');

describe('Workspace OS mission surface wiring', () => {
  it('routes Missions to FlowRunsHistorySurface, not PlanBuilder or legacy MissionsPage', () => {
    expect(source).toMatch(/import\(['"]\.\.\/features\/work\/FlowRunsHistorySurface['"]\)/);
    expect(source).toMatch(/surface\.type === 'missions'[\s\S]*?<FlowRunsHistorySurface/);
    expect(source).not.toMatch(/surface\.type === 'missions'[\s\S]*?<PlanBuilderSurface/);
    expect(source).not.toContain('MissionsPage');
  });

  it('mounts PlanBuilder once via the Composer work surface', () => {
    expect(source).toMatch(/surface\.type === 'work'[\s\S]*?<WorkSurface/);
  });

  it('does not mount legacy MaestroPage on maestro tabs, uses MaestroSurface', () => {
    expect(source).not.toContain('MaestroPage');
    expect(source).toContain('MaestroSurface');
  });
});
