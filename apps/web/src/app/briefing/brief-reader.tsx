'use client';

// BriefReader — the interactive client half of the Civic Daily Brief page.
//
// The page (apps/web/src/app/briefing/page.tsx) is a server component that
// fetches the brief from /api/v1/brief/today and passes it here. This
// component owns:
//
//   - collapsible sections (tap to expand/collapse, all open by default on
//     print)
//   - swipe-between-sections on mobile (CSS scroll-snap)
//   - the AI Summary block, clearly labelled with RealityBadge kind=ASSUMPTION
//     + the platform-wide ai_disclaimer string
//   - the Evidence footer (every source URL cited in the brief, deduped)
//   - Share button (Web Share API → clipboard fallback)
//   - "Subscribe to daily email" button (placeholder → /notifications)
//
// Mobile-first: the layout collapses to a single column under sm:, the
// section-nav becomes a horizontal scroll strip, and every touch target is
// at least 44px (per globals.css).

import { useState, useMemo, useCallback } from 'react';
import Link from 'next/link';
import {
  ExternalLink,
  ChevronDown,
  Share2,
  Mail,
  FileText,
  Scale,
  Eye,
  EyeOff,
  Sparkles,
  AlertTriangle,
} from 'lucide-react';
import { RealityBadge } from '@/components/reality-labels';
import type { CivicBrief, BriefSection } from '@/lib/types';

interface BriefReaderProps {
  brief: CivicBrief;
}

// SECTION_ICONS maps section title → a stable lucide icon. Adding a new
// section means adding an entry here, otherwise the section renders with
// the default FileText icon.
const SECTION_ICONS: Record<string, typeof FileText> = {
  'What Changed Today': FileText,
  'Your Followed Topics': Eye,
  'What to Watch': AlertTriangle,
  'Constitutional Context': Scale,
};

