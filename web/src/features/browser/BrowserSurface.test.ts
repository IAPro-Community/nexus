import { describe, expect, it } from 'vitest';
import {
  formatTabTitle,
  normalizeBrowserURL,
  shouldUserControlTab,
  type NexusBrowserInfo,
} from './BrowserSurface';

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

  it('converts search terms and multi-word inputs to search engine URL', () => {
    expect(normalizeBrowserURL('nexus workspace os')).toBe(
      'https://www.google.com/search?q=nexus%20workspace%20os',
    );
    expect(normalizeBrowserURL('react documentation')).toBe(
      'https://www.google.com/search?q=react%20documentation',
    );
    expect(normalizeBrowserURL('notion')).toBe(
      'https://www.google.com/search?q=notion',
    );
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

describe('shouldUserControlTab', () => {
  it('defaults to controlling human-owned tabs without owner_runtime_id', () => {
    const humanTab: NexusBrowserInfo = {
      id: 'tab-human-1',
      project_id: 'p1',
      url: 'https://example.org',
      width: 1280,
      height: 800,
    };
    const userControlledSet = new Set<string>();
    expect(shouldUserControlTab(humanTab, userControlledSet)).toBe(true);
  });

  it('defaults to observer mode (no control) for agent-owned tabs', () => {
    const agentTab: NexusBrowserInfo = {
      id: 'tab-agent-1',
      project_id: 'p1',
      url: 'https://example.org',
      owner_runtime_id: 'agent-runtime-123',
      width: 1280,
      height: 800,
    };
    const userControlledSet = new Set<string>();
    expect(shouldUserControlTab(agentTab, userControlledSet)).toBe(false);
  });

  it('preserves control when switching to an agent tab the user previously took over', () => {
    const agentTab: NexusBrowserInfo = {
      id: 'tab-agent-1',
      project_id: 'p1',
      url: 'https://example.org',
      owner_runtime_id: 'agent-runtime-123',
      width: 1280,
      height: 800,
    };
    const userControlledSet = new Set<string>(['tab-agent-1']);
    expect(shouldUserControlTab(agentTab, userControlledSet)).toBe(true);
  });

  it('keeps multi-tab permissions consistent when switching back and forth', () => {
    const humanTab: NexusBrowserInfo = {
      id: 'tab-human',
      project_id: 'p1',
      url: 'https://google.com',
      width: 1280,
      height: 800,
    };
    const agentTab: NexusBrowserInfo = {
      id: 'tab-agent',
      project_id: 'p1',
      url: 'https://cloudflare.com',
      owner_runtime_id: 'runtime-xyz',
      width: 1280,
      height: 800,
    };
    const userControlledSet = new Set<string>(['tab-human']);

    // Human tab is always controlled
    expect(shouldUserControlTab(humanTab, userControlledSet)).toBe(true);

    // Agent tab starts in observer mode
    expect(shouldUserControlTab(agentTab, userControlledSet)).toBe(false);

    // User takes over agent tab ("Assumir")
    userControlledSet.add(agentTab.id);
    expect(shouldUserControlTab(agentTab, userControlledSet)).toBe(true);

    // Switching back to human tab retains control
    expect(shouldUserControlTab(humanTab, userControlledSet)).toBe(true);

    // Switching back to agent tab retains takeover control
    expect(shouldUserControlTab(agentTab, userControlledSet)).toBe(true);

    // User releases agent tab ("Você" clicked -> release)
    userControlledSet.delete(agentTab.id);
    expect(shouldUserControlTab(agentTab, userControlledSet)).toBe(false);
    expect(shouldUserControlTab(humanTab, userControlledSet)).toBe(true);
  });
});
