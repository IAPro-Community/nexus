import { describe, expect, it, vi } from 'vitest';
import { observeTerminalPaneResize } from './terminalPaneResize';

describe('observeTerminalPaneResize', () => {
  it('fits once per animation frame when the terminal container changes size', () => {
    const element = {} as Element;
    let observerCallback: ResizeObserverCallback | undefined;
    const observe = vi.fn();
    const disconnect = vi.fn();
    const fit = vi.fn();
    const callbacks = new Map<number, FrameRequestCallback>();
    let nextFrame = 0;

    const dispose = observeTerminalPaneResize(
      element,
      fit,
      (callback) => {
        observerCallback = callback;
        return { observe, disconnect } as unknown as ResizeObserver;
      },
      {
        request: (callback) => {
          const frame = ++nextFrame;
          callbacks.set(frame, callback);
          return frame;
        },
        cancel: (frame) => callbacks.delete(frame),
      },
    );

    expect(observe).toHaveBeenCalledWith(element);
    observerCallback?.([], {} as ResizeObserver);
    observerCallback?.([], {} as ResizeObserver);
    expect(callbacks.size).toBe(1);
    expect(fit).not.toHaveBeenCalled();

    const [frame, callback] = [...callbacks.entries()][0];
    callbacks.delete(frame);
    callback(0);
    expect(fit).toHaveBeenCalledOnce();

    dispose();
    expect(disconnect).toHaveBeenCalledOnce();
  });

  it('cancels a pending fit when the terminal pane unmounts', () => {
    let observerCallback: ResizeObserverCallback | undefined;
    const callbacks = new Map<number, FrameRequestCallback>();
    const cancel = vi.fn((frame: number) => callbacks.delete(frame));
    const fit = vi.fn();

    const dispose = observeTerminalPaneResize(
      {} as Element,
      fit,
      (callback) => {
        observerCallback = callback;
        return { observe: vi.fn(), disconnect: vi.fn() } as unknown as ResizeObserver;
      },
      {
        request: (callback) => {
          callbacks.set(1, callback);
          return 1;
        },
        cancel,
      },
    );

    observerCallback?.([], {} as ResizeObserver);
    dispose();

    expect(cancel).toHaveBeenCalledWith(1);
    expect(callbacks.size).toBe(0);
    expect(fit).not.toHaveBeenCalled();
  });
});
