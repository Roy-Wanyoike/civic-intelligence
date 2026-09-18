'use client';

/**
 * Civic Highlights Carousel (spec §33).
 *
 * A reusable, accessible carousel for Civic Highlights — the most important
 * verified civic developments of the day. Used in the first viewport of the
 * homepage (spec §32 — "must not be hidden below several unrelated sections").
 *
 * Each highlight carries the spec-required metadata fields (§33):
 *   source, date, explanation, evidence, verification, significance,
 *   diversity, relevance, plus optional kind/source_url/current_stage
 *   for richer rendering.
 *
 * Accessibility (§33):
 *   - keyboard navigation (arrows, Escape handled by parent focus)
 *   - pagination dots with role=tab + aria-selected
 *   - pause on hover / focus
 *   - reduced-motion: autoplay disabled
 *   - ARIA region with aria-roledescription=carousel
 *
 * Performance (§33):
 *   - server-rendered initial slide (the parent passes pre-fetched slides)
 *   - small payload (only the visible fields render)
 *   - subsequent slides lazy (no images, just DOM-swap)
 *   - caching delegated to the parent's fetch() (next.revalidate)
 *
 * Legacy compatibility: `TrendingCarousel` (the previous name used by
 * `/trending/page.tsx`) is kept as a thin adapter that maps the older
 * `bills` prop shape onto a `CivicHighlight[]` and forwards to the
 * canonical component.
 */

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import {
  ChevronLeft,
  ChevronRight,
  Sparkles,
  ShieldCheck,
  FileSignature,
  ExternalLink,
} from 'lucide-react';

export interface CivicHighlight {
  /** Stable unique id (used as React key). */
  id: string;
  /** Headline — short, plain-language. */
  title: string;
  /** Subtitle shown above the title, e.g. `NA · 2024` or `Hansard · 2024-09-12`. */
  subtitle?: string;
  /** Optional kind label (e.g. `bill`, `stage_change`, `assent`). */
  kind?: string;
  /** Internal link to the detail page for this highlight (e.g. `/bills/123`). */
  href?: string;
  /** Human-readable source label, e.g. `Kenya Parliament`, `Kenya Law`. */
  source: string;
  /** ISO date or display date string. */
  date: string;
  /** Plain-language explanation of what happened and why it matters. */
  explanation: string;
  /** Evidence URLs (links back to authoritative source documents). */
  evidence?: string[];
  /** Verification state, e.g. `VERIFIED`, `DISCOVERED`, `UNVERIFIED`. */
  verification?: string;
  /** Significance label, e.g. `INFORMATIONAL`, `PROCEDURAL`, `SUBSTANTIVE`,
   *  `HIGH_IMPACT`, `CRITICAL`. */
  significance?: string;
  /** Diversity descriptor, e.g. `multi-topic`, `cross-institutional`. */
  diversity?: string;
  /** Relevance descriptor, e.g. `topic: housing`, `national`, `county: Nairobi`. */
  relevance?: string;
  /** Official source URL on the upstream site (kenyalaw.org, parliament.go.ke). */
  source_url?: string;
  /** Current legislative stage, for bill-type highlights. */
  current_stage?: string;
}

interface CivicHighlightsCarouselProps {
  highlights: CivicHighlight[];
  autoPlay?: boolean;
  /** Autoplay interval in milliseconds (default 6s — slower than the old
   *  5s so readers can finish the explanation). */
  interval?: number;
  ariaLabel?: string;
}

const SIGNIFICANCE_COLORS: Record<string, string> = {
  INFORMATIONAL: 'bg-civic-mist text-civic-stone',
  MINOR: 'bg-civic-mist text-civic-stone',
  PROCEDURAL: 'bg-civic-acacia/20 text-civic-ink',
  SUBSTANTIVE: 'bg-civic-acacia/30 text-civic-clay',
  HIGH_IMPACT: 'bg-civic-clay/20 text-civic-clay',
  CRITICAL: 'bg-red-100 text-red-800',
};

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

