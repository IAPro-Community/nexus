import type { HostSessionImportResult } from '../types';

export type HostSessionImportSummary = {
  imported: number;
  alreadyAuthenticated: number;
  notImported: number;
  unsupported: number;
  failed: number;
};

export function summarizeHostSessionImports(
  results: HostSessionImportResult[],
): HostSessionImportSummary {
  const summary: HostSessionImportSummary = {
    imported: 0,
    alreadyAuthenticated: 0,
    notImported: 0,
    unsupported: 0,
    failed: 0,
  };

  for (const result of results) {
    switch (result.status) {
      case 'IMPORTED':
        summary.imported += 1;
        break;
      case 'ALREADY_AUTHENTICATED':
        summary.alreadyAuthenticated += 1;
        break;
      case 'NO_SESSION':
      case 'LOGIN_REQUIRED':
        summary.notImported += 1;
        break;
      case 'UNSUPPORTED':
      case 'NOT_INSTALLED':
        summary.unsupported += 1;
        break;
      case 'FAILED':
      default:
        summary.failed += 1;
    }
  }

  return summary;
}
