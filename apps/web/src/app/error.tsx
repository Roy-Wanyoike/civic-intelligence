'use client';

/**
 * Root error boundary (Next.js 14 App Router).
 *
 * This file catches ANY uncaught exception thrown by a server component
 * (or a client component) below `app/layout.tsx`. Without it, Next.js
 * renders the bare "Application error: a server-side exception has
 * occurred. Digest: …" page — which gives the user zero context and
 * gives us no way to surface the underlying error.
 *
 * The boundary is a CLIENT component (required by Next.js). It receives
 * two props:
 *
 *   - `error`  — the thrown Error. We surface `error.message` so the
 *                user (and our logs) can see what actually failed,
 *                instead of just the opaque Digest hash.
 *   - `reset`  — a function that re-renders the error boundary's
 *                children. Useful when the error was transient (e.g.
 *                the BFF was briefly unreachable).
 *
 * What this boundary does NOT do:
 *
 *   - Report to Sentry / Datadog. The platform does not yet wire up
 *     an error-reporting SDK; when it does, hook it here + in
 *     `global-error.tsx`. For now, `console.error` is the only sink.
 *   - Render a "wait + auto-retry" UX. We could debounce `reset()` —
 *     the user clicking the "Try again" button is enough for now.
 *
 * Related files:
 *   - `app/global-error.tsx` — catches errors thrown by `layout.tsx`
 *     itself (this boundary can't, because the layout is its parent).
 *   - `app/not-found.tsx` — renders the 404 UI (separate code path;
 *     not an error, just a "no such route" response).
 */

import { useEffect } from 'react';
import Link from 'next/link';
import { AlertTriangle, RefreshCw, Home, Bug } from 'lucide-react';

export default function RootErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  // Log the error to the browser console so it shows up alongside the
  // server logs when debugging. In production we redact the stack (which
  // can leak file paths / env var names) but keep the message + digest
  // so the user can copy-paste them into a support request.
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.error('[app/error.tsx] Uncaught SSR/runtime error:', error);
  }, [error]);

  const isDev = process.env.NODE_ENV === 'development';

  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
      <div className="rounded-xl border border-red-300 bg-red-50 p-6 text-red-900">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-5 w-5 flex-shrink-0" aria-hidden="true" />
          <h1 className="font-serif text-xl font-semibold">
            Something went wrong
          </h1>
        </div>
        <p className="mt-2 text-sm text-red-800">
          The page failed to render. This is usually transient — the
          live data API may be briefly unreachable, or a network blip
          interrupted the request. Try again in a moment.
        </p>

        {/* Surface the actual error message + digest so the user can
            copy them into a support request. Hides the stack trace in
            production to avoid leaking file paths / env var names. */}
        <div className="mt-4 rounded-md bg-white/60 p-3 text-xs text-red-900">
          <div className="flex items-center gap-1.5 font-semibold uppercase tracking-wide">
            <Bug className="h-3.5 w-3.5" aria-hidden="true" />
            <span>Error details</span>
          </div>
          <p className="mt-1.5 break-words font-mono">
            {error.message || 'Unknown error'}
          </p>
          {error.digest && (
            <p className="mt-1 break-all font-mono text-red-700">
              Digest: {error.digest}
            </p>
          )}
          {isDev && error.stack && (
            <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all rounded bg-red-100 p-2 text-[11px] leading-tight">
              {error.stack}
            </pre>
          )}
        </div>

        <div className="mt-5 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => reset()}
            className="inline-flex items-center gap-2 rounded-md bg-red-900 px-4 py-2 text-sm font-semibold text-white transition hover:bg-red-800"
          >
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            Try again
          </button>
          <Link
            href="/"
            className="inline-flex items-center gap-2 rounded-md border border-red-300 bg-white px-4 py-2 text-sm font-semibold text-red-900 transition hover:bg-red-100"
          >
            <Home className="h-4 w-4" aria-hidden="true" />
            Go home
          </Link>
        </div>
      </div>
    </div>
  );
}
