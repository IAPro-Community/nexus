import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./WorkspaceTaskbar.tsx', import.meta.url), 'utf8');

describe('Workspace taskbar version honesty', () => {
  it('does not fabricate Nexus versions before system info is loaded', () => {
    expect(source).not.toContain("|| '0.4.1'");
    expect(source).not.toContain("|| '0.1.25'");
    expect(source).toContain("sysInfo?.nexus_version || 'unknown'");
    expect(source).not.toContain('Maestro</strong>');
  });

  it('does not fabricate main as the active branch when branch data is unknown', () => {
    expect(source).not.toContain('currentBranch || project.default_branch');
    expect(source).toContain("currentBranch || t('git.branchUnknown'");
    expect(source).toContain("result.current_branch || ''");
  });
});
