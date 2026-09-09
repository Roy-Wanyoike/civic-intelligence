import type { Metadata, Viewport } from 'next';
import './globals.css';
import { Header } from '@/components/header';
import { Footer } from '@/components/footer';
import { Providers } from '@/components/providers';

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
  themeColor: '#0c3b2e',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <a href="#main" className="skip-link">Skip to content</a>
        <Providers>
          <Header />
          <main id="main" className="min-h-[60vh]">
            {children}
          </main>
          <Footer />
        </Providers>
      </body>
    </html>
  );
}
