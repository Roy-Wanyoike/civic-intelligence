/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  // Backend is one BFF + AI service. Rewrite /api/v1 to the Go BFF and /api/v1/ai to Python.
  async rewrites() {
    return [
      { source: '/api/v1/ai/:path*', destination: `${process.env.AI_SERVICE_URL ?? 'http://localhost:8000'}/v1/:path*` },
      { source: '/api/v1/:path*', destination: `${process.env.API_SERVICE_URL ?? 'http://localhost:9000'}/api/v1/:path*` },
    ];
  },
};

export default nextConfig;
