import { describe, expect, it } from 'vitest';
import { normalizeLanguage } from './index';
import { en, es, ptBR } from './resources';

const keys = (value: unknown, prefix = ''): string[] =>
  Object.entries(value as Record<string, unknown>)
    .flatMap(([key, child]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      return child && typeof child === 'object' ? keys(child, path) : [path];
    })
    .sort();

describe('i18n', () => {
  it.each([
    ['pt', 'pt-BR'],
    ['pt_PT', 'pt-BR'],
    ['es-MX', 'es'],
    ['en-US', 'en'],
    ['de-DE', 'en'],
    ['', 'en'],
  ])('normalizes %s', (input, expected) => expect(normalizeLanguage(input)).toBe(expected));
  it('keeps all catalogs in parity', () => {
    expect(keys(ptBR)).toEqual(keys(en));
    expect(keys(es)).toEqual(keys(en));
  });
  it('localizes every attention radar tab state', () => {
    expect(en.notifications.radarTab).toBe('Attention Radar');
    expect(ptBR.notifications.radarTab).toBe('Radar de Atenção');
    expect(es.notifications.radarTab).toBe('Radar de Atención');
    expect(ptBR.notifications.radarState.needs_user).toBe('Precisa de você');
    expect(ptBR.notifications.radarState.working).toBe('Trabalhando');
    expect(ptBR.notifications.radarState.completed).toBe('Concluído');
    expect(ptBR.notifications.radarState.error).toBe('Erro');
    expect(ptBR.notifications.radarState.idle).toBe('Ocioso');
  });
});
