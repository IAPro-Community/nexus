/**
 * Formats an 8-character remote access code for display, grouping it as
 * ABCD-EFGH. Input is trimmed and uppercased; unexpected lengths are returned
 * unchanged so the user always sees exactly what the server minted.
 */
export function formatAccessCode(code: string): string {
  const clean = code.trim().toUpperCase();
  if (clean.length !== 8) return clean;
  return `${clean.slice(0, 4)}-${clean.slice(4)}`;
}
