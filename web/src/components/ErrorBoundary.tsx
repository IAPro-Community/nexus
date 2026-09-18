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