export function BriefReader({ brief }: BriefReaderProps) {
  // Collapsible-section state. Track by section index so the same component
  // can be reused for archive briefs with different section sets.
  const [openSections, setOpenSections] = useState<Record<number, boolean>>(
    () => Object.fromEntries(brief.sections.map((_, i) => [i, true])),
  );
  const [shareStatus, setShareStatus] = useState<'idle' | 'copied' | 'shared' | 'error'>('idle');

  const toggleSection = useCallback((i: number) => {
    setOpenSections((prev) => ({ ...prev, [i]: !prev[i] }));
  }, []);

  // Dedupe evidence URLs across all sections. Stable order: first-seen.
  const evidenceLinks = useMemo(() => {
    const seen = new Set<string>();
    const out: { url: string; title: string }[] = [];
    brief.sections.forEach((s) => {
      s.items.forEach((it) => {
        if (!it.evidence_url || seen.has(it.evidence_url)) return;
        seen.add(it.evidence_url);
        out.push({ url: it.evidence_url, title: it.title || it.article || it.evidence_url });
      });
    });
    return out;
  }, [brief]);

  // "3 things you need to know" — first three items from "What Changed
  // Today" (or fall back to the first section's items if there's no
  // "What Changed Today"). Stable, deterministic — the hero never invents
  // items the brief does not contain.
  const threeThings = useMemo(() => {
    const changed = brief.sections.find((s) => s.title === 'What Changed Today');
    const src = changed ?? brief.sections[0];
    if (!src) return [] as { title: string; description?: string }[];
    return src.items.slice(0, 3).map((it) => ({
      title: it.title || it.bill || 'Untitled development',
      description: it.change || it.description,
    }));
  }, [brief]);

  // Share handler — uses Web Share API where available (mobile), falls back
  // to clipboard write. The shared URL is the per-brief detail endpoint
  // (/api/v1/brief/{id}) proxied through the Next.js BFF as /brief/{id}.
  const onShare = useCallback(async () => {
    const shareUrl = `${window.location.origin}/briefing?id=${encodeURIComponent(brief.id)}`;
    const shareText = `${brief.headline}\n\n${shareUrl}`;
    try {
      if (typeof navigator !== 'undefined' && navigator.share) {
        await navigator.share({
          title: `Civic Daily Brief — ${brief.date}`,
          text: brief.headline,
          url: shareUrl,
        });
        setShareStatus('shared');
      } else if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(shareText);
        setShareStatus('copied');
      } else {
        setShareStatus('error');
      }
    } catch {
      // User dismissed the share sheet, or clipboard write blocked.
      setShareStatus('error');
    }
    // Reset after 2.5s so the button label doesn't get stuck on "Copied".
    setTimeout(() => setShareStatus('idle'), 2500);
  }, [brief]);

  return (
    <article className="pb-12" aria-labelledby="brief-headline">
      {/* Action bar (hidden in print) */}
      <div className="no-print mb-6 flex flex-wrap items-center gap-2 print:hidden">
        <button
          type="button"
          onClick={onShare}
          className="inline-flex items-center gap-1 rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm font-medium text-civic-ink hover:bg-civic-mist"
          aria-label="Share this brief"
        >
          <Share2 className="h-4 w-4" aria-hidden="true" />
          {shareStatus === 'copied' && 'Link copied'}
          {shareStatus === 'shared' && 'Shared'}
          {shareStatus === 'idle' && 'Share'}
          {shareStatus === 'error' && 'Copy failed — try again'}
        </button>
        <Link
          href="/notifications"
          className="inline-flex items-center gap-1 rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm font-medium text-civic-ink hover:bg-civic-mist"
          aria-label="Subscribe to the daily email brief"
        >
          <Mail className="h-4 w-4" aria-hidden="true" />
          Subscribe to daily email
        </Link>
        <Link
          href="/following"
          className="inline-flex items-center gap-1 rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm font-medium text-civic-ink hover:bg-civic-mist"
          aria-label="Manage what you follow — personalise your brief"
        >
          <Eye className="h-4 w-4" aria-hidden="true" />
          Personalise
        </Link>
      </div>

      {/* "3 things you need to know" — bulleted hero summary */}
      <section aria-label="3 things you need to know" className="mb-8 rounded-xl border border-civic-border bg-civic-paper p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">3 things you need to know</h2>
        {threeThings.length === 0 ? (
          <p className="mt-2 text-sm text-civic-stone">
            No verified developments to highlight today. Check back tomorrow or browse the archive.
          </p>
        ) : (
          <ul className="mt-3 space-y-2 text-sm text-civic-ink">
            {threeThings.map((t, i) => (
              <li key={i} className="flex gap-2">
                <span className="font-serif text-base font-semibold text-civic-leaf" aria-hidden="true">
                  {i + 1}.
                </span>
                <span>
                  <span className="font-medium">{t.title}</span>
                  {t.description ? (
                    <>
                      {' — '}
                      <span className="text-civic-stone">{t.description}</span>
                    </>
                  ) : null}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Section navigator (mobile swipe strip — desktop pill row) */}
      <nav
        aria-label="Brief sections"
        className="no-print mb-4 -mx-4 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:overflow-visible print:hidden"
      >
        {brief.sections.map((s, i) => {
          const Icon = SECTION_ICONS[s.title] ?? FileText;
          return (
            <a
              key={s.title}
              href={`#section-${i}`}
              className="inline-flex shrink-0 items-center gap-1 rounded-full border border-civic-border bg-civic-paper px-3 py-1.5 text-xs font-medium text-civic-stone hover:border-civic-leaf hover:text-civic-leaf"
            >
              <Icon className="h-3 w-3" aria-hidden="true" />
              {s.title}
            </a>
          );
        })}
      </nav>

      {/* Sections — mobile swipe container with scroll-snap, desktop stack */}
      <div className="space-y-6 sm:space-y-6">
        {brief.sections.map((section, i) => (
          <BriefSectionView
            key={section.title}
            index={i}
            section={section}
            open={!!openSections[i]}
            onToggle={() => toggleSection(i)}
          />
        ))}
      </div>

      {/* AI Summary — clearly labelled with ASSUMPTION reality badge */}
      <section
        aria-label="AI-generated summary"
        className="mt-8 rounded-xl border border-amber-300 bg-amber-50 p-5 print:border-civic-border print:bg-civic-paper"
      >
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-amber-800" aria-hidden="true" />
          <h2 className="font-serif text-lg font-semibold text-civic-ink">AI Summary</h2>
          <RealityBadge kind="ASSUMPTION" />
        </div>
        <div className="mt-3 whitespace-pre-line text-sm leading-relaxed text-civic-ink">
          {brief.ai_summary}
        </div>
        <p className="mt-3 border-t border-amber-300 pt-2 text-xs text-amber-900 print:text-civic-stone">
          {brief.ai_disclaimer}
          {brief.ai_source === 'template-fallback' && (
            <>
              {' '}
              <span className="italic">
                (AI service unavailable — this is an auto-generated summary, not a model output.)
              </span>
            </>
          )}
        </p>
      </section>

      {/* Evidence footer — every source URL cited in the brief, deduped */}
      <section
        aria-label="Evidence — primary sources cited"
        className="mt-8 rounded-xl border border-civic-border bg-civic-mist p-5"
      >
        <div className="flex items-center justify-between">
          <h2 className="font-serif text-lg font-semibold text-civic-ink">Evidence</h2>
          <span className="rounded-full bg-civic-paper px-2 py-0.5 text-xs font-medium text-civic-stone">
            {brief.evidence_count} citation{brief.evidence_count === 1 ? '' : 's'}
          </span>
        </div>
        <p className="mt-2 text-xs text-civic-stone">
          Every claim in this brief links to a primary source. Verify any claim against the linked source before relying on it.
        </p>
        {evidenceLinks.length === 0 ? (
          <p className="mt-3 text-sm text-civic-stone">No evidence links in this brief.</p>
        ) : (
          <ol className="mt-3 space-y-1.5 text-xs">
            {evidenceLinks.map((e, i) => (
              <li key={`${e.url}-${i}`} className="flex items-start gap-2">
                <span className="font-mono text-civic-stone" aria-hidden="true">[{i + 1}]</span>
                <a
                  href={e.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-start gap-1 text-civic-leaf hover:underline"
                >
                  <span className="break-all">{e.url}</span>
                  <ExternalLink className="mt-0.5 h-3 w-3 shrink-0" aria-hidden="true" />
                </a>
              </li>
            ))}
          </ol>
        )}
      </section>

      <p className="mt-8 text-xs text-civic-stone">
        The Civic Daily Brief explains what happened without political
        persuasion. Every item links to evidence. We do not rank
        politicians, parties, or positions. Information that could not be
        verified from available authoritative sources is excluded.
      </p>
    </article>
  );
}

// BriefSectionView renders one titled section of the brief with a
// collapse/expand toggle. The header is a <button> for a11y (keyboard
// operable + screen-reader friendly). Print always renders expanded.
function BriefSectionView({
  index,
  section,
  open,
  onToggle,
}: {
  index: number;
  section: BriefSection;
  open: boolean;
  onToggle: () => void;
}) {
  const Icon = SECTION_ICONS[section.title] ?? FileText;
  return (
    <section
      id={`section-${index}`}
      // scroll-mt gives the in-page anchor some breathing room below the
      // sticky header when the section-nav links are tapped.
      className="scroll-mt-24 rounded-xl border border-civic-border bg-civic-paper p-5 print:break-inside-avoid"
      aria-labelledby={`section-${index}-title`}
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        aria-controls={`section-${index}-body`}
        className="flex w-full items-center justify-between gap-2 text-left no-print print:hidden"
      >
        <span className="inline-flex items-center gap-2">
          <Icon className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
          <h2
            id={`section-${index}-title`}
            className="font-serif text-lg font-semibold text-civic-ink"
          >
            {section.title}
          </h2>
        </span>
        <ChevronDown
          className={`h-5 w-5 text-civic-stone transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden="true"
        />
      </button>
      {/* Print-only header (always visible) */}
      <h2 className="hidden print:block font-serif text-lg font-semibold text-civic-ink">
        {section.title}
      </h2>
      <div
        id={`section-${index}-body`}
        className={open ? 'mt-3' : 'hidden print:mt-3 print:block'}
      >
        <p className="text-sm text-civic-stone">{section.summary}</p>

        {section.items.length === 0 ? (
          <p className="mt-3 text-sm text-civic-stone italic">No items in this section.</p>
        ) : (
          <ul className="mt-4 space-y-3">
            {section.items.map((it, i) => (
              <li
                key={`${it.type}-${i}`}
                className="border-l-2 border-civic-border pl-3"
              >
                <div className="flex flex-wrap items-center gap-2 text-xs">
                  <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 font-medium uppercase text-civic-leaf">
                    {it.type.replace(/_/g, ' ')}
                  </span>
                  {it.house ? (
                    <span className="text-civic-stone">{it.house}</span>
                  ) : null}
                  {it.date ? (
                    <span className="text-civic-stone">{it.date}</span>
                  ) : null}
                  {it.significance ? (
                    <span className="rounded-full bg-civic-acacia/20 px-2 py-0.5 font-medium text-civic-clay">
                      {it.significance}
                    </span>
                  ) : null}
                  {it.article ? (
                    <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium text-civic-ink">
                      {it.article}
                    </span>
                  ) : null}
                </div>
                {it.title ? (
                  <h3 className="mt-1.5 font-serif text-base font-semibold text-civic-ink">
                    {it.title}
                  </h3>
                ) : null}
                {it.description ? (
                  <p className="mt-1 text-sm text-civic-ink">{it.description}</p>
                ) : null}
                {it.change ? (
                  <p className="mt-1 text-sm text-civic-stone">
                    <span className="font-medium text-civic-ink">Change:</span> {it.change}
                  </p>
                ) : null}
                {it.text ? (
                  <blockquote className="mt-2 border-l-2 border-civic-border pl-3 text-sm italic text-civic-ink">
                    {it.text}
                  </blockquote>
                ) : null}
                {it.connection ? (
                  <p className="mt-2 text-sm text-civic-stone">
                    <span className="font-medium text-civic-ink">Why this matters:</span>{' '}
                    {it.connection}
                  </p>
                ) : null}
                {/* Every item MUST have an evidence_url — render the link. */}
                <a
                  href={it.evidence_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-2 inline-flex items-center gap-1 text-xs text-civic-leaf hover:underline"
                >
                  View evidence
                  <ExternalLink className="h-3 w-3" aria-hidden="true" />
                </a>
                {it.bill ? (
                  <Link
                    href={`/bills/${it.bill}`}
                    className="ml-3 inline-flex items-center gap-1 text-xs text-civic-leaf hover:underline"
                  >
                    View Bill
                    <ExternalLink className="h-3 w-3" aria-hidden="true" />
                  </Link>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
