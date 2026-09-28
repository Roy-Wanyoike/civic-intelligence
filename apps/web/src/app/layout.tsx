import type { Metadata, Viewport } from 'next';
import { cookies } from 'next/headers';
import './globals.css';
import { Header } from '@/components/header';
import { Footer } from '@/components/footer';
import { Providers } from '@/components/providers';
import { ThemeProvider } from '@/components/theme-provider';
import { CommandPalette } from '@/components/command-palette';
import { ServiceWorkerRegister } from '@/components/service-worker-register';
import { PlatformJsonLd } from '@/components/json-ld';
import { colors } from '@/lib/design-tokens';
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import { GovernmentProvider } from '@/lib/government-context';
import { isEmbedRequest } from '@/lib/embed';
import {
  DEFAULT_SELECTION,
  GOVERNMENT_COOKIE,
  type GovernmentSelection,
} from '@/lib/government-defaults';

// siteConfig is the single source of truth for the canonical site URL, social
// handles, and other SEO-relevant constants. The URL is read from the
// NEXT_PUBLIC_SITE_URL env var (set on Vercel); falls back to the production
// domain so Open Graph + canonical URLs work out of the box.
const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? 'https://civicintelligence.vercel.app';
const SITE_NAME = 'Civic Intelligence';
const SITE_TAGLINE = 'Understand what your government is doing';
const SITE_DESCRIPTION =
  'Evidence-grounded explanations of Kenyan legislation, regulations, and civic activity across 14 African countries. Track Bills, read the Hansard, follow MPs, and ask grounded questions — no legal jargon required.';

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: `${SITE_NAME} — ${SITE_TAGLINE}`,
    template: `%s · ${SITE_NAME}`,
  },
  description: SITE_DESCRIPTION,
  applicationName: SITE_NAME,
  authors: [{ name: 'Civic Intelligence Platform', url: 'https://github.com/Roy-Wanyoike/civic-intelligence' }],
  creator: 'Roy Wanyoike',
  publisher: SITE_NAME,
  keywords: [
    'Kenya Parliament',
    'Kenya Bills',
    'Kenya Hansard',
    'Kenya Law',
    'Kenya Gazette',
    'civic intelligence',
    'parliamentary monitoring',
    'legislative tracking',
    'evidence-grounded AI',
    'Africa parliament',
    'Rwanda parliament',
    'Nigeria National Assembly',
    'South Africa Parliament',
    'Ghana Parliament',
    'Tanzania Bunge',
    'Uganda Parliament',
    'Senegal Assemblée Nationale',
    'Egyptian Parliament',
    'Morocco Parliament',
    'Ethiopia Parliament',
    'DR Congo Parliament',
    'Malawi National Assembly',
    'Zambia National Assembly',
    'civic tech Kenya',
    'open government Africa',
  ],
  alternates: {
    canonical: '/',
    languages: {
      'en-KE': '/',
      'sw-KE': '/',
    },
    // RSS 2.0 auto-discovery (issue #280). Each entry emits a
    // <link rel="alternate" type="application/rss+xml" title="…" href="…">
    // in the rendered <head> so RSS readers (Feedly, Inoreader, NetNewsWire)
    // auto-discover the platform's three primary feeds on any page. The
    // per-MP feed (/api/v1/feed/people/{id}.rss) is intentionally NOT
    // listed here — it's per-person, so it's discovered via the RSS icon
    // on each /people/{id} page rather than globally.
    types: {
      'application/rss+xml': [
        {
          url: '/api/v1/feed/bills.rss',
          title: 'Civic Intelligence — Bills',
        },
        {
          url: '/api/v1/feed/what-changed.rss',
          title: 'Civic Intelligence — What Changed',
        },
        {
          url: '/api/v1/feed/brief.rss',
          title: 'Civic Intelligence — Daily Brief',
        },
      ],
    },
  },
  openGraph: {
    title: `${SITE_NAME} — ${SITE_TAGLINE}`,
    description: SITE_DESCRIPTION,
    type: 'website',
    locale: 'en_KE',
    alternateLocale: ['sw_KE'],
    url: SITE_URL,
    siteName: SITE_NAME,
    // Note: images is NOT set here — Next.js auto-detects the OG image
    // from /opengraph-image.tsx in the same directory.
  },
  twitter: {
    card: 'summary_large_image',
    title: `${SITE_NAME} — ${SITE_TAGLINE}`,
    description: SITE_DESCRIPTION,
    // Note: images is NOT set here — Next.js auto-detects the Twitter image
    // from /twitter-image.tsx in the same directory.
    creator: '@roywanyoike',
    site: '@civintellig',
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-image-preview': 'large',
      'max-snippet': -1,
      'max-video-preview': -1,
    },
  },
  manifest: '/manifest.json',
  // Note: icons is NOT set here — Next.js auto-detects the favicon + apple
  // icon from /icon.tsx and /apple-icon.tsx in the same directory.
  category: 'government',
  classification: 'Civic Tech · Parliamentary Monitoring · Open Government',
  other: {
    // JSON-LD structured data — emitted as a meta tag so search engines can
    // pick it up. The WebApplication schema enables rich results for the
    // platform's identity card.
    'application-name': SITE_NAME,
    'apple-mobile-web-app-title': SITE_NAME,
    'apple-mobile-web-app-capable': 'yes',
    'apple-mobile-web-app-status-bar-style': 'black-translucent',
    'format-detection': 'telephone=no',
    'mobile-web-app-capable': 'yes',
    'theme-color': colors.forest,
    'color-scheme': 'light',
  },
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: colors.forest,
  colorScheme: 'light',
};

