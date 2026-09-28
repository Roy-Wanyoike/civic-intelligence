// Embed-mode helpers — issue #291.
//
// Routes under /embed/* render as standalone, chrome-less widgets so
// they can be embedded in third-party sites via <iframe>. The root
// layout reads the request's pathname to detect embed routes — no
// middleware needed (middleware was removed because Vercel's
// multi-service "services" config does not support Edge Functions,
// which is the runtime Next.js 14 middleware always uses).
//
// In Next.js 14 App Router, the `headers()` function exposes the
// `x-invoke-path` header (set by the Next.js server internally) which
// contains the full pathname. We use this to detect /embed/* routes.
//
// Always returns false in non-App-Router contexts (e.g. unit tests),
// which is the safe default — a non-embed request should render the
// full app chrome.
import { headers } from 'next/headers';

/**
 * Returns true when the current request is rendering an embed route.
 *
 * Reads the `x-invoke-path` header (set by Next.js internally) to
 * check if the pathname starts with /embed. This replaces the previous
 * middleware-based approach which set a custom header — middleware
 * was removed because Vercel's multi-service config doesn't support
 * Edge Functions.
 */
export function isEmbedRequest(): boolean {
  const h = headers();
  const pathname = h.get('x-invoke-path') ?? '';
  return pathname.startsWith('/embed');
}
