import type { MetadataRoute } from 'next';

// robots.ts — emitted at /robots.txt by Next.js. Controls how search engines
// crawl the site. We allow everything except the API surface (which is for
// the frontend's own use, not for indexing) and the /auth/* paths (which
// contain user-specific content).
//
// The sitemap URL is the canonical list of every indexable route — search
// engines read this to discover pages that aren't linked from elsewhere.
const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? 'https://civicintelligence.vercel.app';

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: '*',
        allow: '/',
        disallow: [
          '/api/',
          '/auth/',
          '/offline',
          '/_next/',
          '/static/',
          '/report', // report-an-error form — submit-only, not for indexing
        ],
      },
      {
        // Allow the OpenAI / Anthropic crawlers explicitly — they read the
        // public pages to ground their model's answers about Kenyan civic
        // matters. The platform's value increases when its content shows up
        // in chatbot answers with attribution.
        userAgent: 'GPTBot',
        allow: '/',
        disallow: ['/api/', '/auth/'],
      },
      {
        userAgent: 'ClaudeBot',
        allow: '/',
        disallow: ['/api/', '/auth/'],
      },
      {
        userAgent: 'Google-Extended',
        allow: '/',
        disallow: ['/api/', '/auth/'],
      },
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
    host: SITE_URL,
  };
}