/**
 * Read the persisted government selection cookie server-side so the very
 * first server render already reflects the user's previous choice (no
 * client-side hydration flicker for the GovernmentSelector breadcrumb).
 */
async function readGovernmentSelection(): Promise<Partial<GovernmentSelection>> {
  try {
    const store = await cookies();
    const raw = store.get(GOVERNMENT_COOKIE)?.value;
    if (!raw) return {};
    return JSON.parse(decodeURIComponent(raw)) as Partial<GovernmentSelection>;
  } catch {
    return {};
  }
}

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // Issue #291 — embeddable widgets. Routes under /embed/* render as
  // chrome-less, standalone widgets that third-party sites load via
  // <iframe>. They don't need the navbar/footer/CommandPalette/service
  // worker, nor the React Query / theme / government providers (the
  // widgets use inline styles + their own fetch helpers, so the
  // provider tree is dead weight inside an iframe). The middleware
  // sets `x-civic-embed: 1` on the request for /embed/* paths so the
  // layout can branch without re-parsing the URL here (App Router
  // layouts don't receive the pathname).
  if (isEmbedRequest()) {
    return (
      <html lang="en">
        <body style={{ margin: 0, padding: 0, background: '#ffffff' }}>{children}</body>
      </html>
    );
  }

  // Spec §66 — resolve locale + messages on the server (reads the
  // `civic-locale` cookie). Falls back to 'en' if the cookie is absent
  // or unknown — see src/i18n/request.ts.
  const locale = await getLocale();
  const messages = await getMessages();
  const initialSelection = await readGovernmentSelection();
  const safe: Partial<GovernmentSelection> = {
    countryCode: initialSelection.countryCode ?? DEFAULT_SELECTION.countryCode,
    countryName: initialSelection.countryName ?? DEFAULT_SELECTION.countryName,
    administrationId: initialSelection.administrationId ?? '',
    administrationName:
      initialSelection.administrationName ?? DEFAULT_SELECTION.administrationName,
    presidentName: initialSelection.presidentName ?? DEFAULT_SELECTION.presidentName,
    termNumber:
      initialSelection.termNumber === undefined
        ? DEFAULT_SELECTION.termNumber
        : initialSelection.termNumber,
  };

  return (
    <html lang={locale}>
      <head>
        <PlatformJsonLd />
      </head>
      <body>
        <a href="#main" className="skip-link">Skip to content</a>
        <NextIntlClientProvider locale={locale} messages={messages}>
          <Providers>
            <ThemeProvider>
              <ServiceWorkerRegister />
              <GovernmentProvider initialSelection={safe}>
                <Header />
                <main id="main" className="min-h-[60vh]">
                  {children}
                </main>
                <Footer />
              </GovernmentProvider>
              {/* Spec §37 — global Command Palette, mounted once */}
              <CommandPalette />
            </ThemeProvider>
          </Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
