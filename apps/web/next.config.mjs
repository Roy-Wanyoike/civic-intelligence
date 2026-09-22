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
};

export default withNextIntl(nextConfig);
