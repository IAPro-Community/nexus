import { describe, expect, it, vi } from 'vitest';
import { setPlatformBridge, type PlatformBridge } from '../../platform';
import { selectProjectDirectory } from './projectDirectoryPicker';

function bridgeWithFolderPicker(
  folderPicker: boolean,
  selectedPath: string | null,
): PlatformBridge {
  return {
    kind: 'desktop',
    getCapabilities: () => ({
      native: true,
      filePicker: false,
      folderPicker,
      notifications: false,
      tray: false,
      nativeMenus: false,
      deepLinks: false,
      autoStart: false,
      windowManagement: false,
    }),
    selectDirectory: vi.fn(async () => selectedPath),
    selectFile: vi.fn(async () => null),
    showNotification: vi.fn(async () => undefined),
    openExternal: vi.fn(async () => undefined),
    getSystemTheme: vi.fn(async () => 'unknown' as const),
  };
}

describe('selectProjectDirectory', () => {
  it('does not open the HTML fallback when native folder picking is unavailable', async () => {
    const bridge = bridgeWithFolderPicker(false, null);
    setPlatformBridge(bridge);

    await expect(selectProjectDirectory('Choose project')).resolves.toEqual({
      supported: false,
      path: null,
    });
    expect(bridge.selectDirectory).not.toHaveBeenCalled();
  });

  it('delegates to the native picker and preserves cancellation', async () => {
    const bridge = bridgeWithFolderPicker(true, null);
    setPlatformBridge(bridge);

    await expect(selectProjectDirectory('Choose project')).resolves.toEqual({
      supported: true,
      path: null,
    });
    expect(bridge.selectDirectory).toHaveBeenCalledWith('Choose project');
  });
});
