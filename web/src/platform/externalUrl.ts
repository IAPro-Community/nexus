export function safeExternalUrl(raw: string): string | null {
  try {
    const parsed = new URL(raw.trim());
    if ((parsed.protocol !== 'http:' && parsed.protocol !== 'https:') || !parsed.hostname) {
      return null;
    }
    return parsed.toString();
  } catch {
    return null;
  }
}
