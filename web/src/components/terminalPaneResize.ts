type ResizeObserverHandle = Pick<ResizeObserver, 'observe' | 'disconnect'>;
type ResizeObserverFactory = (callback: ResizeObserverCallback) => ResizeObserverHandle;

interface AnimationFrameScheduler {
  request: (callback: FrameRequestCallback) => number;
  cancel: (frame: number) => void;
}

export function observeTerminalPaneResize(
  element: Element,
  onResize: () => void,
  observerFactory?: ResizeObserverFactory,
  frameScheduler?: AnimationFrameScheduler,
): () => void {
  if (!observerFactory && typeof ResizeObserver === 'undefined') return () => {};

  const createObserver = observerFactory ?? ((callback) => new ResizeObserver(callback));
  const frames = frameScheduler ?? {
    request: (callback: FrameRequestCallback) => window.requestAnimationFrame(callback),
    cancel: (frame: number) => window.cancelAnimationFrame(frame),
  };
  let pendingFrame: number | undefined;
  const observer = createObserver(() => {
    if (pendingFrame !== undefined) return;
    pendingFrame = frames.request(() => {
      pendingFrame = undefined;
      onResize();
    });
  });

  observer.observe(element);

  return () => {
    observer.disconnect();
    if (pendingFrame === undefined) return;
    frames.cancel(pendingFrame);
    pendingFrame = undefined;
  };
}
