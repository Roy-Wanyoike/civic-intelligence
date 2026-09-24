import type { ReactNode } from 'react';

/**
 * Civic Education Hub — guide content model.
 * -------------------------------------------
 * Each guide is a server component module that exports a `Guide` object.
 * The renderer at `apps/web/src/app/learn/[slug]/page.tsx` looks up the
 * guide by slug, emits SEO metadata, and renders the body alongside a
 * table-of-contents sidebar, a sources list, related platform links, and
 * a "Was this helpful?" feedback section.
 *
 * Why a structured content model rather than 5 hand-rolled pages:
 *   - The TOC, feedback widget, sources, and breadcrumb are identical
 *     across guides. A single renderer means every guide picks up
 *     design-system updates for free (e.g. the prose styles in
 *     globals.css).
 *   - Adding a 6th guide = dropping one file in this directory + adding
 *     one line to `guides/index.ts`. No plumbing changes.
 *   - The renderer pre-renders every guide at build time via
 *     `generateStaticParams`, so guides ship as static HTML (fast TTFB,
 *     indexable by search engines).
 */

export type GuideCategory =
  | 'Legislation'
  | 'Parliament'
  | 'Constitution'
  | 'Civic Participation'
  | 'Reading Skills';

export interface GuideSection {
  /** Anchor id — must be unique within the guide. Used as the `#fragment`
   * the TOC links to. Lowercase, hyphen-separated. */
  id: string;
  /** Section heading rendered inside the prose body. */
  title: string;
  /** Optional stage label rendered as a badge next to the heading. Used by
   * the bill-becomes-law walkthrough for "Stage 1" … "Stage 8". */
  badge?: string;
  /** Section content — JSX rendered inside the `civic-prose` wrapper. */
  body: ReactNode;
}

export interface GuideSource {
  /** Display label, e.g. "Constitution of Kenya, 2010 — Article 118". */
  label: string;
  /** Canonical URL of the authoritative source. Always external. */
  url: string;
}

export interface GuideRelatedLink {
  /** Display label. */
  label: string;
  /** Internal platform path, e.g. `/bills`, `/acts/[id]`, `/people/[id]`. */
  href: string;
  /** Optional one-line context for why this link is relevant. */
  description?: string;
}

export interface Guide {
  slug: string;
  title: string;
  /** Short summary shown on the landing card (~120 chars). */
  summary: string;
  /** SEO meta description (≤160 chars). */
  description: string;
  category: GuideCategory;
  /** Estimated reading time in minutes — shown on the guide card + header. */
  readingTimeMinutes: number;
  /** ISO date (YYYY-MM-DD) the guide was last reviewed. Shown to the user
   * so they can judge whether the content is current. */
  lastReviewed: string;
  /** Authoritative upstream sources — kenyalaw.org, parliament.go.ke, etc. */
  sources: GuideSource[];
  /** Internal platform pages related to this guide. */
  relatedLinks: GuideRelatedLink[];
  /** Ordered sections rendered in the main body. The TOC is derived from
   * this list automatically. */
  sections: GuideSection[];
}
