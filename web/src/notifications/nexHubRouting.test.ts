import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  isNexHubActive,
  refreshNexHubStatus,
  setNexHubStatusForTest,
  subscribeNexHubStatus,
} from './nexHubRouting';

describe('nexHubRouting', () => {
  afterEach(() => {
    setNexHubStatusForTest(null);
    vi.restoreAllMocks();
  });

  it('defaults to inactive (fallback enabled) when NEX is offline', () => {
    expect(isNexHubActive()).toBe(false);
  });

  it('updates subscribers cleanly when test override toggles active and inactive', () => {
    const seen: boolean[] = [];
    const unsubscribe = subscribeNexHubStatus((status) => {
      seen.push(status.active);
    });

    setNexHubStatusForTest({
      active: true,
      can_notify: true,
      do_not_disturb: false,
      reason: 'nex_active',
    });
    expect(isNexHubActive()).toBe(true);

    setNexHubStatusForTest({
      active: false,
      can_notify: false,
      do_not_disturb: false,
      reason: 'nex_offline',
    });
    expect(isNexHubActive()).toBe(false);
    expect(seen).toEqual([true, false]);

    unsubscribe();
  });

  it('falls back to inactive when /api/v1/nex/hub-status fails or rejects', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        active: true,
        can_notify: true,
        do_not_disturb: false,
        reason: 'nex_active',
      }),
    } as Response);

    const activeStatus = await refreshNexHubStatus();
    expect(activeStatus.active).toBe(true);
    expect(isNexHubActive()).toBe(true);

    fetchSpy.mockRejectedValueOnce(new Error('network down'));
    const fallbackStatus = await refreshNexHubStatus();
    expect(fallbackStatus.active).toBe(false);
    expect(fallbackStatus.reason).toBe('nex_offline');
    expect(isNexHubActive()).toBe(false);
  });
});
