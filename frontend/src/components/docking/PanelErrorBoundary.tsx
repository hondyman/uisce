import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  panelTitle?: string;
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class PanelErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error(`[PanelErrorBoundary: ${this.props.panelTitle || 'Panel'}] caught error:`, error, errorInfo);
  }

  private handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  public render() {
    if (this.state.hasError) {
      return (
        <div style={{
          padding: '24px',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: '#050d1a',
          color: '#f87171',
          fontFamily: 'monospace',
          textAlign: 'center',
        }}>
          <div style={{ fontSize: '18px', fontWeight: 600, marginBottom: '8px' }}>
            Panel Error: {this.props.panelTitle || 'Trading View'}
          </div>
          <div style={{ fontSize: '12px', color: '#94a3b8', maxWidth: '400px', marginBottom: '16px' }}>
            {this.state.error?.message || 'An unexpected runtime error occurred in this panel.'}
          </div>
          <button
            onClick={this.handleReset}
            style={{
              padding: '6px 16px',
              borderRadius: '4px',
              border: '1px solid #38bdf8',
              background: '#0284c7',
              color: '#ffffff',
              fontSize: '12px',
              cursor: 'pointer',
            }}
          >
            Reload Panel
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
