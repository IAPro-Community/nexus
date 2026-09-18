import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';

const railSource = readFileSync(new URL('./ProjectRail.tsx', import.meta.url), 'utf8');
const treeSource = readFileSync(new URL('./ProjectTreeItem.tsx', import.meta.url), 'utf8');

describe('Project rail situational outline', () => {
  it('exposes Maestro and overview as system surfaces in optional tools', () => {
    expect(railSource).not.toContain('rail.maestroMethod');
    expect(railSource).toContain("id: 'maestro'");
    expect(railSource).toContain("id: 'overview'");
    expect(railSource).toContain("label: t('nav.overview')");
  });

  it('uses the project tree outline instead of a flat agents accordion', () => {
    expect(railSource).toContain('ProjectTreeItem');
    expect(railSource).toContain('useProjectRailOutline');
    expect(railSource).not.toContain('nx_rail_agents_open');
    expect(railSource).toContain("id: 'agents'");
  });

  it('keeps expand distinct from project selection', () => {
    expect(treeSource).toContain('onToggleExpand');
    expect(treeSource).toContain('onSelect');
    expect(treeSource).toContain('ArrowRight');
    expect(treeSource).toContain('ArrowLeft');
    expect(treeSource).toContain('rail.neverOpened');
    expect(treeSource).toContain('rail.emptyOutline');
  });

  it('does not select the project when the chevron is clicked', () => {
    expect(treeSource).toContain('event.stopPropagation()');
    expect(treeSource).toContain('onToggleExpand()');
  });
});
