import { getPlatformBridge, type PlatformBridge } from '../../platform/platformBridge';

export const preferredVoiceMimeTypes = [
  'audio/webm;codecs=opus',
  'audio/webm',
  'audio/ogg;codecs=opus',
  'audio/ogg',
  'audio/mp4',
] as const;

export type VoiceRecording = {
  blob: Blob;
  mimeType: string;
  durationMs: number;
};

export type VoiceCaptureSession = {
  stop: () => Promise<VoiceRecording>;
  cancel: () => void;
};

export const chooseVoiceMimeType = (
  supported: (mime: string) => boolean = (mime) =>
    typeof MediaRecorder !== 'undefined' && MediaRecorder.isTypeSupported(mime),
): string => preferredVoiceMimeTypes.find((mime) => supported(mime)) || '';

const supportsWebVoiceCapture = () =>
  typeof navigator !== 'undefined' &&
  Boolean(navigator.mediaDevices?.getUserMedia) &&
  typeof MediaRecorder !== 'undefined';

const nativeVoiceBridge = (): PlatformBridge | null => {
  try {
    const bridge = getPlatformBridge();
    const capabilities = bridge.getCapabilities();
    if (
      bridge.kind === 'desktop' &&
      capabilities.nativeVoiceCapture &&
      bridge.startNativeVoiceCapture &&
      bridge.stopNativeVoiceCapture
    ) {
      return bridge;
    }
  } catch {
    // Platform bridge may not be initialized during early bootstrap/tests.
  }
  return null;
};

export const supportsVoiceCapture = () =>
  supportsWebVoiceCapture() || Boolean(nativeVoiceBridge());

const base64ToBlob = (dataBase64: string, mimeType: string) => {
  const binary = atob(dataBase64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return new Blob([bytes], { type: mimeType });
};

const startNativeVoiceCapture = async (): Promise<VoiceCaptureSession> => {
  const bridge = nativeVoiceBridge();
  const start = bridge?.startNativeVoiceCapture;
  const stop = bridge?.stopNativeVoiceCapture;
  if (!bridge || !start || !stop) {
    throw new Error('Captura nativa de microfone indisponível.');
  }

  await start.call(bridge);
  let done = false;

  return {
    stop: async () => {
      if (done) throw new Error('A captura de voz já foi encerrada.');
      done = true;
      const recording = await stop.call(bridge);
      const mimeType = recording.mimeType || 'audio/wav';
      return {
        blob: base64ToBlob(recording.dataBase64, mimeType),
        mimeType,
        durationMs: recording.durationMs,
      };
    },
    cancel: () => {
      if (done) return;
      done = true;
      void bridge.cancelNativeVoiceCapture?.();
    },
  };
};

const startWebVoiceCapture = async (): Promise<VoiceCaptureSession> => {
  const stream = await navigator.mediaDevices.getUserMedia({
    audio: {
      echoCancellation: true,
      noiseSuppression: true,
      autoGainControl: true,
    },
  });
  const mimeType = chooseVoiceMimeType();
  let recorder: MediaRecorder;
  try {
    recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined);
  } catch (error) {
    stream.getTracks().forEach((track) => track.stop());
    throw error;
  }

  const chunks: BlobPart[] = [];
  const startedAt = performance.now();
  let settled = false;
  let stopPromise: Promise<VoiceRecording> | null = null;

  recorder.addEventListener('dataavailable', (event) => {
    if (event.data.size > 0) chunks.push(event.data);
  });
  recorder.start(250);

  const cleanup = () => stream.getTracks().forEach((track) => track.stop());

  const stop = () => {
    if (stopPromise) return stopPromise;
    stopPromise = new Promise<VoiceRecording>((resolve, reject) => {
      if (settled) {
        reject(new Error('A captura de voz já foi encerrada.'));
        return;
      }

      const finish = () => {
        if (settled) return;
        settled = true;
        cleanup();
        const actualMime = recorder.mimeType || mimeType || 'application/octet-stream';
        resolve({
          blob: new Blob(chunks, { type: actualMime }),
          mimeType: actualMime,
          durationMs: Math.max(1, Math.round(performance.now() - startedAt)),
        });
      };

      recorder.addEventListener('stop', finish, { once: true });
      recorder.addEventListener(
        'error',
        () => {
          if (settled) return;
          settled = true;
          cleanup();
          reject(new Error('Falha durante a captura do microfone.'));
        },
        { once: true },
      );

      if (recorder.state === 'inactive') finish();
      else recorder.stop();
    });
    return stopPromise;
  };

  const cancel = () => {
    if (settled) return;
    settled = true;
    try {
      if (recorder.state !== 'inactive') recorder.stop();
    } catch {
      // best effort
    }
    cleanup();
  };

  return { stop, cancel };
};

export async function startVoiceCapture(): Promise<VoiceCaptureSession> {
  let webError: unknown;
  if (supportsWebVoiceCapture()) {
    try {
      return await startWebVoiceCapture();
    } catch (error) {
      webError = error;
    }
  }

  if (nativeVoiceBridge()) {
    return await startNativeVoiceCapture();
  }

  if (webError instanceof Error) throw webError;
  throw new Error('Captura de microfone não é suportada neste navegador/WebView.');
}

export const extensionForVoiceMime = (mime: string) => {
  const normalized = mime.toLowerCase();
  if (normalized.includes('ogg')) return 'ogg';
  if (normalized.includes('mp4')) return 'm4a';
  if (normalized.includes('mpeg')) return 'mp3';
  if (normalized.includes('wav')) return 'wav';
  return 'webm';
};
