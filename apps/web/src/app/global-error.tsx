'use client';

/**
 * Global error boundary (Next.js 14 App Router).
 *
 * This is the LAST-RESORT error boundary. It catches errors thrown by
 * `app/layout.tsx` itself — errors that `app/error.tsx` cannot catch
 * because the layout is the error boundary's parent.
 *
 * Unlike `app/error.tsx`, this boundary MUST render its own `<html>`
 * and `<body>` tags because the root layout is bypassed entirely when
 * this boundary activates. If you forget the `<html>` + `<body>`, the
 * page renders as a fragment and CSS/JS won't load.
 *
 * Common triggers:
 *
 *   - `next-intl` `getMessages()` / `getLocale()` throws (e.g. the
 *     `civic-locale` cookie is malformed or the locale JSON file is
 *     missing on the serverless fs)
 *   - `cookies()` from `next/headers` throws (very rare, but possible
 *     if the request is malformed)
 *   - A module imported by `layout.tsx` throws at import time
 *   - The `GovernmentProvider` `initialSelection` somehow throws
 *
 * This boundary renders a minimal HTML shell so the user at least sees
 * a friendly error UI instead of a blank page or the bare Next.js
 * "Application error" message.
 */

import { useEffect } from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.error('[app/global-error.tsx] Uncaught root error:', error);
  }, [error]);

  const isDev = process.env.NODE_ENV === 'development';

  return (
    <html lang="en">
      <body
        style={{
          margin: 0,
          padding: 0,
          fontFamily:
            'ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
          background: '#fff7f0',
          color: '#7c2d12',
        }}
      >
        <div
          style={{
            maxWidth: '40rem',
            margin: '0 auto',
            padding: '4rem 1rem',
          }}
        >
          <div
            style={{
              border: '1px solid #fca5a5',
              background: '#fff',
              borderRadius: '0.75rem',
              padding: '1.5rem',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <AlertTriangle size={20} aria-hidden="true" />
              <h1 style={{ margin: 0, fontSize: '1.25rem', fontWeight: 600 }}>
                The site could not be loaded
              </h1>
            </div>
            <p style={{ marginTop: '0.5rem', fontSize: '0.875rem' }}>
              A critical error occurred while loading the page shell. This is
              usually transient — please try again in a moment. If the
              problem persists, the platform&apos;s data API may be down.
            </p>

            <div
              style={{
                marginTop: '1rem',
                padding: '0.75rem',
                background: '#fef3c7',
                borderRadius: '0.375rem',
                fontSize: '0.75rem',
                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
              }}
            >
              <div style={{ fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                Error details
              </div>
              <p style={{ marginTop: '0.375rem', wordBreak: 'break-word' }}>
                {error.message || 'Unknown error'}
              </p>
              {error.digest && (
                <p style={{ marginTop: '0.25rem', wordBreak: 'break-all', color: '#92400e' }}>
                  Digest: {error.digest}
                </p>
              )}
              {isDev && error.stack && (
                <pre
                  style={{
                    marginTop: '0.5rem',
                    maxHeight: '12rem',
                    overflow: 'auto',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-all',
                    background: '#fde68a',
                    padding: '0.5rem',
                    borderRadius: '0.25rem',
                    fontSize: '0.6875rem',
                    lineHeight: 1.2,
                  }}
                >
                  {error.stack}
                </pre>
              )}
            </div>

            <button
              type="button"
              onClick={() => reset()}
              style={{
                marginTop: '1.25rem',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '0.5rem',
                background: '#7c2d12',
                color: '#fff',
                border: 'none',
                borderRadius: '0.375rem',
                padding: '0.5rem 1rem',
                fontSize: '0.875rem',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              <RefreshCw size={16} aria-hidden="true" />
              Try again
            </button>
          </div>
        </div>
      </body>
    </html>
  );
}
