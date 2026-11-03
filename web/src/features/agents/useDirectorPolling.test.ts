import { describe, expect, it } from 'vitest';
import { DIRECTOR_POLL_INTERVAL_MS, shouldPollDirector } from './useDirectorPolling';

describe('Director visibility-scoped polling', () => {
  it('does not poll hidden tabs or while a request is pending', () => {
    expect(shouldPollDirector('hidden', false, 100, 100_000)).toBe(false);
    expect(shouldPollDirector('visible', true, 100, 100_000)).toBe(false);
  });

  it('honors the minimum interval and avoids premature retries', () => {
    expect(shouldPollDirector('visible', false, 100, 100 + DIRECTOR_POLL_INTERVAL_MS - 1)).toBe(false);
    expect(shouldPollDirector('visible', false, 100, 100 + DIRECTOR_POLL_INTERVAL_MS)).toBe(true);
  });

  it('can refresh after a suspended tab becomes visible', () => {
    expect(shouldPollDirector('visible', false, 0, DIRECTOR_POLL_INTERVAL_MS * 3)).toBe(true);
  });
});
