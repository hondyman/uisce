import { Component, ReactNode } from 'react';
import './ErrorBoundary.css';
import { devError } from '../utils/devLogger';

interface ErrorBoundaryState {
  hasError: boolean;
  error: any;
  errorInfo: any;
}

export class ErrorBoundary extends Component<{children: ReactNode}, ErrorBoundaryState> {
  constructor(props: {children: ReactNode}) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error: any) {
    return { hasError: true, error, errorInfo: null };
  }

  componentDidCatch(error: any, errorInfo: any) {
    // You can log error to an error reporting service here
    this.setState({ error, errorInfo });
  devError('ErrorBoundary caught an error:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="error-boundary" role="alert">
          <h2>Something went wrong.</h2>
          <pre className="error-boundary-details">
            {String(this.state.error)}
            {'\n'}
            {this.state.errorInfo && this.state.errorInfo.componentStack}
          </pre>
        </div>
      );
    }
    return this.props.children;
  }
}
