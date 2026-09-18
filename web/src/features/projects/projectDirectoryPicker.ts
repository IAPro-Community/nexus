import { getPlatformBridge } from '../../platform';

export interface ProjectDirectorySelection {
  supported: boolean;
  path: string | null;
}

/**
 * Uses the native picker when the host can return a real filesystem path.
 * Web mode intentionally reports unsupported so callers can show the local
 * HTML fallback instead of receiving a browser-only directory handle.
 */
export async function selectProjectDirectory(title: string): Promise<ProjectDirectorySelection> {
  const bridge = getPlatformBridge();
  if (!bridge.getCapabilities().folderPicker) {
    return { supported: false, path: null };
  }

  try {
    return { supported: true, path: await bridge.selectDirectory(title) };
  } catch {
    return { supported: false, path: null };
  }
}
