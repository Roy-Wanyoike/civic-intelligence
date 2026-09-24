import Link from 'next/link';
import { notFound } from 'next/navigation';
import type { Metadata } from 'next';
import {
  ArrowLeft,
  BookOpen,
  Clock,
  Calendar,
  ExternalLink,
  Link2,
  List,
} from 'lucide-react';
import {
  guides,
  getGuideBySlug,
  getAllGuideSlugs,
  CATEGORY_STYLES,
} from '../guides';
import { GuideFeedback } from '@/components/guide-feedback';

/**
 * Per-guide page at `/learn/<slug>`.
 *
 * Server component. Pre-renders every published guide at build time via
 * `generateStaticParams`. Reads the slug, finds the matching guide in the
 * registry, emits SEO metadata, and renders:
 *   - a breadcrumb back to /learn,
 *   - a header with category badge + reading time + last-reviewed,
 *   - a body grid: main content (left, prose-styled) + TOC sidebar (right),
 *   - a sources list,
 *   - a related-platform-links list, and
 *   - the "Was this helpful?" feedback widget.
 *
 * The TOC is derived from `guide.sections` — no manual TOC maintenance.
 */
export const dynamicParams = false;

export function generateStaticParams() {
  return getAllGuideSlugs().map((slug) => ({ slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const guide = getGuideBySlug(slug);
  if (!guide) {
    return { title: 'Guide not found' };
  }
  const url = `/learn/${guide.slug}`;
  return {
    title: guide.title,
    description: guide.description,
    alternates: { canonical: url },
    openGraph: {
      title: `${guide.title} · Civic Intelligence`,
      description: guide.description,
      type: 'article',
      url,
      publishedTime: guide.lastReviewed,
      modifiedTime: guide.lastReviewed,
      authors: ['Civic Intelligence Platform'],
      tags: [guide.category, 'Kenya', 'civic education'],
    },
    twitter: {
      card: 'summary_large_image',
      title: `${guide.title} · Civic Intelligence`,
      description: guide.description,
    },
    keywords: [
      'Kenya',
      'Parliament of Kenya',
      'civic education',
      guide.category,
      ...guide.title.split(/\s+/).slice(0, 6),
    ],
  };
}

export default async function GuidePage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const guide = getGuideBySlug(slug);
  if (!guide) {
    notFound();
  }
  const style = CATEGORY_STYLES[guide.category];

  return (
    <article className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      {/* Breadcrumb */}
      <nav aria-label="Breadcrumb" className="mb-6 text-sm text-civic-stone">
        <ol className="flex flex-wrap items-center gap-2">
          <li>
            <Link href="/" className="hover:text-civic-leaf">
              Home
            </Link>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <Link href="/learn" className="hover:text-civic-leaf">
              Learn
            </Link>
          </li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink" aria-current="page">
            {guide.title}
          </li>
        </ol>
      </nav>

      {/* Header */}
      <header className="max-w-3xl">
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <span
            className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 font-semibold ${style.badge}`}
          >
            <span
              className={`h-1.5 w-1.5 rounded-full ${style.dot}`}
              aria-hidden="true"
            />
            {guide.category}
          </span>
          <span className="inline-flex items-center gap-1 text-civic-stone">
            <Clock className="h-3.5 w-3.5" aria-hidden="true" />
            {guide.readingTimeMinutes} min read
          </span>
          <span className="inline-flex items-center gap-1 text-civic-stone">
            <Calendar className="h-3.5 w-3.5" aria-hidden="true" />
            Last reviewed {formatReviewDate(guide.lastReviewed)}
          </span>
        </div>
        <h1 className="mt-4 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          {guide.title}
        </h1>
        <p className="mt-3 text-base text-civic-stone">{guide.summary}</p>
      </header>

      {/* Body grid: main content + sticky TOC */}
      <div className="mt-10 grid gap-10 lg:grid-cols-[1fr_18rem]">
        {/* Main content */}
        <div className="min-w-0">
          <div className="civic-prose">
            {guide.sections.map((section) => (
              <section
                key={section.id}
                id={section.id}
                aria-labelledby={`${section.id}-title`}
                className="scroll-mt-24"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <h2 id={`${section.id}-title`} className="!mt-0 !mb-0">
                    {section.title}
                  </h2>
                  {section.badge && (
                    <span
                      className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ${style.badge}`}
                    >
                      {section.badge}
                    </span>
                  )}
                </div>
                <div className="mt-3">{section.body}</div>
              </section>
            ))}
          </div>

          {/* Sources */}
          <section
            aria-labelledby="sources-heading"
            className="mt-12 rounded-xl border border-civic-border bg-civic-paper p-6"
          >
            <h2
              id="sources-heading"
              className="inline-flex items-center gap-2 font-serif text-lg font-semibold text-civic-forest"
            >
              <BookOpen className="h-5 w-5" aria-hidden="true" />
              Sources
            </h2>
            <p className="mt-1 text-sm text-civic-stone">
              Every claim in this guide is traceable to an authoritative
              upstream source. Verify anything you read here against the
              originals.
            </p>
            <ul className="mt-4 space-y-2">
              {guide.sources.map((source) => (
                <li key={source.url}>
                  <a
                    href={source.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-start gap-2 text-sm text-civic-leaf hover:underline"
                  >
                    <ExternalLink
                      className="mt-0.5 h-4 w-4 flex-shrink-0"
                      aria-hidden="true"
                    />
                    <span>{source.label}</span>
                  </a>
                </li>
              ))}
            </ul>
          </section>

          {/* Related platform links */}
          <section
            aria-labelledby="related-heading"
            className="mt-6 rounded-xl border border-civic-border bg-civic-mist/50 p-6"
          >
            <h2
              id="related-heading"
              className="inline-flex items-center gap-2 font-serif text-lg font-semibold text-civic-forest"
            >
              <Link2 className="h-5 w-5" aria-hidden="true" />
              Related on the platform
            </h2>
            <p className="mt-1 text-sm text-civic-stone">
              Move from understanding to observing — these pages reflect the
              live state of Parliament.
            </p>
            <ul className="mt-4 grid gap-3 sm:grid-cols-2">
              {guide.relatedLinks.map((link) => (
                <li key={link.href}>
                  <Link
                    href={link.href}
                    className="group block rounded-lg border border-civic-border bg-civic-paper p-3 transition hover:border-civic-leaf/40 hover:shadow-sm"
                  >
                    <span className="block text-sm font-semibold text-civic-ink group-hover:text-civic-leaf">
                      {link.label}
                    </span>
                    {link.description && (
                      <span className="mt-1 block text-xs text-civic-stone">
                        {link.description}
                      </span>
                    )}
                  </Link>
                </li>
              ))}
            </ul>
          </section>

          {/* Feedback */}
          <section className="mt-6">
            <GuideFeedback slug={guide.slug} />
          </section>
        </div>

        {/* TOC sidebar */}
        <aside className="hidden lg:block">
          <nav
            aria-label="Table of contents"
            className="sticky top-24 rounded-xl border border-civic-border bg-civic-paper p-5"
          >
            <h2 className="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-civic-stone">
              <List className="h-3.5 w-3.5" aria-hidden="true" />
              On this page
            </h2>
            <ol className="mt-3 space-y-1 text-sm">
              {guide.sections.map((section, idx) => (
                <li key={section.id} className="flex items-start gap-2">
                  <span className="mt-0.5 text-xs text-civic-stone">
                    {String(idx + 1).padStart(2, '0')}
                  </span>
                  <a
                    href={`#${section.id}`}
                    className="block text-civic-ink transition hover:text-civic-leaf"
                  >
                    {section.title}
                  </a>
                </li>
              ))}
            </ol>
            <div className="mt-5 border-t border-civic-border pt-4">
              <Link
                href="/learn"
                className="inline-flex items-center gap-1.5 text-xs text-civic-stone hover:text-civic-leaf"
              >
                <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
                All guides
              </Link>
            </div>
          </nav>
        </aside>
      </div>

      {/* Footer navigation: prev / next guide + back to /learn */}
      <nav
        aria-label="More guides"
        className="mt-14 border-t border-civic-border pt-8"
      >
        <GuidePager currentSlug={guide.slug} />
      </nav>
    </article>
  );
}

