import type { Metadata, Viewport } from 'next';
import { cookies } from 'next/headers';
import './globals.css';
import { Header } from '@/components/header';
import { Footer } from '@/components/footer';
import { Providers } from '@/components/providers';
import { ThemeProvider } from '@/components/theme-provider';
import { CommandPalette } from '@/components/command-palette';
import { ServiceWorkerRegister } from '@/components/service-worker-register';
import { colors } from '@/lib/design-tokens';
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import { GovernmentProvider } from '@/lib/government-context';
import {
  DEFAULT_SELECTION,
  GOVERNMENT_COOKIE,
  type GovernmentSelection,
} from '@/lib/government-defaults';

export const metadata: Metadata = {
  title: {
    default: 'Civic Intelligence — Understand what your government is doing',
    template: '%s · Civic Intelligence',
  },
  description:
    'Evidence-grounded explanations of Kenyan legislation, regulations, and civic activity. No legal jargon required.',
  applicationName: 'Civic Intelligence',
  authors: [{ name: 'Civic Intelligence Platform' }],
  keywords: ['Kenya', 'Parliament', 'Bills', 'Legislation', 'Civic', 'Evidence-grounded AI'],
  openGraph: {
    title: 'Civic Intelligence',
    description: 'Understand what your government is doing. Evidence-grounded civic intelligence for Kenya.',
    type: 'website',
    locale: 'en_KE',
  },
  robots: { index: true, follow: true },
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: colors.forest,
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
