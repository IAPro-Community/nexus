import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const shell = readFileSync(new URL('./NexusShell.tsx', import.meta.url), 'utf8');
const welcome = readFileSync(new URL('./modals/WelcomeModal.tsx', import.meta.url), 'utf8');

describe('Nexus UI version honesty', () => {
  it('does not claim a release version before the backend reports it', () => {
    for (const source of [shell, welcome]) {
      expect(source).not.toContain("|| '0.4.1'");
      expect(source).toContain("sysInfo?.nexus_version || 'unknown'");
    }
  });

  it('does not fabricate main as the project branch in the shell', () => {
    expect(shell).not.toContain("project.default_branch || 'main'");
    expect(shell).toContain("project.default_branch || 'unknown'");
  });
});
