import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  // API rewrites — in production, set these env vars to your backend URLs:
  //   AI_SERVICE_URL=https://your-ai-service.railway.app
  //   API_SERVICE_URL=https://your-go-api.railway.app
  // In dev (localhost), they default to local ports.
  async rewrites() {
    const aiUrl = process.env.AI_SERVICE_URL ?? 'http://localhost:8000';
    const apiUrl = process.env.API_SERVICE_URL ?? 'http://localhost:9000';
    return [
      { source: '/api/v1/ai/:path*', destination: `${aiUrl}/v1/:path*` },
      { source: '/api/v1/:path*', destination: `${apiUrl}/api/v1/:path*` },
    ];
  },
  // Issue #291 — embeddable widgets. Routes under /embed/* are loaded
  // in third-party sites via <iframe>, so they need to opt out of the
  // platform's default clickjacking protection. We:
  //   1. Set `Content-Security-Policy: frame-ancestors *` — the modern
  //      standard, respected by all evergreen browsers. This permits
  //      any origin to embed the route.
  //   2. Remove `X-Frame-Options` (Next.js does not set it by default,
  //      but Vercel's edge layer sometimes injects `SAMEORIGIN`). Setting
  //      it to an empty string overrides any inherited value.
  // All other routes keep their default headers (no `frame-ancestors`
  // directive = `X-Frame-Options: SAMEORIGIN` semantics) so the main
  // site remains protected against clickjacking.
  async headers() {
    return [
      {
        source: '/embed/:path*',
        headers: [
          {
            key: 'Content-Security-Policy',
            value: 'frame-ancestors *',
          },
          {
            key: 'X-Frame-Options',
            value: '',
          },
        ],
      },
    ];
  },
};

export default withNextIntl(nextConfig);
