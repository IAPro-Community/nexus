import { describe, expect, it } from 'vitest';
import {
  consumePtyOutputForChrome,
  extractOscTitle,
  looksLikeQuestionnaire,
  ptyWindowHeading,
} from './ptyLiveChrome';

describe('ptyLiveChrome', () => {
  it('keeps the newest OSC 0/2 settitle', () => {
    expect(extractOscTitle('\x1b]0;first\x07work\x1b]2;Question 1 of 3\x1b\\')).toBe(
      'Question 1 of 3',
    );
  });

  it('promotes live settitle over the stable identity', () => {
    expect(ptyWindowHeading({ identity: 'Shell', liveTitle: 'agy · thinking' })).toEqual({
      heading: 'agy · thinking',
      identityHint: 'Shell',
    });
  });

  it('keeps dynamic provider status updating after a manual customization', () => {
    const customized = { customTitle: 'Minha janela', identity: 'Shell' };
    expect(ptyWindowHeading({ ...customized, liveTitle: 'codex · thinking' })).toEqual({
      heading: 'codex · thinking',
      identityHint: 'Minha janela',
    });
    expect(ptyWindowHeading({ ...customized, liveTitle: 'codex · ready' })).toEqual({
      heading: 'codex · ready',
      identityHint: 'Minha janela',
    });
    expect(
      ptyWindowHeading({ ...customized, customTitle: 'Nova janela', liveTitle: 'codex · ready' }),
    ).toEqual({
      heading: 'codex · ready',
      identityHint: 'Nova janela',
    });
    expect(
      ptyWindowHeading({ ...customized, customTitle: '', liveTitle: 'codex · ready' }),
    ).toEqual({
      heading: 'codex · ready',
      identityHint: 'Shell',
    });
  });

  it('lets users choose manual title priority without losing later AI updates', () => {
    const customized = {
      customTitle: 'Backend',
      identity: 'Codex',
      allowAiTitle: false,
    };
    expect(ptyWindowHeading({ ...customized, liveTitle: 'codex · analyzing' })).toEqual({
      heading: 'Backend',
      identityHint: 'codex · analyzing',
    });
    expect(ptyWindowHeading({ ...customized, liveTitle: 'codex · ready' })).toEqual({
      heading: 'Backend',
      identityHint: 'codex · ready',
    });
    expect(ptyWindowHeading({ ...customized, allowAiTitle: true, liveTitle: 'codex · ready' })).toEqual({
      heading: 'codex · ready',
      identityHint: 'Backend',
    });
    expect(ptyWindowHeading({ ...customized, customTitle: '', liveTitle: 'codex · ready' })).toEqual({
      heading: 'codex · ready',
      identityHint: 'Codex',
    });
  });

  it('falls back to identity when the PTY has no settitle', () => {
    expect(ptyWindowHeading({ customTitle: 'Ops', identity: 'Shell' })).toEqual({
      heading: 'Ops',
      identityHint: '',
    });
  });

  it('flags AGY questionnaires and clears them on working output', () => {
    expect(looksLikeQuestionnaire('Question 1 of 2\nSelect all that apply')).toBe(true);
    const waiting = consumePtyOutputForChrome('\x1b]0;Question 1 of 2\x07', {
      title: '',
      questionnaire: false,
    });
    expect(waiting).toEqual({ title: 'Question 1 of 2', questionnaire: true });
    expect(consumePtyOutputForChrome('\x1b]0;agy · thinking\x07thinking...\n', waiting)).toEqual({
      title: 'agy · thinking',
      questionnaire: false,
    });
  });
});
