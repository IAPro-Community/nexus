import { getDesktopAuthToken } from '../api';

/**
 * Authenticated binary export. Unlike a plain anchor, this works in Wails
 * desktop mode where the Core API requires a Bearer token.
 */
export async function downloadAuthenticatedCsv(path: string, filename: string): Promise<void> {
  if (!path.startsWith('/api/v1/')) {
    throw new Error('Invalid CSV endpoint');
  }
  const headers = new Headers({ Accept: 'text/csv' });
  const desktopToken = getDesktopAuthToken();
  if (desktopToken) {
    headers.set('Authorization', `Bearer ${desktopToken}`);
  }
  const response = await fetch(path, { method: 'GET', headers, credentials: 'same-origin' });
  if (!response.ok) {
    throw new Error(`Falha ao exportar CSV (HTTP ${response.status}).`);
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  try {
    anchor.click();
  } finally {
    anchor.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
}
