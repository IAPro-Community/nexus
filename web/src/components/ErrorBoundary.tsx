import React from 'react';
import i18n from 'i18next';
import styles from './ErrorBoundary.module.scss';

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

const CHUNK_RELOAD_SESSION_KEY = 'nexus:chunk-reload-attempted:v1';
type SessionStorageLike = Pick<Storage, 'getItem' | 'setItem'>;

export function isDynamicImportError(error: Error): boolean {
  return /chunkloaderror|loading chunk .* failed|failed to fetch dynamically imported module|importing a module script failed|error loading dynamically imported module/i.test(
    `${error.name}: ${error.message}`,
  );
}

export function reloadForChunkErrorOnce(
  error: Error,
  storage: SessionStorageLike,
  reload: () => void,
): boolean {
  if (!isDynamicImportError(error)) return false;

  try {
    if (storage.getItem(CHUNK_RELOAD_SESSION_KEY) === '1') return false;
    storage.setItem(CHUNK_RELOAD_SESSION_KEY, '1');
  } catch {
    return false;
  }

  reload();
  return true;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('[ErrorBoundary]', error, errorInfo);
    try {
      reloadForChunkErrorOnce(error, window.sessionStorage, () => window.location.reload());
    } catch {
      // Keep the error boundary usable when browser storage is unavailable.
    }
  }

  handleReset = () => {
    this.setState({ hasError: false, error: undefined });
  };

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback || (
          <div className={styles.errorBoundary}>
            <h2>{i18n.t('common.errorBoundaryTitle', 'Something went wrong')}</h2>
            <p>{this.state.error?.message}</p>
            <button type="button" onClick={this.handleReset}>
              {i18n.t('common.errorBoundaryTryAgain', 'Try again')}
            </button>
          </div>
        )
      );
    }
    return this.props.children;
  }
}
