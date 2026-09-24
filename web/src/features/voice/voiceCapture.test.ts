import { describe, expect, it } from 'vitest';
import { chooseVoiceMimeType, extensionForVoiceMime } from './voiceCapture';

describe('voice capture capability selection', () => {
  it('prefers opus/webm when the runtime supports it', () => {
    expect(chooseVoiceMimeType((mime) => mime === 'audio/webm;codecs=opus')).toBe(
      'audio/webm;codecs=opus',
    );
  });

  it('falls back to a supported cross-platform container', () => {
    expect(chooseVoiceMimeType((mime) => mime === 'audio/mp4')).toBe('audio/mp4');
  });

  it('does not invent a MIME when the WebView exposes no supported preference', () => {
    expect(chooseVoiceMimeType(() => false)).toBe('');
  });

  it('maps MIME containers to upload file extensions', () => {
    expect(extensionForVoiceMime('audio/ogg;codecs=opus')).toBe('ogg');
    expect(extensionForVoiceMime('audio/mp4')).toBe('m4a');
    expect(extensionForVoiceMime('audio/webm')).toBe('webm');
  });
});
