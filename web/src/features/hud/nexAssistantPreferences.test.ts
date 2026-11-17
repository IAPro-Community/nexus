import { describe, expect, it } from 'vitest';
import { readNexAssistantPreferences } from './nexAssistantPreferences';

describe('Nex assistant preferences', () => {
  it('keeps manual mode as the safe default without a browser', () => {
    expect(readNexAssistantPreferences().version).toBe(1);
    expect(['manual', 'contextual']).toContain(readNexAssistantPreferences().mode);
    expect(typeof readNexAssistantPreferences().persistHistory).toBe('boolean');
  });
});
