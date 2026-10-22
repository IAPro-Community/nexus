/**
 * A local browser origin is required to request consented host transcript
 * scans. The API repeats the stronger peer/Host/Origin validation and remains
 * authoritative; this predicate only prevents a misleading remote UI action.
 */
export const isLocalTranscriptOrigin = (hostname: string): boolean => {
  const normalized = hostname.trim().toLowerCase();
  return (
    normalized === 'localhost' ||
    normalized === '127.0.0.1' ||
    normalized === '::1' ||
    normalized === '[::1]' ||
    normalized === 'wails.localhost'
  );
};
