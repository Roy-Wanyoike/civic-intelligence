// Embed-mode helpers — issue #291.
//
// Routes under /embed/* render as standalone, chrome-less widgets so
// they can be embedded in third-party sites via <iframe>. The root
// layout (apps/web/src/app/layout.tsx) reads the marker header set by
// middleware (apps/web/src/middleware.ts) and conditionally skips the
// navbar/footer/CommandPalette/providers when rendering an embed
// route.
//
// This constant is the single source of truth for the header name —
// imported by both middleware (setter) and the layout (getter) so the
// two stay in sync without magic strings.
import { headers } from 'next/headers';

export const EMBED_HEADER = 'x-civic-embed';

/**
 * Returns true when the current request is rendering an embed route.
 *
 * Reads the marker header set by middleware. Safe to call from any
 * Server Component or Route Handler that runs inside the App Router —
 * `headers()` is the synchronous Next.js 14 API that throws only when
 * called outside a request scope (callers in the App Router are
 * always inside a request scope).
 *
 * Always returns false in non-App-Router contexts (e.g. unit tests
 * that don't go through middleware), which is the safe default — a
 * non-embed request should render the full app chrome.
 */
export function isEmbedRequest(): boolean {
  const h = headers();
  return h.get(EMBED_HEADER) === '1';
}
