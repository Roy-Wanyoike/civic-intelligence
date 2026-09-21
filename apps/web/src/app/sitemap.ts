import type { MetadataRoute } from 'next';

// sitemap.ts — emitted at /sitemap.xml by Next.js. Lists every indexable
// route on the platform so search engines can discover pages that aren't
// linked from elsewhere (e.g. country-specific landing pages for the 13
// non-default countries).
//
// The sitemap is generated at build time. For dynamic routes ([id]), the
// platform currently emits the index page only (not every individual
// record) — the individual record pages are linked from the index pages,
// so search engines will discover them via crawl. A future enhancement
// could fetch every record ID from the API and emit a per-record entry,
// but that requires a server-side fetch at build time.
const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? 'https://civicintelligence.vercel.app';

// Country landing pages — the platform supports 14 countries, but only the
// big-6 have dedicated /country/<name> pages today. The other 8 are
// reachable via the Government Selector dropdown.
const COUNTRY_PAGES = [
  { path: '/country/kenya', priority: 0.9, changeFrequency: 'daily' as const },
  { path: '/country/uganda', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/country/tanzania', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/country/ghana', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/country/nigeria', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/country/south-africa', priority: 0.6, changeFrequency: 'weekly' as const },
];

// Primary content pages — the highest-priority pages a citizen would land on
// from a search result.
const PRIMARY_PAGES = [
  { path: '/', priority: 1.0, changeFrequency: 'daily' as const },
  { path: '/bills', priority: 0.9, changeFrequency: 'daily' as const },
  { path: '/acts', priority: 0.9, changeFrequency: 'weekly' as const },
  { path: '/constitution', priority: 0.9, changeFrequency: 'monthly' as const },
  { path: '/gazette', priority: 0.8, changeFrequency: 'daily' as const },
  { path: '/people', priority: 0.8, changeFrequency: 'weekly' as const },
  { path: '/committees', priority: 0.7, changeFrequency: 'weekly' as const },
  { path: '/governments', priority: 0.7, changeFrequency: 'monthly' as const },
  { path: '/calendar', priority: 0.7, changeFrequency: 'daily' as const },
];

// Secondary content pages — list/explorer pages for the platform's other
// document types. Lower priority because they're less likely to be the
// direct target of a search.
const SECONDARY_PAGES = [
  { path: '/search', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/graph', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/indicators', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/compare', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/debt', priority: 0.6, changeFrequency: 'weekly' as const },
  { path: '/loans', priority: 0.5, changeFrequency: 'weekly' as const },
  { path: '/grants', priority: 0.5, changeFrequency: 'weekly' as const },
  { path: '/policies', priority: 0.5, changeFrequency: 'weekly' as const },
  { path: '/feed', priority: 0.5, changeFrequency: 'daily' as const },
  { path: '/trending', priority: 0.5, changeFrequency: 'daily' as const },
  { path: '/what-changed', priority: 0.5, changeFrequency: 'daily' as const },
  { path: '/briefing', priority: 0.6, changeFrequency: 'daily' as const },
  { path: '/scenarios', priority: 0.4, changeFrequency: 'weekly' as const },
  { path: '/ask', priority: 0.5, changeFrequency: 'weekly' as const },
  { path: '/trust', priority: 0.4, changeFrequency: 'weekly' as const },
];

// Reference / utility pages — discoverable but not primary content.
const UTILITY_PAGES = [
  { path: '/about', priority: 0.3, changeFrequency: 'monthly' as const },
  { path: '/datasets', priority: 0.4, changeFrequency: 'monthly' as const },
  { path: '/developers', priority: 0.5, changeFrequency: 'weekly' as const },
  { path: '/participation', priority: 0.4, changeFrequency: 'monthly' as const },
  { path: '/institutions', priority: 0.4, changeFrequency: 'monthly' as const },
  { path: '/legislatures', priority: 0.4, changeFrequency: 'monthly' as const },
  { path: '/constituencies', priority: 0.4, changeFrequency: 'monthly' as const },
  { path: '/topics', priority: 0.3, changeFrequency: 'weekly' as const },
  { path: '/countries', priority: 0.3, changeFrequency: 'monthly' as const },
  { path: '/dashboard', priority: 0.3, changeFrequency: 'daily' as const },
  { path: '/following', priority: 0.2, changeFrequency: 'daily' as const },
  { path: '/notifications', priority: 0.2, changeFrequency: 'daily' as const },
  { path: '/gazette/alerts', priority: 0.4, changeFrequency: 'weekly' as const },
];

// Helper for the per-record routes — emit the index page only.
function recordPages() {
  return [
    { path: '/bills/[id]', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/acts/[id]', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/people/[id]/scorecard', priority: 0.6, changeFrequency: 'monthly' as const },
    { path: '/constituencies/[id]', priority: 0.5, changeFrequency: 'monthly' as const },
    { path: '/governments/[id]', priority: 0.6, changeFrequency: 'monthly' as const },
    { path: '/scenarios/[id]', priority: 0.5, changeFrequency: 'weekly' as const },
    { path: '/legislatures/[id]', priority: 0.5, changeFrequency: 'monthly' as const },
  ];
}

export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date().toISOString();
  const all = [
    ...PRIMARY_PAGES,
    ...SECONDARY_PAGES,
    ...UTILITY_PAGES,
    ...COUNTRY_PAGES,
    ...recordPages(),
  ];
  return all.map((page) => ({
    url: `${SITE_URL}${page.path}`,
    lastModified: now,
    changeFrequency: page.changeFrequency,
    priority: page.priority,
  }));
}
