import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Uncaught error:", error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      return (
        <article className="pico-background-red-200" style={{ margin: '20px', textAlign: 'center' }}>
          <h3>⚠️ Something went wrong</h3>
          <p>{this.state.error?.message}</p>
          <button onClick={() => window.location.reload()}>Reload Application</button>
        </article>
      );
    }

    return this.props.children;
  }
}