/**
 * GuidePager renders prev / next guide links at the bottom of every guide
 * page so a reader can move sequentially through the hub. Wrapped in a
 * component so the page body stays scannable.
 */
function GuidePager({ currentSlug }: { currentSlug: string }) {
  const idx = guides.findIndex((g) => g.slug === currentSlug);
  if (idx === -1) return null;
  const prev = idx > 0 ? guides[idx - 1] : null;
  const next = idx < guides.length - 1 ? guides[idx + 1] : null;

  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      {prev ? (
        <Link
          href={`/learn/${prev.slug}`}
          className="group inline-flex max-w-sm flex-1 items-start gap-3 rounded-lg border border-civic-border bg-civic-paper p-4 transition hover:border-civic-leaf/40"
        >
          <ArrowLeft
            className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-stone group-hover:text-civic-leaf"
            aria-hidden="true"
          />
          <span className="min-w-0">
            <span className="block text-xs uppercase tracking-wide text-civic-stone">
              Previous guide
            </span>
            <span className="block truncate text-sm font-semibold text-civic-ink group-hover:text-civic-leaf">
              {prev.title}
            </span>
          </span>
        </Link>
      ) : (
        <span className="hidden sm:block sm:flex-1" aria-hidden="true" />
      )}
      <Link
        href="/learn"
        className="inline-flex items-center justify-center gap-1.5 rounded-md border border-civic-border bg-civic-paper px-4 py-2 text-sm font-medium text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf"
      >
        <BookOpen className="h-4 w-4" aria-hidden="true" />
        All guides
      </Link>
      {next ? (
        <Link
          href={`/learn/${next.slug}`}
          className="group inline-flex max-w-sm flex-1 items-start gap-3 rounded-lg border border-civic-border bg-civic-paper p-4 text-right transition hover:border-civic-leaf/40 sm:flex-row-reverse"
        >
          <ArrowLeft
            className="mt-0.5 h-4 w-4 flex-shrink-0 rotate-180 text-civic-stone group-hover:text-civic-leaf"
            aria-hidden="true"
          />
          <span className="min-w-0">
            <span className="block text-xs uppercase tracking-wide text-civic-stone">
              Next guide
            </span>
            <span className="block truncate text-sm font-semibold text-civic-ink group-hover:text-civic-leaf">
              {next.title}
            </span>
          </span>
        </Link>
      ) : (
        <span className="hidden sm:block sm:flex-1" aria-hidden="true" />
      )}
    </div>
  );
}

/** Format the last-reviewed date as e.g. "15 January 2025" — long form. */
function formatReviewDate(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleDateString('en-KE', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    });
  } catch {
    return iso;
  }
}
