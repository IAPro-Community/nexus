import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const settings = readFileSync(
  new URL('../features/settings/SettingsSurface.module.scss', import.meta.url),
  'utf8',
);
const gate = readFileSync(new URL('./AccessCodeGate.module.scss', import.meta.url), 'utf8');

/** The phone pass must cover a 390px viewport, the narrowest supported handset. */
describe('remote access at 390px', () => {
  it('stacks pairing controls instead of overflowing the card', () => {
    const block = settings.slice(settings.indexOf('@media (max-width: 420px)'));
    expect(block).not.toBe('');
    expect(block).toMatch(/\.tunnelActions\s*\{[^}]*flex-direction:\s*column/s);
    expect(block).toMatch(/\.accessCodeRow\s*\{[^}]*flex-direction:\s*column/s);
    expect(block).toMatch(/\.qrContainer\s*\{[^}]*flex-direction:\s*column/s);
  });

  it('keeps the pairing input above the iOS zoom threshold', () => {
    const block = gate.slice(gate.indexOf('@media (max-width: 420px)'));
    expect(block).not.toBe('');
    // 1.125rem == 18px at the default root size, above Safari's 16px trigger.
    expect(block).toMatch(/\.input\s*\{[^}]*font-size:\s*1\.125rem/s);
    expect(block).toMatch(/\.form\s*\{[^}]*max-width:\s*100%/s);
  });
});
