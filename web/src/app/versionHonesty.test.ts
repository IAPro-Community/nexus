import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const shell = readFileSync(new URL('./NexusShell.tsx', import.meta.url), 'utf8');
const welcome = readFileSync(new URL('./modals/WelcomeModal.tsx', import.meta.url), 'utf8');
const splash = readFileSync(new URL('./NexusSplashScreen.tsx', import.meta.url), 'utf8');
const settings = readFileSync(
  new URL('../features/settings/SettingsSurface.tsx', import.meta.url),
  'utf8',
);

describe('Nexus UI version honesty', () => {
  it('does not claim a release version before the backend reports it', () => {
    for (const source of [shell, welcome]) {
      expect(source).not.toContain("|| '0.4.1'");
      expect(source).toContain("sysInfo?.nexus_version || 'unknown'");
    }
  });

  it('surfaces signed Nexus update availability separately from Maestro updates', () => {
    expect(shell).toContain('sysInfo?.nexus_update_available || sysInfo?.update_available');
    expect(shell).toContain('updateAvailable={hasAvailableUpdates}');
    expect(shell).toContain('{hasAvailableUpdates && (');
    expect(shell).toContain("window.addEventListener('focus', load)");
    expect(shell).toContain('window.clearInterval(refreshInterval)');
  });

  it('keeps the available-update indicator static while it awaits user action', () => {
    const indicator = shell.match(/className="nx-update-indicator"[\s\S]*?<\/button>/)?.[0];

    expect(indicator).toContain('<ArrowUpCircle size={13} />');
    expect(indicator).not.toContain('nx-spin');
  });

  it('does not show a hardcoded release version in the pre-backend splash screen', () => {
    expect(splash).not.toMatch(/v\d+\.\d+\.\d+(?:-[\w.]+)?/);
    expect(splash).toContain('NexusSplashScreen');
  });

  it('never presents Maestro maintenance as an in-place Nexus update', () => {
    expect(settings).toContain('Baixar instalador oficial do Nexus');
    expect(settings).toContain('Ver release oficial do Nexus');
    expect(settings).toContain("'Atualizar Maestro'");
    expect(settings).toContain('getPlatformBridge().openExternal(url)');
    expect(settings).toContain('nexus_installer_url || updateInfo.nexus_release_url');
    expect(settings).not.toContain("'Atualizar Nexus automaticamente'");
  });

  it('does not fabricate main as the project branch in the shell', () => {
    expect(shell).not.toContain("project.default_branch || 'main'");
    expect(shell).toContain("project.default_branch || 'unknown'");
  });
});
