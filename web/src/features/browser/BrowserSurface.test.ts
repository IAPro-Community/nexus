import { describe, expect, it } from 'vitest';
import { formatTabTitle, normalizeBrowserURL, type NexusBrowserInfo } from './BrowserSurface';

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

describe('formatTabTitle', () => {
  it('prefers explicit page title when present', () => {
    const candidate: NexusBrowserInfo = {
      id: 'session-1',
      project_id: 'p1',
      url: 'https://example.com/docs',
      title: 'Documentation · IAPro Nexus',
      width: 1280,
      height: 800,
    };
    expect(formatTabTitle(candidate)).toBe('Documentation · IAPro Nexus');
  });

  it('formats clean domain and path when title is absent', () => {
    const candidate: NexusBrowserInfo = {
      id: 'session-2',
      project_id: 'p1',
      url: 'https://github.com/IAPro-Community/nexus',
      width: 1280,
      height: 800,
    };
    expect(formatTabTitle(candidate)).toBe('github.com/IAPro-Communi');
  });

  it('formats agent label for agent-owned sessions without URL or title', () => {
    const candidate: NexusBrowserInfo = {
      id: 'session-3',
      project_id: 'p1',
      url: '',
      owner_runtime_id: 'agy-06GQFCC3DFS0TGZHM3QC',
      width: 1280,
      height: 800,
    };
    expect(formatTabTitle(candidate)).toBe('Agente · agy-06GQ');
  });

  it('falls back to session ID slice when no other identifiers exist', () => {
    const candidate: NexusBrowserInfo = {
      id: 'abcdef1234567890',
      project_id: 'p1',
      url: '',
      width: 1280,
      height: 800,
    };
    expect(formatTabTitle(candidate)).toBe('Aba · abcdef');
  });
});
