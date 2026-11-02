export type RepositoryPathPlaceholderKey =
  | 'projects.repositoryPathPlaceholderWindows'
  | 'projects.repositoryPathPlaceholderMac'
  | 'projects.repositoryPathPlaceholderPosix';

export function getRepositoryPathPlaceholderKey(userAgent: string): RepositoryPathPlaceholderKey {
  const normalizedUserAgent = userAgent.toLowerCase();

  if (normalizedUserAgent.includes('windows')) {
    return 'projects.repositoryPathPlaceholderWindows';
  }

  if (normalizedUserAgent.includes('macintosh') || normalizedUserAgent.includes('mac os')) {
    return 'projects.repositoryPathPlaceholderMac';
  }

  return 'projects.repositoryPathPlaceholderPosix';
}
