import Link from 'next/link';
import type { Metadata } from 'next';
import {
  ArrowRight,
  Search,
  BookOpen,
  HelpCircle,
  Radio,
  Globe,
  TrendingUp,
  Bookmark,
  Building2,
  FlaskConical,
  Newspaper,
  Bell,
  FileText,
  Sparkles,
} from 'lucide-react';
import { mockBills } from '@/lib/mock-data';
import {
  CivicHighlightsCarousel,
  type CivicHighlight,
} from '@/components/trending-carousel';
import { ConstitutionSpotlight } from '@/components/constitution-spotlight';
import { RealityBadge } from '@/components/reality-labels';
import { LiveParliamentBanner } from '@/components/live-parliament-banner';
import { fetchCalendarLiveFromBFF } from '@/lib/api';
import type { Bill } from '@/lib/types';

/**
 * Homepage (spec §32, §34 — Homepage + Homepage Component System).
 *
 * Section order (spec §32 mandates Ask + Civic Highlights + What Changed
 * in the FIRST viewport; spec §34 enumerates the 10 required components).
 *
 *   1. First viewport — Ask search bar + Civic Highlights Carousel
 *      (side-by-side on desktop; Civic Highlights is NOT buried)
 *   2. What Changed (since last visit)
 *   3. Constitution Spotlight
 *   4. Followed civic matters (topics + institutions + recent matters)
 *   5. Trending Bills
 *   6. Explore countries
 *   7. Research updates + Civic Brief
 *   8. What If? (scenarios)
 *
 * The page is a server component; data is fetched at request time with
 * a 5-minute revalidate. Mock data is used only as a fallback so the
 * homepage never renders empty during BFF outages.
 */

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

// Homepage metadata — overrides the root layout's defaults with homepage-
// specific values. The Open Graph image is auto-detected from
// /opengraph-image.tsx; canonical URL is set so search engines know this
// is the homepage (not a duplicate of the root).
export const metadata: Metadata = {
  title: 'Civic Intelligence — Understand what your government is doing',
  description:
    'Track Bills through every legislative stage. Read the Hansard. Follow MPs. Subscribe to Kenya Gazette alerts. Ask grounded questions — all with primary-source evidence, across 14 African countries.',
  alternates: {
    canonical: '/',
  },
  openGraph: {
    title: 'Civic Intelligence — Understand what your government is doing',
    description:
      'Evidence-grounded explanations of legislation, Hansard, and civic activity across 14 African countries.',
    type: 'website',
    url: '/',
  },
};

