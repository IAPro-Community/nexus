import { describe, expect, it } from 'vitest';
import { isLocalTranscriptOrigin } from './externalClaudeAvailability';

describe('external Claude source availability', () => {
  it.each(['localhost', '127.0.0.1', '[::1]', '::1', 'wails.localhost', 'LOCALHOST'])(
    'permits a local host %s',
    (host) => expect(isLocalTranscriptOrigin(host)).toBe(true),
  );

  it.each(['my-tunnel.example', '192.168.1.5', 'nexus.dev', 'remote.localhost', ''])(
    'does not advertise a privileged import from remote host %s',
    (host) => expect(isLocalTranscriptOrigin(host)).toBe(false),
  );
});
