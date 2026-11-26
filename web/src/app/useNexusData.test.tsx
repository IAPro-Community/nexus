// @vitest-environment jsdom
import React, { act } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

declare global {
  // React only batches updates inside act() when the flag is exposed on the
  // global object; Vitest's jsdom environment does not set it for us.
  var IS_REACT_ACT_ENVIRONMENT: boolean;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
import { api } from '../api';
import { nexus } from '../nexus/api';
import { FAST_POLL_MS, METADATA_POLL_MS, useNexusData } from './useNexusData';

const renderHook = <T,>(hook: () => T) => {
  const container = document.createElement('div');
  document.body.append(container);
  const root = createRoot(container);
  const result: { current: T } = { current: undefined as T };
  const Probe = () => {
    result.current = hook();
    return null;
  };
  act(() => root.render(React.createElement(Probe)));
  return {
    result,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

const settle = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const wait = async (ms: number) => {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
};

describe('useNexusData polling lanes', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(nexus, 'listProjects').mockResolvedValue([]);
    vi.spyOn(api, 'getWorkspaces').mockResolvedValue([]);
    vi.spyOn(api, 'getRuntimes').mockResolvedValue(null as never);
    vi.spyOn(api, 'getProviders').mockResolvedValue([]);
    vi.spyOn(api, 'getProfiles').mockResolvedValue([]);
    vi.spyOn(api, 'getEvents').mockResolvedValue(null as never);
    Object.defineProperty(document, 'hidden', { configurable: true, value: false });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('keeps metadata on the slow cadence while runtime state polls on the fast one', async () => {
    const hook = renderHook(useNexusData);
    await settle();
    expect(nexus.listProjects).toHaveBeenCalledTimes(1);
    expect(api.getRuntimes).toHaveBeenCalledTimes(1);

    await wait(FAST_POLL_MS);
    expect(api.getRuntimes).toHaveBeenCalledTimes(2);
    expect(api.getEvents).toHaveBeenCalledTimes(2);
    expect(nexus.listProjects).toHaveBeenCalledTimes(1);
    expect(api.getWorkspaces).toHaveBeenCalledTimes(1);
    expect(api.getProviders).toHaveBeenCalledTimes(1);
    expect(api.getProfiles).toHaveBeenCalledTimes(1);

    await wait(FAST_POLL_MS);
    expect(api.getRuntimes).toHaveBeenCalledTimes(3);
    expect(nexus.listProjects).toHaveBeenCalledTimes(1);

    // 15s: the metadata lane still has not fired a second time.
    await wait(METADATA_POLL_MS - 2 * FAST_POLL_MS);
    expect(api.getRuntimes).toHaveBeenCalledTimes(7);
    expect(api.getWorkspaces).toHaveBeenCalledTimes(2);
    expect(nexus.listProjects).toHaveBeenCalledTimes(2);
    hook.unmount();
  });

  it('pauses both lanes while the document is hidden and refreshes when visible again', async () => {
    const hook = renderHook(useNexusData);
    await settle();
    Object.defineProperty(document, 'hidden', { configurable: true, value: true });
    // Ticks registered before the visibility flip still fire; start counting
    // after the next fast cadence so both lanes are provably idle while hidden.
    await wait(FAST_POLL_MS);
    const initialRuntimes = vi.mocked(api.getRuntimes).mock.calls.length;
    const initialProjects = vi.mocked(nexus.listProjects).mock.calls.length;

    await wait(METADATA_POLL_MS * 2);
    expect(api.getRuntimes).toHaveBeenCalledTimes(initialRuntimes);
    expect(nexus.listProjects).toHaveBeenCalledTimes(initialProjects);

    Object.defineProperty(document, 'hidden', { configurable: true, value: false });
    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
      await Promise.resolve();
    });
    expect(api.getRuntimes).toHaveBeenCalledTimes(initialRuntimes + 1);
    expect(nexus.listProjects).toHaveBeenCalledTimes(initialProjects + 1);
    hook.unmount();
  });

  it('refreshGlobal refreshes both lanes and returns the runtime snapshot', async () => {
    const hook = renderHook(useNexusData);
    await settle();
    vi.mocked(api.getRuntimes).mockClear();
    vi.mocked(nexus.listProjects).mockClear();

    let snapshot: { runtimes: unknown[] } | undefined;
    await act(async () => {
      snapshot = await hook.result.current.refreshGlobal();
    });
    expect(api.getRuntimes).toHaveBeenCalledTimes(1);
    expect(nexus.listProjects).toHaveBeenCalledTimes(1);
    // Go encodes empty slices as null; the hook must normalize to [].
    expect(snapshot).toEqual({ runtimes: [] });
    expect(hook.result.current.runtimes).toEqual([]);
    hook.unmount();
  });
});
