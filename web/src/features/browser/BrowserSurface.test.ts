import { describe, expect, it } from 'vitest';
import { normalizeBrowserURL } from './BrowserSurface';

describe('normalizeBrowserURL', () => {
  it('keeps existing http and https schemes intact', () => {
    expect(normalizeBrowserURL('http://localhost:3000')).toBe('http://localhost:3000');
    expect(normalizeBrowserURL('https://example.com/page')).toBe('https://example.com/page');
  });

  it('normalizes local addresses without scheme to http://', () => {
    expect(normalizeBrowserURL('localhost:3000')).toBe('http://localhost:3000');
    expect(normalizeBrowserURL('localhost:3000/api')).toBe('http://localhost:3000/api');
    expect(normalizeBrowserURL('127.0.0.1:8080')).toBe('http://127.0.0.1:8080');
    expect(normalizeBrowserURL('0.0.0.0:4000')).toBe('http://0.0.0.0:4000');
    expect(normalizeBrowserURL('[::1]:3000')).toBe('http://[::1]:3000');
  });

  it('normalizes external domains without scheme to https://', () => {
    expect(normalizeBrowserURL('google.com')).toBe('https://google.com');
    expect(normalizeBrowserURL('github.com/org/repo')).toBe('https://github.com/org/repo');
    expect(normalizeBrowserURL('//example.org')).toBe('https://example.org');
  });

  it('handles empty or whitespace strings', () => {
    expect(normalizeBrowserURL('')).toBe('');
    expect(normalizeBrowserURL('   ')).toBe('');
  });
});
