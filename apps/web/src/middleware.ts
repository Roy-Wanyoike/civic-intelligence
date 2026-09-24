import { NextResponse, type NextRequest } from 'next/server';

// Country subdomain detection — ke.civicintelligence.com → country=KE
const COUNTRY_MAP: Record<string, string> = {
  ke: 'KE',
  ug: 'UG',
  tz: 'TZ',
  gh: 'GH',
  ng: 'NG',
  za: 'ZA',
};

// Embed-mode marker — issue #291. Routes under /embed/* render as
// standalone, chrome-less widgets (no navbar/footer/CommandPalette/
// service worker) so they can be embedded in third-party sites via
// <iframe>. The root layout reads this request header via `headers()`
// to decide whether to wrap the children in the normal app chrome or
// render them bare. We use a request header (rather than reading the
// URL inside the layout) because Next.js App Router layouts don't
// receive the pathname directly — middleware is the only place that
// sees the URL before the route renders.
const EMBED_HEADER = 'x-civic-embed';

export function middleware(request: NextRequest) {
  const isEmbed = request.nextUrl.pathname.startsWith('/embed');

  // Mark embed routes so the root layout can skip the navbar/footer.
  // We forward the marker as a request header — Next.js middleware
  // propagates request headers set via `NextResponse.next({ request })`
  // to the downstream Server Component, where `headers()` reads them
  // (https://nextjs.org/docs/app/api-reference/functions/headers).
  // Setting `request.headers.set(...)` directly on the incoming
  // request DOES NOT propagate — only the `request` field on the
  // `NextResponse.next()` argument does.
  const forwarded = new Headers(request.headers);
  if (isEmbed) forwarded.set(EMBED_HEADER, '1');

  const response = NextResponse.next({
    request: { headers: forwarded },
  });

  // Also stamp the marker on the response so a reverse proxy / debug
  // log can see at a glance that an embed route was served.
  if (isEmbed) response.headers.set(EMBED_HEADER, '1');

  const host = request.headers.get('host') ?? '';
  const parts = host.split('.');
  
  // Check if the first part is a country code
  // e.g., ke.civicintelligence.com → KE
  if (parts.length >= 3) {
    const subdomain = parts[0].toLowerCase();
    const countryCode = COUNTRY_MAP[subdomain];
    
    if (countryCode) {
      // Add country header so API calls can use it
      response.headers.set('x-civic-country', countryCode);
      
      // Also set a cookie for client-side access
      response.cookies.set('civic-country', countryCode, {
        path: '/',
        maxAge: 60 * 60 * 24 * 365, // 1 year
      });
    }
  }
  
  return response;
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico).*)'],
};
