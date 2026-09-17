'use client';

import { Component, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-8 text-center">
          <h3 className="font-serif text-lg font-semibold text-red-800">Something went wrong</h3>
          <p className="mt-2 text-sm text-red-600">
            We couldn&apos;t load this content. Please try refreshing the page.
          </p>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="mt-4 rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-100"
          >
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
