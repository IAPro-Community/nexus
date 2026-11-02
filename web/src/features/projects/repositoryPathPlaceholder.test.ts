import { describe, expect, it } from 'vitest';
import { getRepositoryPathPlaceholderKey } from './repositoryPathPlaceholder';

describe('getRepositoryPathPlaceholderKey', () => {
  it('selects a Windows repository path for Windows browsers and WebView2', () => {
    expect(getRepositoryPathPlaceholderKey('Mozilla/5.0 (Windows NT 10.0; Win64; x64)')).toBe(
      'projects.repositoryPathPlaceholderWindows',
    );
  });

  it('selects a macOS repository path for Mac browsers', () => {
    expect(getRepositoryPathPlaceholderKey('Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0)')).toBe(
      'projects.repositoryPathPlaceholderMac',
    );
  });

  it('uses a POSIX repository path for Linux and unknown browsers', () => {
    expect(getRepositoryPathPlaceholderKey('Mozilla/5.0 (X11; Linux x86_64)')).toBe(
      'projects.repositoryPathPlaceholderPosix',
    );
    expect(getRepositoryPathPlaceholderKey('')).toBe('projects.repositoryPathPlaceholderPosix');
  });
});