export function CivicHighlightsCarousel({
  highlights,
  autoPlay = true,
  interval = 6000,
  ariaLabel = 'Civic Highlights — top developments today',
}: CivicHighlightsCarouselProps) {
  const [current, setCurrent] = useState(0);
  const [isPaused, setIsPaused] = useState(false);

  const next = useCallback(() => {
    setCurrent((prev) => (prev + 1) % Math.max(highlights.length, 1));
  }, [highlights.length]);

  const prev = useCallback(() => {
    setCurrent((prev) => (prev - 1 + highlights.length) % Math.max(highlights.length, 1));
  }, [highlights.length]);

  // Keyboard arrows when the carousel region has focus.
  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      next();
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      prev();
    }
  }

  useEffect(() => {
    if (!autoPlay || isPaused || highlights.length <= 1) return;
    if (prefersReducedMotion()) return;
    const timer = setInterval(next, interval);
    return () => clearInterval(timer);
  }, [autoPlay, isPaused, next, interval, highlights.length]);

  // Reset to the first slide if the highlights array changes identity
  // (e.g. when the parent re-fetches).
  useEffect(() => {
    setCurrent(0);
  }, [highlights]);

  if (!highlights || highlights.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        No civic highlights available right now.
      </div>
    );
  }

  const h = highlights[current];

  return (
    <div
      className="relative overflow-hidden rounded-xl border border-civic-border bg-civic-paper"
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={() => setIsPaused(false)}
      onFocus={() => setIsPaused(true)}
      onBlur={() => setIsPaused(false)}
      onKeyDown={onKeyDown}
      tabIndex={0}
      role="region"
      aria-roledescription="carousel"
      aria-label={ariaLabel}
    >
      {/* Header strip */}
      <div className="flex items-center gap-2 border-b border-civic-border bg-civic-mist px-4 py-3">
        <Sparkles className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
        <h2 className="font-serif text-sm font-semibold text-civic-ink">
          Civic Highlights
        </h2>
        <span className="ml-auto text-xs text-civic-stone" aria-live="polite">
          {current + 1} / {highlights.length}
        </span>
      </div>

      {/* Slide */}
      <div className="p-6">
        {h.href ? (
          <Link href={h.href} className="block focus:outline-none">
            <HighlightBody h={h} />
          </Link>
        ) : (
          <HighlightBody h={h} />
        )}
      </div>

      {/* Navigation arrows */}
      {highlights.length > 1 && (
        <>
          <button
            type="button"
            onClick={prev}
            className="absolute left-2 top-1/2 -translate-y-1/2 rounded-full bg-civic-paper/80 p-2 text-civic-stone shadow-md hover:bg-civic-paper hover:text-civic-leaf focus:outline-none focus:ring-2 focus:ring-civic-leaf"
            aria-label="Previous highlight"
          >
            <ChevronLeft className="h-5 w-5" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={next}
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded-full bg-civic-paper/80 p-2 text-civic-stone shadow-md hover:bg-civic-paper hover:text-civic-leaf focus:outline-none focus:ring-2 focus:ring-civic-leaf"
            aria-label="Next highlight"
          >
            <ChevronRight className="h-5 w-5" aria-hidden="true" />
          </button>
        </>
      )}

      {/* Pagination dots */}
      {highlights.length > 1 && (
        <div
          className="flex justify-center gap-1.5 pb-3"
          role="tablist"
          aria-label="Choose slide"
        >
          {highlights.map((_, i) => (
            <button
              key={i}
              type="button"
              role="tab"
              aria-selected={i === current}
              aria-label={`Go to slide ${i + 1}`}
              onClick={() => setCurrent(i)}
              className={`h-2 rounded-full transition-all ${
                i === current ? 'w-6 bg-civic-leaf' : 'w-2 bg-civic-border'
              }`}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function HighlightBody({ h }: { h: CivicHighlight }) {
  return (
    <>
      <div className="flex flex-wrap items-center gap-2 text-xs text-civic-stone">
        <FileSignature className="h-4 w-4 text-civic-clay" aria-hidden="true" />
        {h.subtitle && <span>{h.subtitle}</span>}
        {h.current_stage && (
          <span className="rounded-full bg-civic-acacia/20 px-2 py-0.5 font-medium text-civic-ink">
            {h.current_stage}
          </span>
        )}
        {h.kind && (
          <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 font-medium text-civic-leaf">
            {h.kind.replace(/_/g, ' ')}
          </span>
        )}
      </div>

      <h3 className="mt-3 font-serif text-xl font-semibold text-civic-ink hover:text-civic-leaf">
        {h.title}
      </h3>

      {h.explanation && (
        <p className="mt-3 text-sm text-civic-ink leading-relaxed">{h.explanation}</p>
      )}

      {/* Meta strip — source · date · verification · significance · relevance · diversity */}
      <dl className="mt-4 grid gap-2 text-xs sm:grid-cols-2">
        <div className="flex items-center gap-1.5">
          <dt className="font-semibold text-civic-stone">Source:</dt>
          <dd className="text-civic-ink">{h.source}</dd>
        </div>
        <div className="flex items-center gap-1.5">
          <dt className="font-semibold text-civic-stone">Date:</dt>
          <dd className="text-civic-ink">{h.date}</dd>
        </div>
        {h.verification && (
          <div className="flex items-center gap-1.5">
            <ShieldCheck className="h-3 w-3 text-civic-leaf" aria-hidden="true" />
            <dt className="font-semibold text-civic-stone">Verification:</dt>
            <dd className="text-civic-leaf">{h.verification}</dd>
          </div>
        )}
        {h.significance && (
          <div className="flex items-center gap-1.5">
            <dt className="font-semibold text-civic-stone">Significance:</dt>
            <dd>
              <span
                className={`rounded-full px-2 py-0.5 font-medium ${
                  SIGNIFICANCE_COLORS[h.significance] ?? ''
                }`}
              >
                {h.significance}
              </span>
            </dd>
          </div>
        )}
        {h.relevance && (
          <div className="flex items-center gap-1.5">
            <dt className="font-semibold text-civic-stone">Relevance:</dt>
            <dd className="text-civic-ink">{h.relevance}</dd>
          </div>
        )}
        {h.diversity && (
          <div className="flex items-center gap-1.5">
            <dt className="font-semibold text-civic-stone">Diversity:</dt>
            <dd className="text-civic-ink">{h.diversity}</dd>
          </div>
        )}
      </dl>

      {/* Evidence */}
      {h.evidence && h.evidence.length > 0 && (
        <div className="mt-4 border-t border-civic-border pt-3">
          <p className="text-xs font-semibold text-civic-stone">Evidence</p>
          <ul className="mt-1 space-y-1">
            {h.evidence.map((e, i) => (
              <li key={i} className="flex items-center gap-1 text-xs text-civic-leaf">
                <ExternalLink className="h-3 w-3 flex-shrink-0" aria-hidden="true" />
                <a
                  href={e}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="truncate hover:underline"
                >
                  {e}
                </a>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Source URL */}
      {h.source_url && (
        <p className="mt-3 flex items-center gap-1 text-xs text-civic-leaf">
          <ExternalLink className="h-3 w-3 flex-shrink-0" aria-hidden="true" />
          <a
            href={h.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="truncate hover:underline"
          >
            {h.source_url}
          </a>
        </p>
      )}
    </>
  );
}

// ---------------------------------------------------------------------------
// Legacy compatibility — `/trending/page.tsx` still imports `TrendingCarousel`
// and passes a `bills` prop with the older CarouselBill shape. This adapter
// maps the older shape onto a CivicHighlight[] and forwards to the canonical
// component. New callers should use CivicHighlightsCarousel directly.
// ---------------------------------------------------------------------------

export interface CarouselBill {
  id: string;
  title: string;
  house: string;
  year: number;
  source_url: string;
  publication_date?: string;
  current_stage?: string;
}

interface TrendingCarouselProps {
  bills: CarouselBill[];
  autoPlay?: boolean;
  interval?: number;
}

export function TrendingCarousel({
  bills,
  autoPlay = true,
  interval = 5000,
}: TrendingCarouselProps) {
  const highlights: CivicHighlight[] = bills.map((b) => ({
    id: b.id,
    title: b.title,
    subtitle: `${b.house} · ${b.year}`,
    kind: 'bill',
    href: `/bills/${b.id}`,
    source: 'Kenya Parliament',
    date: b.publication_date ?? String(b.year),
    explanation: b.current_stage
      ? `Current stage: ${b.current_stage}.`
      : 'Active Bill before Parliament.',
    evidence: b.source_url ? [b.source_url] : [],
    verification: 'VERIFIED',
    significance: 'PROCEDURAL',
    relevance: 'national',
    diversity: 'single-topic',
    source_url: b.source_url,
    current_stage: b.current_stage,
  }));

  return (
    <CivicHighlightsCarousel
      highlights={highlights}
      autoPlay={autoPlay}
      interval={interval}
      ariaLabel="Trending Bills this month"
    />
  );
}
