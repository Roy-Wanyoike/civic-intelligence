import Link from 'next/link';
import type { Metadata } from 'next';
import {
  BookOpen,
  Scale,
  Landmark,
  FileText,
  Users,
  Sparkles,
  ArrowRight,
  Clock,
  Calendar,
} from 'lucide-react';
import { guides, CATEGORY_STYLES } from './guides';
import type { GuideCategory } from './guides';

/**
 * Civic Education Hub — landing page at `/learn`.
 *
 * Server component. Renders one card per published guide in the registry.
 * Cards are colour-coded by category (see `CATEGORY_STYLES`) and link to
 * the per-guide page at `/learn/<slug>`.
 *
 * The page itself is statically rendered at build time — guides are
 * authored content, not live data — and pre-rendered via the static
 * params in `learn/[slug]/page.tsx`. Together the two pages form a small,
 * fully indexable civic-ed mini-site.
 */
export const metadata: Metadata = {
  title: 'Civic Education Hub',
  description:
    'Plain-language guides to how Kenya\'s Parliament works — how a Bill becomes a law, how to read a Bill, your constitutional rights, and how to participate in public participation.',
  alternates: { canonical: '/learn' },
  openGraph: {
    title: 'Civic Education Hub · Civic Intelligence',
    description:
      'Plain-language guides to Kenya\'s Parliament, Bills, the Constitution, and public participation.',
    type: 'website',
    url: '/learn',
  },
};

// Per-category icons — a different lucide glyph per category so the
// landing grid reads as a visual map even before the reader parses the
// headings. Falls back to BookOpen for any new category not yet mapped.
const CATEGORY_ICONS: Record<GuideCategory, typeof BookOpen> = {
  Legislation: FileText,
  Parliament: Landmark,
  Constitution: Scale,
  'Civic Participation': Users,
  'Reading Skills': BookOpen,
};

export default function LearnLandingPage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      {/* Breadcrumb — back to the platform home. */}
      <nav aria-label="Breadcrumb" className="mb-6 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li>
            <Link href="/" className="hover:text-civic-leaf">
              Home
            </Link>
          </li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink" aria-current="page">
            Learn
          </li>
        </ol>
      </nav>

      {/* Hero */}
      <header className="max-w-3xl">
        <div className="inline-flex items-center gap-2 rounded-full bg-civic-leaf/10 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-civic-leaf">
          <Sparkles className="h-3.5 w-3.5" aria-hidden="true" />
          Civic Education Hub
        </div>
        <h1 className="mt-4 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          Understand how Kenya&rsquo;s Parliament works
        </h1>
        <p className="mt-3 text-base text-civic-stone">
          Plain-language guides to Bills, Parliament, the Constitution, and
          public participation. Each guide pairs a structured walkthrough with
          links to live Bills, Acts, MPs, and committees on the platform —
          so you can move from <em>understanding</em> to{' '}
          <em>observing</em> in one click.
        </p>
      </header>

      {/* Guides grid */}
      <section aria-labelledby="guides-heading" className="mt-10">
        <h2 id="guides-heading" className="sr-only">
          All civic education guides
        </h2>
        <ul className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {guides.map((guide) => {
            const Icon = CATEGORY_ICONS[guide.category] ?? BookOpen;
            const style = CATEGORY_STYLES[guide.category];
            return (
              <li key={guide.slug}>
                <Link
                  href={`/learn/${guide.slug}`}
                  className={`group flex h-full flex-col rounded-xl border border-civic-border bg-civic-paper p-6 transition hover:shadow-md ${style.ring}`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <span
                      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-semibold ${style.badge}`}
                    >
                      <span
                        className={`h-1.5 w-1.5 rounded-full ${style.dot}`}
                        aria-hidden="true"
                      />
                      {guide.category}
                    </span>
                    <Icon
                      className="h-5 w-5 flex-shrink-0 text-civic-stone group-hover:text-civic-leaf"
                      aria-hidden="true"
                    />
                  </div>
                  <h3 className="mt-4 font-serif text-lg font-semibold text-civic-forest group-hover:text-civic-leaf">
                    {guide.title}
                  </h3>
                  <p className="mt-2 flex-1 text-sm text-civic-stone">
                    {guide.description}
                  </p>
                  <div className="mt-5 flex items-center gap-4 text-xs text-civic-stone">
                    <span className="inline-flex items-center gap-1">
                      <Clock className="h-3.5 w-3.5" aria-hidden="true" />
                      {guide.readingTimeMinutes} min read
                    </span>
                    <span className="inline-flex items-center gap-1">
                      <Calendar className="h-3.5 w-3.5" aria-hidden="true" />
                      Reviewed {formatReviewDate(guide.lastReviewed)}
                    </span>
                  </div>
                  <span className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-civic-leaf group-hover:underline">
                    Read guide
                    <ArrowRight
                      className="h-4 w-4 transition-transform group-hover:translate-x-0.5"
                      aria-hidden="true"
                    />
                  </span>
                </Link>
              </li>
            );
          })}
        </ul>
      </section>

      {/* Bottom CTA — encourage the reader to act on what they've learned. */}
      <section className="mt-14 rounded-2xl border border-civic-border bg-civic-mist/60 p-8">
        <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
          <div className="max-w-xl">
            <h2 className="font-serif text-xl font-semibold text-civic-forest">
              Ready to act on what you&rsquo;ve learned?
            </h2>
            <p className="mt-2 text-sm text-civic-stone">
              Track a Bill, find your MP, or read the Constitution on the
              platform. Every page on Civic Intelligence links back to the
              authoritative source so you can verify what you read here.
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            <Link
              href="/bills"
              className="inline-flex items-center gap-2 rounded-full bg-civic-forest px-5 py-2.5 text-sm font-semibold text-civic-paper transition hover:bg-civic-leaf"
            >
              <FileText className="h-4 w-4" aria-hidden="true" />
              Browse Bills
            </Link>
            <Link
              href="/constitution"
              className="inline-flex items-center gap-2 rounded-full border border-civic-border bg-civic-paper px-5 py-2.5 text-sm font-semibold text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf"
            >
              <Scale className="h-4 w-4" aria-hidden="true" />
              Read the Constitution
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}

/** Format the last-reviewed date as e.g. "Jan 2025" — compact + human. */
function formatReviewDate(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleDateString('en-KE', {
      month: 'short',
      year: 'numeric',
    });
  } catch {
    return iso;
  }
}
