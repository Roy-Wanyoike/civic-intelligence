// Civic Daily Brief — server component.
//
// Fetches today's personalised brief from /api/v1/brief/today (Go BFF) and
// renders it with the BriefReader client component. The brief is
// AI-grounded (every item carries an evidence_url), personalised to the
// caller's followed topics / institutions / Bills (when the caller is
// authenticated — see /api/v1/brief/generate), and clearly labelled: the
// AI summary carries a RealityBadge kind=ASSUMPTION + a disclaimer.
//
// Mobile-first: collapsible sections, swipe-between-sections via the section
// nav strip (BriefReader), print-friendly CSS via globals.css @media print
// + Tailwind print: variants.
//
// Query param ?id={brief-id} (set by the Archive page) fetches a specific
// past brief instead of today's.

import Link from 'next/link';
import { Calendar, ShieldCheck, AlertCircle, Rss } from 'lucide-react';
import { PrintButton } from '@/components/print-button';
import { BriefReader } from './brief-reader';
import { formatDate } from '@/lib/utils';
import type { Metadata } from 'next';
import type { CivicBrief } from '@/lib/types';

export const metadata: Metadata = {
  title: 'Civic Daily Brief',
  description:
    'A personalised, evidence-grounded daily summary of civic developments in Kenya — with AI plain-language explanations and primary-source citations.',
};

// fetchBrief calls the Go BFF. When id is supplied, fetches a specific past
// brief; otherwise fetches today's brief (generated on-demand if missing).
// The fetch is server-side (this is a server component) so the user never
// sees the BFF URL. revalidate=300 caches the brief for 5 minutes — the
// on-demand generation in the BFF itself is idempotent for the same day.
async function fetchBrief(id?: string): Promise<CivicBrief | null> {
  const base = process.env.API_SERVICE_URL ?? 'http://localhost:9000';
  const path = id ? `/api/v1/brief/${encodeURIComponent(id)}` : '/api/v1/brief/today';
  try {
    const resp = await fetch(`${base}${path}`, {
      next: { revalidate: 300 },
      headers: { Accept: 'application/json' },
    });
    if (!resp.ok) return null;
    return (await resp.json()) as CivicBrief;
  } catch {
    return null;
  }
}

// generateEmptyBrief is the deterministic fallback rendered when the BFF is
// unreachable. We surface an honest empty state — never a fabricated brief.
// The page header still renders (with the date + headline) so the reader
// sees the page chrome, and the body explains what happened.
function generateEmptyBrief(): CivicBrief {
  const today = new Date().toISOString().slice(0, 10);
  return {
    id: `brief-${today}`,
    date: today,
    headline: 'Civic Daily Brief is temporarily unavailable.',
    sections: [
      {
        title: 'What Changed Today',
        summary:
          'The Civic Intelligence API could not be reached. The platform does not fabricate activity — when the source feed is unreachable, the brief is honestly empty. Please try again in a few minutes.',
        items: [],
      },
      { title: 'Your Followed Topics', summary: 'Unavailable.', items: [] },
      { title: 'What to Watch', summary: 'Unavailable.', items: [] },
      {
        title: 'Constitutional Context',
        summary: 'Constitutional context will appear here when the brief is available.',
        items: [
          {
            type: 'constitutional_provision',
            article: 'Article 10',
            title: 'Article 10 — National Values and Principles of Governance',
            text: 'The national values and principles of governance include patriotism, national unity, sharing and devolution of power, the rule of law, democracy and participation of the people, human dignity, equity, social justice, inclusiveness, equality, human rights, non-discrimination, protection of the marginalised, good governance, integrity, transparency and accountability.',
            connection:
              'Even when the brief is unavailable, the Constitution remains the lens through which to evaluate civic activity.',
            evidence_url: 'https://www.kenyalaw.org/kl/index.php?id=398',
          },
        ],
      },
    ],
    ai_summary:
      'Auto-generated summary. The Civic Intelligence API could not be reached, so today\u2019s brief is unavailable. The platform does not fabricate activity — please try again in a few minutes.',
    ai_disclaimer: 'AI-generated summary. Verify against primary sources.',
    ai_source: 'template-fallback',
    evidence_count: 1,
    generated_at: new Date().toISOString(),
    country: 'KE',
  };
}

export default async function BriefingPage({
  searchParams,
}: {
  searchParams?: { id?: string };
}) {
  const briefId = searchParams?.id;
  const brief = (await fetchBrief(briefId)) ?? generateEmptyBrief();
  const isFallback = brief.headline.startsWith('Civic Daily Brief is temporarily unavailable');

  return (
    <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6 sm:py-10">
      {/* Hero — date + headline + meta */}
      <header className="mb-6">
        <div className="no-print flex items-center justify-between">
          <p className="text-xs font-semibold uppercase tracking-widest text-civic-acacia">
            {brief.country === 'KE' ? 'Kenya' : brief.country} Civic Brief
          </p>
          <div className="flex items-center gap-2">
            <a
              href="/api/v1/feed/brief.rss"
              title="Subscribe to the Daily Brief RSS feed"
              aria-label="Subscribe to the Daily Brief RSS feed"
              className="inline-flex items-center gap-1.5 rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-xs font-medium text-civic-stone transition hover:border-civic-leaf hover:text-civic-leaf"
            >
              <Rss className="h-4 w-4" aria-hidden="true" />
              <span className="hidden sm:inline">RSS</span>
            </a>
            <Link
              href="/briefing/archive"
              className="hidden text-xs text-civic-stone hover:text-civic-leaf hover:underline sm:inline"
            >
              Archive
            </Link>
            <PrintButton />
          </div>
        </div>
        <div className="mt-2 flex items-center gap-2 text-xs text-civic-stone">
          <Calendar className="h-3 w-3" aria-hidden="true" />
          <time dateTime={brief.date}>{formatDate(brief.date)}</time>
          <span aria-hidden="true">·</span>
          <span className="inline-flex items-center gap-1">
            <ShieldCheck className="h-3 w-3 text-civic-leaf" aria-hidden="true" />
            {brief.evidence_count} evidence citation{brief.evidence_count === 1 ? '' : 's'}
          </span>
          <span aria-hidden="true">·</span>
          <span>Generated {formatDate(brief.generated_at)}</span>
        </div>
        <h1
          id="brief-headline"
          className="mt-3 font-serif text-3xl font-semibold leading-tight text-civic-forest sm:text-4xl"
        >
          {brief.headline}
        </h1>
        {isFallback && (
          <div
            role="alert"
            className="mt-3 flex items-start gap-2 rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900"
          >
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
            <span>
              The Civic Intelligence API could not be reached. Showing a fallback brief with no
              new activity. <Link href="/briefing" className="underline">Retry</Link>.
            </span>
          </div>
        )}
      </header>

      <BriefReader brief={brief} />
    </div>
  );
}
