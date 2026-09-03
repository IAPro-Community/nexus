import { describe, expect, it } from 'vitest';
import { notificationTitle } from './notificationModel';

describe('notificationTitle', () => {
  it('uses the provider/runtime dynamic title when available', () => {
    expect(notificationTitle('proxy-nginx', 'QUESTION', 'Deploy requires approval')).toBe(
      '[Nexus | proxy-nginx] Deploy requires approval',
    );
  });

  it('keeps semantic fallback titles for attention reasons', () => {
    expect(notificationTitle(undefined, 'APPROVAL')).toContain('Aprovação Necessária');
  });

  it('falls back to a semantic title when the dynamic title contains secrets', () => {
    expect(notificationTitle('Payments', 'QUESTION', 'transcript: Bearer sk-live-secret')).toBe(
      '[Nexus | Payments] ❓ Pergunta Requer Atenção',
    );
  });
});
