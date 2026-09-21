import type { ReactElement } from 'react';

// JsonLd renders a JSON-LD <script> tag in the page's <head>. JSON-LD is the
// recommended format for structured data — search engines (Google, Bing,
// Yandex) and AI crawlers (GPTBot, ClaudeBot) read it to understand the
// page's content type, identity, and key facts.
//
// Usage (in any page.tsx server component):
//
//   <JsonLd schema={{
//     '@context': 'https://schema.org',
//     '@type': 'Article',
//     headline: '...',
//     datePublished: '...',
//   }} />
//
// The component is a server component (no 'use client') so the JSON-LD is
// emitted into the static HTML — it's available to crawlers without any
// client-side JavaScript execution.
//
// The schema for the platform-wide WebApplication + Organization is emitted
// once in layout.tsx via <PlatformJsonLd />. Page-specific schemas (Bill,
// Act, Person, Event) are emitted on the relevant pages.
type JsonLdProps = {
  schema: Record<string, unknown>;
};

export function JsonLd({ schema }: JsonLdProps): ReactElement {
  return (
    <script
      type="application/ld+json"
      // The JSON is stringified server-side and emitted as a child of the
      // <script> tag. Next.js's React server renderer escapes the content
      // safely — there's no XSS risk from the schema data because it's
      // server-controlled (not user input).
      // eslint-disable-next-line react/no-danger
      dangerouslySetInnerHTML={{ __html: JSON.stringify(schema) }}
    />
  );
}

// platformJsonLd is the platform-wide structured data — emitted once in the
// root layout. Combines:
//   - WebApplication: identifies the site as a web app for rich results
//   - Organization: identifies the publisher
//   - WebSite: identifies the site itself + enables sitelinks search box
//
// The schema uses @graph so multiple entities can share the same @context.
const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? 'https://civicintelligence.vercel.app';
const SITE_NAME = 'Civic Intelligence';
const SITE_TAGLINE = 'Understand what your government is doing';

export function PlatformJsonLd(): ReactElement {
  const schema = {
    '@context': 'https://schema.org',
    '@graph': [
      {
        '@type': 'WebSite',
        '@id': `${SITE_URL}/#website`,
        url: SITE_URL,
        name: SITE_NAME,
        description: SITE_TAGLINE,
        publisher: { '@id': `${SITE_URL}/#organization` },
        potentialAction: {
          '@type': 'SearchAction',
          target: {
            '@type': 'EntryPoint',
            urlTemplate: `${SITE_URL}/search?q={search_term_string}`,
          },
          'query-input': 'required name=search_term_string',
        },
      },
      {
        '@type': 'Organization',
        '@id': `${SITE_URL}/#organization`,
        name: SITE_NAME,
        url: SITE_URL,
        logo: {
          '@type': 'ImageObject',
          url: `${SITE_URL}/og-image.png`,
          width: 1200,
          height: 630,
        },
        founder: {
          '@type': 'Person',
          name: 'Roy Wanyoike',
          url: 'https://github.com/Roy-Wanyoike',
        },
        sameAs: [
          'https://github.com/Roy-Wanyoike/civic-intelligence',
        ],
      },
      {
        '@type': 'WebApplication',
        '@id': `${SITE_URL}/#webapp`,
        url: SITE_URL,
        name: SITE_NAME,
        description: SITE_TAGLINE,
        applicationCategory: 'GovernmentApplication',
        operatingSystem: 'Web',
        browserRequirements: 'Requires JavaScript',
        offers: {
          '@type': 'Offer',
          price: '0',
          priceCurrency: 'KES',
        },
        publisher: { '@id': `${SITE_URL}/#organization` },
        featureList: [
          'Track Bills through all legislative stages',
          'Read the Hansard (parliamentary debate transcripts)',
          'View Order Papers (daily House agendas)',
          'View Votes & Proceedings (official vote records)',
          'Browse committee reports',
          'Subscribe to Kenya Gazette alerts by keyword',
          'Compare civic indicators across 14 African countries',
          'Ask grounded questions about civic matters',
          'Visualize the civic graph (Bills ↔ people ↔ committees ↔ topics)',
          'Run policy scenarios with the simulation engine',
        ],
        audience: {
          '@type': 'Audience',
          audienceType: 'Citizens of Kenya, Uganda, Tanzania, Ghana, Nigeria, South Africa, Rwanda, Zambia, Senegal, Egypt, Morocco, DR Congo, Ethiopia, Malawi',
        },
      },
    ],
  };
  return <JsonLd schema={schema} />;
}
