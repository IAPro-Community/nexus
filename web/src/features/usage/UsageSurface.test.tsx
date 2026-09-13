import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import React from 'react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../i18n';
import { UsageSurface } from './UsageSurface';

const listResources = vi.fn();
const refreshResources = vi.fn();
const recommendResources = vi.fn();

vi.mock('../../nexus/api', () => ({
  nexus: {
    listResources: (...args: unknown[]) => listResources(...args),
    refreshResources: (...args: unknown[]) => refreshResources(...args),
    recommendResources: (...args: unknown[]) => recommendResources(...args),
  },
}));

describe('UsageSurface responsiveness', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    listResources.mockResolvedValue({
      accounts: [
        {
          id: 'agy:demo',
          provider: 'agy',
          profile: 'demo',
          display_name: 'demo@example.com',
          available: true,
          health: 'healthy',
          quota_view: {
            status: 'LIVE',
            fetched_at: new Date().toISOString(),
            model_groups: [
              {
                key: 'claude',
                name: 'Claude',
                windows: [{ kind: '5h', remaining: 80, reset_desc: '1h' }],
              },
            ],
          },
        },
      ],
      policy: 'BALANCED',
    });
    recommendResources.mockResolvedValue({ recommended: null });
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('soft-reloads cached resources on an interval while visible', async () => {
    render(
      <I18nextProvider i18n={i18n}>
        <UsageSurface />
      </I18nextProvider>,
    );

    await vi.waitFor(() => expect(listResources).toHaveBeenCalledTimes(1));
    await screen.findByText('demo@example.com');

    await vi.advanceTimersByTimeAsync(60_000);
    await vi.waitFor(() => expect(listResources).toHaveBeenCalledTimes(2));
  });
});
