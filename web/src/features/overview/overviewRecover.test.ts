import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const overviewSource = readFileSync(
  new URL('./ProjectOverviewSurface.tsx', import.meta.url),
  'utf8',
);

describe('Overview Recover and Start honest actions', () => {
  it('implements explicit recover and start handlers', () => {
    expect(overviewSource).toContain('handleRecover');
    expect(overviewSource).toContain('handleStart');
    expect(overviewSource).toContain('nexus.recoverAgent');
    expect(overviewSource).toContain('nexus.startAgent');
  });

  it('passes runtimeId upon successful recover/start to open terminal with new runtime', () => {
    expect(overviewSource).toContain('onOpenAgent(agent, runtimeId)');
  });

  it('handles REQUIRED_RESOURCE_SELECTION via ResourcePicker dialog', () => {
    expect(overviewSource).toContain('REQUIRED_RESOURCE_SELECTION');
    expect(overviewSource).toContain('ResourcePicker');
    expect(overviewSource).toContain('resourceAgent');
  });

  it('does not confuse RECOVERABLE with normal click-to-open', () => {
    // Clicking the mini card opens the terminal directly
    expect(overviewSource).toContain('onClick={() => onOpenAgent(agent)}');
    // Clicking the button stops propagation and invokes recover
    expect(overviewSource).toContain('e.stopPropagation()');
    expect(overviewSource).toContain('void handleRecover(agent)');
  });

  it('uses degraded count instead of mixing attention with agent health', () => {
    expect(overviewSource).toContain('const degraded =');
    expect(overviewSource).toContain("t('overview.degraded'");
  });

  it('keeps the informational update alert static to avoid continuous redraws', () => {
    const updateAlert = overviewSource.match(
      /\{updateInfo\?\.update_available && \([\s\S]*?<\/Card>\s*\)}/,
    )?.[0];

    expect(updateAlert).toContain('<ArrowUpCircle size={18} />');
    expect(updateAlert).not.toContain('nx-spin');
  });
});
