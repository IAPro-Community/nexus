import { describe, expect, it } from 'vitest';
import { launcherActionTarget, workspaceLauncherActions } from './launcherModel';

describe('workspace launcher', () => {
  it('exposes only core-first workspace actions', () => {
    expect(workspaceLauncherActions.map((item) => item.action)).toEqual(['NEW_AGENT', 'NEW_TERMINAL', 'OPEN_COMPOSER', 'OPEN_ACTIVE_FLOW_RUN']);
    expect(workspaceLauncherActions.map((item) => item.label).join(' ')).not.toMatch(/Mission|Assisted|Planned/);
  });

  it('maps actions to stable surface targets', () => {
    expect(launcherActionTarget('NEW_AGENT')).toBe('agents');
    expect(launcherActionTarget('NEW_TERMINAL')).toBe('project-shell');
    expect(launcherActionTarget('OPEN_COMPOSER')).toBe('work');
    expect(launcherActionTarget('OPEN_ACTIVE_FLOW_RUN')).toBe('legacy-runtimes');
  });
});