async function fetchBills(): Promise<{ items: Bill[]; source: 'api' | 'mock' }> {
  try {
    const resp = await fetch(`${API_BASE}/api/v1/bills?pageSize=20`, {
      next: { revalidate: 300 },
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    return { items: data.items || [], source: 'api' as const };
  } catch {
    return { items: mockBills, source: 'mock' as const };
  }
}

interface WhatChangedItem {
  kind: string;
  title: string;
  description: string;
  house: string;
  date: string;
  source_url: string;
  significance: string;
  verification: string;
  evidence_url: string;
  country: string;
}

async function fetchWhatChanged(): Promise<WhatChangedItem[] | null> {
  try {
    const resp = await fetch(`${API_BASE}/api/v1/what-changed`, {
      next: { revalidate: 300 },
    });
    if (!resp.ok) return null;
    const data = await resp.json();
    return data.items ?? null;
  } catch {
    return null;
  }
}

/**
 * Map a Bill to a CivicHighlight so the Civic Highlights Carousel can
 * render it with the spec-required fields (source, date, explanation,
 * evidence, verification, significance, diversity, relevance).
 */
function billToHighlight(b: Bill): CivicHighlight {
  const date =
    b.updated_at?.slice(0, 10) ?? b.publication_date ?? String(b.year);
  const topics = b.topics ?? [];
  return {
    id: b.id,
    title: b.title,
    subtitle: `${b.house_name ?? b.house ?? 'Parliament'} · ${b.year}`,
    kind: 'bill',
    href: `/bills/${b.id}`,
    source: 'Kenya Parliament',
    date,
    explanation:
      b.current_stage_simple_explanation ??
      b.purpose ??
      b.description ??
      'Active Bill before Parliament.',
    evidence: b.source_url ? [b.source_url] : [],
    verification: 'VERIFIED',
    significance: b.status === 'enacted' ? 'SUBSTANTIVE' : 'PROCEDURAL',
    relevance: topics.length ? `topic: ${topics[0]}` : 'national',
    diversity: topics.length > 1 ? 'multi-topic' : 'single-topic',
    source_url: b.source_url,
    current_stage: b.current_stage ?? undefined,
  };
}

export default async function HomePage() {
  // The plenary livestream fetch (issue #283) runs in parallel with the
  // bills + what-changed fetches so the homepage's TTFB is unchanged.
  // `cache: 'no-store'` is set inside fetchCalendarLiveFromBFF so the
  // banner always reflects the current live status — a stale cached
  // `is_live: false` would hide the banner during a sitting.
  const [{ items: bills, source: billsSource }, whatChanged, liveStatus] =
    await Promise.all([fetchBills(), fetchWhatChanged(), fetchCalendarLiveFromBFF(API_BASE)]);

  // Top 5 bills become Civic Highlights; non-enacted bills become Trending.
  const highlights = bills.slice(0, 5).map(billToHighlight);
  const trendingBills = bills
    .filter((b) => b.status !== 'enacted')
    .slice(0, 4);
  // "Followed" preview — until identity (#20) lands, surface three
  // recent Bills so the section has content. Logged-in users will see
  // their actual follows via /following.
  const followedBills = bills.slice(0, 3);
  const followedTopics =
    bills[0]?.topics ?? ['housing', 'public-finance', 'data-protection'];

  return (
    <div>
      {/* === Plenary livestream banner (issue #283) ===
          Rendered at the very top so a citizen landing on the homepage
          during a sitting sees the "🔴 Parliament is live — Watch now"
          banner before any other content. The LiveParliamentBanner
          client component renders nothing when live.is_live is false,
          so the banner slot collapses cleanly during recesses. */}
      {liveStatus && (
        <div className="border-b border-red-200 bg-civic-paper">
          <div className="mx-auto max-w-7xl px-4 py-3 sm:px-6">
            <LiveParliamentBanner live={liveStatus} />
          </div>
        </div>
      )}

      {/* === SECTION 1: First viewport — Hero with parliament image + Ask bar === */}
      <section className="relative min-h-[560px] overflow-hidden">
        {/* Background image — Kenya Parliament Buildings */}
        <div
          className="absolute inset-0 bg-cover bg-center"
          style={{ backgroundImage: 'url(/hero-parliament.jpg)' }}
          aria-hidden="true"
        />
        {/* Dark gradient overlay for text readability */}
        <div
          className="absolute inset-0 bg-gradient-to-br from-civic-forest/95 via-civic-forest/80 to-civic-ink/90"
          aria-hidden="true"
        />

        <div className="relative mx-auto max-w-7xl px-4 py-16 sm:px-6 sm:py-20 lg:py-28">
          {/* Country badge */}
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-white/10 px-3 py-1 text-xs font-medium uppercase tracking-wider text-civic-acacia backdrop-blur-sm">
              <span className="text-base" aria-hidden="true">🇰🇪</span>
              Now serving 14 African countries
            </span>
          </div>

          {/* Headline */}
          <h1 className="mt-6 max-w-3xl font-serif text-4xl font-bold leading-tight text-white sm:text-5xl md:text-6xl">
            Understand what your{' '}
            <span className="text-civic-acacia">government</span>{' '}
            is doing.
          </h1>

          {/* Subheadline */}
          <p className="mt-5 max-w-2xl text-lg text-white/85 sm:text-xl">
            Evidence-grounded explanations of legislation, regulations, and civic activity across Africa.
            No legal jargon. No political bias. Just the facts — with sources you can verify.
          </p>

          {/* Ask bar + quick actions */}
          <div className="mt-8 max-w-2xl">
            <form action="/ask" className="flex flex-col gap-3 sm:flex-row">
              <label htmlFor="q" className="sr-only">
                Search a Bill, law, topic, or ask a question
              </label>
              <input
                id="q"
                type="search"
                name="q"
                placeholder="Ask Kenya anything…"
                className="w-full flex-1 rounded-lg border border-white/20 bg-white/10 px-5 py-4 text-base text-white placeholder:text-white/60 backdrop-blur-sm focus:border-civic-acacia focus:bg-white/15 focus:outline-none"
                autoComplete="off"
              />
              <button
                type="submit"
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-civic-acacia px-6 py-4 font-semibold text-civic-ink transition hover:bg-civic-acacia/90"
              >
                <Search className="h-5 w-5" aria-hidden="true" />
                Ask
              </button>
            </form>
            {/* Quick suggestions */}
            <div className="mt-4 flex flex-wrap gap-2 text-xs">
              <span className="text-white/70">Try:</span>
              {[
                'What is happening with housing?',
                'What did the President sign today?',
                'What changed in data protection?',
              ].map((q) => (
                <Link
                  key={q}
                  href={`/ask?q=${encodeURIComponent(q)}`}
                  className="rounded-full border border-white/20 px-3 py-1.5 text-white/90 backdrop-blur-sm transition hover:border-civic-acacia hover:text-white"
                >
                  {q}
                </Link>
              ))}
            </div>

            {/* Stats row */}
            <div className="mt-8 flex flex-wrap gap-x-8 gap-y-3 text-sm">
              <div className="flex items-center gap-2">
                <FileText className="h-4 w-4 text-civic-acacia" aria-hidden="true" />
                <span className="text-white/80"><strong className="text-white">50+</strong> Bills tracked</span>
              </div>
              <div className="flex items-center gap-2">
                <Globe className="h-4 w-4 text-civic-acacia" aria-hidden="true" />
                <span className="text-white/80"><strong className="text-white">14</strong> Countries</span>
              </div>
              <div className="flex items-center gap-2">
                <BookOpen className="h-4 w-4 text-civic-acacia" aria-hidden="true" />
                <span className="text-white/80"><strong className="text-white">12</strong> Official sources</span>
              </div>
              <div className="flex items-center gap-2">
                <Radio className="h-4 w-4 text-civic-acacia" aria-hidden="true" />
                <span className="text-white/80"><strong className="text-white">Daily</strong> Gazette alerts</span>
              </div>
            </div>

            {billsSource === 'mock' && (
              <p className="mt-4 rounded-md bg-civic-acacia/10 px-3 py-1.5 text-xs text-civic-acacia">
                Showing sample data — connect the Go API (port 9000) for live Bills from kenyalaw.org
              </p>
            )}
          </div>
        </div>

        {/* Bottom fade into next section */}
        <div
          className="absolute bottom-0 left-0 right-0 h-16 bg-gradient-to-t from-civic-paper to-transparent"
          aria-hidden="true"
        />
      </section>

      {/* === SECTION 1b: Civic Highlights Carousel === */}
      <section className="bg-civic-paper">
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
          <div className="mb-4 flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-civic-acacia" aria-hidden="true" />
            <h2 className="font-serif text-xl font-semibold text-civic-forest">
              Civic Highlights
            </h2>
            <span className="text-xs text-civic-stone">Top verified developments today</span>
          </div>
          <CivicHighlightsCarousel highlights={highlights} />
        </div>
      </section>

      {/* === SECTION 2: What Changed (since last visit) === */}
      <section className="border-b border-civic-border bg-civic-paper">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="flex items-end justify-between">
            <div>
              <div className="flex items-center gap-2">
                <Radio className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
                <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                  What Changed
                </h2>
              </div>
              <p className="mt-1 text-sm text-civic-stone">
                Verified civic developments since your last visit. Every item links to evidence.
              </p>
            </div>
            <Link
              href="/what-changed"
              className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
            >
              See all changes
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>

          {whatChanged && whatChanged.length > 0 ? (
            <ul className="mt-6 grid gap-3 md:grid-cols-2">
              {whatChanged.slice(0, 4).map((c, i) => (
                <li
                  key={`${c.kind}-${i}`}
                  className="rounded-lg border border-civic-border bg-civic-mist/40 p-4 hover:border-civic-leaf"
                >
                  <a
                    href={c.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block"
                  >
                    <div className="flex items-center justify-between text-xs">
                      <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 font-medium uppercase text-civic-leaf">
                        {c.kind.replace(/_/g, ' ')}
                      </span>
                      <span className="text-civic-stone">{c.date}</span>
                    </div>
                    <h3 className="mt-2 font-serif text-base font-semibold text-civic-ink">
                      {c.title}
                    </h3>
                    <p className="mt-1 line-clamp-2 text-xs text-civic-stone">
                      {c.description}
                    </p>
                  </a>
                </li>
              ))}
            </ul>
          ) : (
            <div className="mt-6 rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
              Change feed is temporarily unavailable.{' '}
              <Link href="/what-changed" className="text-civic-leaf hover:underline">
                Try again →
              </Link>
            </div>
          )}
        </div>
      </section>

      {/* === SECTION 3: Constitution Spotlight === */}
      <section className="border-b border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="flex items-end justify-between">
            <div>
              <div className="flex items-center gap-2">
                <BookOpen className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
                <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                  Constitution Spotlight
                </h2>
              </div>
              <p className="mt-1 text-sm text-civic-stone">
                A random article of the Constitution of Kenya, 2010.
              </p>
            </div>
            <Link
              href="/constitution"
              className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
            >
              Read the Constitution
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>
          <div className="mt-6">
            <ConstitutionSpotlight />
          </div>
        </div>
      </section>

      {/* === SECTION 4: Followed civic matters (topics + institutions + recent matters) === */}
      <section className="border-b border-civic-border bg-civic-paper">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="flex items-end justify-between">
            <div>
              <div className="flex items-center gap-2">
                <Bookmark className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
                <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                  Followed civic matters
                </h2>
              </div>
              <p className="mt-1 text-sm text-civic-stone">
                Bills, topics, committees, and institutions you follow. Sign in to personalise.
              </p>
            </div>
            <Link
              href="/following"
              className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
            >
              Manage follows
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>

          <div className="mt-6 grid gap-6 md:grid-cols-3">
            {/* Followed Topics */}
            <div>
              <h3 className="text-xs font-semibold uppercase tracking-wider text-civic-stone">
                Topics
              </h3>
              <ul className="mt-2 flex flex-wrap gap-1.5">
                {followedTopics.map((t) => (
                  <li key={t}>
                    <Link
                      href={`/topics#${t}`}
                      className="rounded-full bg-civic-leaf/10 px-3 py-1 text-xs text-civic-leaf hover:bg-civic-leaf/20"
                    >
                      {t}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            {/* Followed Institutions */}
            <div>
              <h3 className="text-xs font-semibold uppercase tracking-wider text-civic-stone">
                Institutions
              </h3>
              <ul className="mt-2 space-y-1.5">
                {['National Assembly', 'Senate', 'Cabinet'].map((name) => (
                  <li key={name} className="flex items-center gap-2 text-sm text-civic-ink">
                    <Building2 className="h-3.5 w-3.5 text-civic-stone" aria-hidden="true" />
                    <Link href="/institutions" className="hover:text-civic-leaf">
                      {name}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            {/* Recent Matters (followed) */}
            <div>
              <h3 className="text-xs font-semibold uppercase tracking-wider text-civic-stone">
                Recent matters
              </h3>
              <ul className="mt-2 space-y-1.5">
                {followedBills.map((b) => (
                  <li key={b.id} className="text-sm">
                    <Link
                      href={`/bills/${b.id}`}
                      className="text-civic-ink hover:text-civic-leaf"
                    >
                      {b.identifier} — {b.title.length > 40 ? `${b.title.slice(0, 40)}…` : b.title}
                    </Link>
                  </li>
                ))}
                {followedBills.length === 0 && (
                  <li className="text-xs text-civic-stone">No followed matters yet.</li>
                )}
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* === SECTION 5: Trending Bills === */}
      <section className="border-b border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="flex items-end justify-between">
            <div>
              <div className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
                <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                  Trending Bills
                </h2>
              </div>
              <p className="mt-1 text-sm text-civic-stone">
                Active Bills currently before Parliament.
              </p>
            </div>
            <Link href="/bills" className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
              All Bills
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>
          {trendingBills.length === 0 ? (
            <div className="mt-6 rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
              No active Bills right now. Try again later.
            </div>
          ) : (
            <ul className="mt-6 grid gap-4 md:grid-cols-2">
              {trendingBills.map((b) => (
                <li
                  key={b.id}
                  className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
                >
                  <Link href={`/bills/${b.id}`} className="block">
                    <div className="flex items-center justify-between gap-2 text-xs text-civic-stone">
                      <span>{b.house_name ?? b.house ?? 'Parliament'}</span>
                      <span className="rounded-full bg-civic-acacia/20 px-2 py-0.5 font-medium text-civic-ink">
                        {b.current_stage ?? 'In progress'}
                      </span>
                    </div>
                    <h3 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                      {b.title}
                    </h3>
                    {b.purpose && (
                      <p className="mt-2 line-clamp-2 text-sm text-civic-stone">{b.purpose}</p>
                    )}
                    {b.topics && b.topics.length > 0 && (
                      <div className="mt-3 flex flex-wrap gap-1.5">
                        {b.topics.slice(0, 3).map((t) => (
                          <span
                            key={t}
                            className="rounded-full bg-civic-leaf/10 px-2 py-0.5 text-xs text-civic-leaf"
                          >
                            {t}
                          </span>
                        ))}
                      </div>
                    )}
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      {/* === SECTION 6: Explore countries === */}
      <section className="border-b border-civic-border bg-civic-paper">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="flex items-end justify-between">
            <div>
              <div className="flex items-center gap-2">
                <Globe className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
                <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                  Explore countries
                </h2>
              </div>
              <p className="mt-1 text-sm text-civic-stone">
                Civic Intelligence is built country-by-country. Kenya is first — more African countries coming.
              </p>
            </div>
            <Link href="/countries" className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
              All countries
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>
          <ul className="mt-6 grid gap-3 sm:grid-cols-3 lg:grid-cols-6">
            {[
              { code: 'KE', name: 'Kenya', flag: '🇰🇪', active: true },
              { code: 'UG', name: 'Uganda', flag: '🇺🇬', active: false },
              { code: 'TZ', name: 'Tanzania', flag: '🇹🇿', active: false },
              { code: 'GH', name: 'Ghana', flag: '🇬🇭', active: false },
              { code: 'NG', name: 'Nigeria', flag: '🇳🇬', active: false },
              { code: 'ZA', name: 'South Africa', flag: '🇿🇦', active: false },
            ].map((c) => (
              <li
                key={c.code}
                className={`rounded-lg border p-4 ${
                  c.active
                    ? 'border-civic-leaf bg-civic-paper'
                    : 'border-civic-border bg-civic-mist/30 opacity-70'
                }`}
              >
                {c.active ? (
                  <Link
                    href={`/country/${c.code.toLowerCase()}`}
                    className="block text-center"
                  >
                    <div className="text-2xl">{c.flag}</div>
                    <div className="mt-1 text-sm font-semibold text-civic-ink">{c.name}</div>
                    <div className="text-[10px] text-civic-leaf">Active</div>
                  </Link>
                ) : (
                  <div className="text-center">
                    <div className="text-2xl">{c.flag}</div>
                    <div className="mt-1 text-sm font-semibold text-civic-ink">{c.name}</div>
                    <div className="text-[10px] text-civic-stone">Planned</div>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
      </section>

      {/* === SECTION 7: Research updates + Civic Brief === */}
      <section className="border-b border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="grid gap-8 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <div className="flex items-center gap-2">
                <FlaskConical className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
                <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                  Research updates
                </h2>
              </div>
              <p className="mt-1 text-sm text-civic-stone">
                Save Bills, documents, and searches into research missions. Reproducible, citation-linked.
              </p>
              <ul className="mt-6 space-y-3">
                <li className="rounded-lg border border-civic-border bg-civic-paper p-4">
                  <div className="text-xs font-semibold uppercase tracking-wider text-civic-leaf">
                    Mission
                  </div>
                  <h3 className="mt-1 font-serif text-base font-semibold text-civic-ink">
                    Affordable Housing Bill — fiscal impact review
                  </h3>
                  <p className="mt-1 text-xs text-civic-stone">
                    12 documents · 3 timelines · 4 cross-references · 2 contradictions flagged
                  </p>
                </li>
                <li className="rounded-lg border border-civic-border bg-civic-paper p-4">
                  <div className="text-xs font-semibold uppercase tracking-wider text-civic-leaf">
                    Mission
                  </div>
                  <h3 className="mt-1 font-serif text-base font-semibold text-civic-ink">
                    Data Protection Act — amendment lineage
                  </h3>
                  <p className="mt-1 text-xs text-civic-stone">
                    8 documents · 2 timelines · 1 contradiction under review
                  </p>
                </li>
              </ul>
              <div className="mt-4">
                <Link
                  href="/research"
                  className="inline-flex items-center gap-2 rounded-lg border border-civic-border bg-civic-paper px-5 py-3 text-sm font-semibold text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf"
                >
                  Open research workspace
                  <ArrowRight className="h-4 w-4" aria-hidden="true" />
                </Link>
              </div>
            </div>

            <div className="rounded-2xl bg-civic-forest px-6 py-8 text-center text-white">
              <Newspaper className="mx-auto h-8 w-8 text-civic-acacia" aria-hidden="true" />
              <h3 className="mt-3 font-serif text-xl font-semibold">Daily Civic Brief</h3>
              <p className="mt-2 text-sm text-white/90">
                Today&apos;s verified civic developments in plain language — delivered every morning.
              </p>
              <Link
                href="/briefing"
                className="mt-5 inline-flex items-center gap-2 rounded-lg bg-civic-acacia px-5 py-3 text-sm font-semibold text-civic-ink hover:bg-civic-acacia/90"
              >
                <Bell className="h-4 w-4" aria-hidden="true" />
                Read today&apos;s brief
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* === SECTION 8: What If? (scenarios) === */}
      <section className="bg-violet-50">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="flex items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-2">
                <HelpCircle className="h-6 w-6 text-violet-700" aria-hidden="true" />
                <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                  Explore a &ldquo;What If?&rdquo;
                </h2>
                <RealityBadge kind="HYPOTHETICAL" size="md" />
              </div>
              <p className="mt-2 max-w-2xl text-sm text-stone-700">
                Explore hypothetical civic scenarios — what could happen if a proposal becomes law,
                if implementation is delayed, or if funding changes. Every scenario is clearly labelled
                <strong> HYPOTHETICAL</strong>; every result is labelled <strong>SIMULATED</strong>.
                Reality is observed; scenarios are constructed; assumptions are explicit; evidence remains traceable.
              </p>
            </div>
            <Link
              href="/scenarios"
              className="hidden flex-shrink-0 items-center gap-2 rounded-lg bg-violet-700 px-5 py-3 text-sm font-semibold text-white transition hover:bg-violet-800 sm:inline-flex"
            >
              Explore scenarios
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>
          <div className="mt-6 grid gap-4 sm:grid-cols-3">
            <div className="rounded-lg border border-violet-200 bg-white p-4">
              <h3 className="font-serif text-base font-semibold text-violet-900">
                Policy scenarios
              </h3>
              <p className="mt-1 text-xs text-stone-700">
                What could happen if a proposed policy is implemented?
              </p>
            </div>
            <div className="rounded-lg border border-violet-200 bg-white p-4">
              <h3 className="font-serif text-base font-semibold text-violet-900">
                Historical counterfactuals
              </h3>
              <p className="mt-1 text-xs text-stone-700">
                What would the timeline look like under different documented historical conditions?
              </p>
            </div>
            <div className="rounded-lg border border-violet-200 bg-white p-4">
              <h3 className="font-serif text-base font-semibold text-violet-900">
                Comparative scenarios
              </h3>
              <p className="mt-1 text-xs text-stone-700">
                Compare scenarios side-by-side without political rankings.
              </p>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
