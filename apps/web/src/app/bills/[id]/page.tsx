import Link from 'next/link';
import {
  FileText,
  GitCompare,
  History,
  MessageCircle,
  Bell,
  AlertCircle,
} from 'lucide-react';
import { TimelineView } from '@/components/timeline';
import { BillAskPanel } from '@/components/bill-ask-panel';
import { getJSON_ as getJSON, ApiError } from '@/lib/api';
import { mockTimeline } from '@/lib/mock-data';
import type { Bill, BillEvent } from '@/lib/types';
import type { Metadata } from 'next';

/**
 * Bill detail page (spec §13 — Bill Intelligence).
 *
 * Server component that fetches the Bill from `/api/v1/bills/{id}` at
 * request time. If the API is unreachable, shows a friendly error rather
 * than rendering mock data — the audit explicitly flagged this page for
 * silently falling back to `mockBills` (GAP-13-1).
 *
 * The timeline preview fetches from `/api/v1/bills/{id}/timeline` and
 * falls back to the curated mock events ONLY when the API is unreachable,
 * so the page still renders something useful during local dev.
 */

// Force dynamic rendering — Bill data lives behind the live Go BFF which
// discovers Bills from kenyalaw.org; a static export would freeze the
// page at build time and miss any subsequent publications.
export const dynamic = 'force-dynamic';
export const revalidate = 60; // ISR-friendly cache, max 1 minute.

const API_BASE =
  process.env.API_BASE_URL ?? 'http://localhost:9000';

async function fetchBill(id: string): Promise<Bill | null> {
  try {
    return await getJSON<Bill>(`${API_BASE}/api/v1/bills/${encodeURIComponent(id)}`);
  } catch (err) {
    // Surface unexpected errors to logs; the friendly error UI is rendered
    // by the caller. ApiError carries the status code so callers can branch.
    if (err instanceof ApiError) {
      // eslint-disable-next-line no-console
      console.warn(`[bills/[id]] API error ${err.status} for ${id}: ${err.message}`);
    } else {
      // eslint-disable-next-line no-console
      console.warn(`[bills/[id]] fetch failed for ${id}`, err);
    }
    return null;
  }
}

async function fetchTimeline(
  id: string,
): Promise<{ events: BillEvent[]; source: 'api' | 'mock' }> {
  try {
    const r = await getJSON<{ events: BillEvent[] }>(
      `${API_BASE}/api/v1/bills/${encodeURIComponent(id)}/timeline`,
    );
    return { events: r.events ?? [], source: 'api' };
  } catch {
    // Fall back to mock timeline events so the preview section still
    // renders during local dev when the Go BFF is offline. Real Bills
    // (from the API) won't match mock timeline entries, so the fallback
    // is empty in production — no misleading data is shown.
    return {
      events: mockTimeline.filter((e) => e.bill_id === id),
      source: 'mock',
    };
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const bill = await fetchBill(id);
  if (!bill) return { title: 'Bill not found' };
  return {
    title: bill.title,
    description: bill.purpose ?? bill.description ?? bill.title,
  };
}

export default async function BillDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const bill = await fetchBill(id);

  if (!bill) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
        <div className="rounded-xl border border-amber-300 bg-amber-50 p-6 text-amber-900">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5 flex-shrink-0" aria-hidden="true" />
            <h1 className="font-serif text-xl font-semibold">
              This Bill could not be loaded
            </h1>
          </div>
          <p className="mt-2 text-sm">
            The Bill data API is unreachable right now, or this Bill does not
            exist. Please try again in a few minutes — every Bill is fetched
            live from <span className="font-medium">kenyalaw.org</span> via the
            Go BFF and may be unavailable briefly during ingestion.
          </p>
          <Link
            href="/bills"
            className="mt-4 inline-flex items-center gap-2 rounded-md bg-civic-forest px-4 py-2 text-sm font-semibold text-civic-paper hover:bg-civic-leaf"
          >
            ← Back to Bills
          </Link>
        </div>
      </div>
    );
  }

  const { events: timelineEvents, source: timelineSource } = await fetchTimeline(bill.id);

  return (
    <article className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      {/* Breadcrumb */}
      <nav aria-label="Breadcrumb" className="mb-6 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/" className="hover:text-civic-leaf">Home</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">{bill.identifier}</li>
        </ol>
      </nav>

      {/* Header — title + status */}
      <header className="mb-8">
        <div className="flex flex-wrap items-center gap-2 text-xs text-civic-stone">
          <span className="rounded-full bg-civic-mist px-3 py-1 font-medium text-civic-ink">{bill.identifier}</span>
          <span>{bill.year}</span>
          <span aria-hidden="true">·</span>
          <span>{bill.house_name ?? bill.house}</span>
          {bill.committee_name && (
            <>
              <span aria-hidden="true">·</span>
              <span>{bill.committee_name}</span>
            </>
          )}
        </div>
        <h1 className="mt-4 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          {bill.title}
        </h1>
        {bill.sponsor_name && (
          <p className="mt-3 text-sm text-civic-stone">Sponsored by {bill.sponsor_name}</p>
        )}
        <div className="mt-5 flex flex-wrap items-center gap-2">
          <span className="rounded-full bg-civic-leaf px-4 py-1.5 text-xs font-semibold text-white">
            {bill.status.replace('_', ' ')}
          </span>
          {bill.current_stage && (
            <span className="rounded-full bg-civic-acacia/20 px-4 py-1.5 text-xs font-medium text-civic-ink">
              Stage: {bill.current_stage}
            </span>
          )}
          {bill.topics?.map((t) => (
            <span key={t} className="rounded-full bg-civic-mist px-2.5 py-1 text-xs text-civic-stone">{t}</span>
          ))}
        </div>
      </header>

      {/* Quick Explanation */}
      <section className="mb-8 rounded-xl border border-civic-border bg-civic-mist/50 p-6">
        <h2 className="font-serif text-xl font-semibold text-civic-forest">In plain language</h2>
        <p className="mt-3 text-civic-ink leading-relaxed">
          {bill.description ?? bill.purpose ?? 'No plain-language summary available yet.'}
        </p>
        {bill.current_stage_simple_explanation && (
          <div className="mt-5 border-l-2 border-civic-acacia pl-4">
            <h3 className="text-sm font-semibold text-civic-ink">What this stage means</h3>
            <p className="mt-1 text-sm text-civic-stone">{bill.current_stage_simple_explanation}</p>
          </div>
        )}
      </section>

      {/* Tab nav */}
      <nav className="mb-8 border-b border-civic-border" aria-label="Bill sections">
        <ul className="flex flex-wrap gap-1">
          {[
            { href: '', icon: FileText, label: 'Overview', active: true },
            { href: '/timeline', icon: History, label: 'Timeline' },
            { href: '/versions', icon: FileText, label: 'Versions' },
            { href: '/compare', icon: GitCompare, label: 'Compare' },
            { href: '/documents', icon: FileText, label: 'Documents' },
            { href: '/chat', icon: MessageCircle, label: 'Ask' },
          ].map(({ href, icon: Icon, label, active }) => (
            <li key={label}>
              <Link
                href={`/bills/${bill.id}${href}`}
                className={`inline-flex items-center gap-2 border-b-2 px-5 py-3.5 text-sm transition ${
                  active
                    ? 'border-civic-leaf font-semibold text-civic-leaf'
                    : 'border-transparent text-civic-stone hover:text-civic-leaf'
                }`}
              >
                <Icon className="h-4 w-4" aria-hidden="true" />
                {label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      {/* Body — main content + sidebar */}
      <div className="grid gap-10 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-10">
          {/* Purpose */}
          {bill.purpose && (
            <section>
              <h2 className="font-serif text-xl font-semibold text-civic-forest">Purpose</h2>
              <p className="mt-3 text-civic-ink leading-relaxed">{bill.purpose}</p>
            </section>
          )}

          {/* Timeline preview */}
          <section>
            <div className="flex items-center justify-between">
              <h2 className="font-serif text-xl font-semibold text-civic-forest">Timeline</h2>
              <Link href={`/bills/${bill.id}/timeline`} className="text-sm text-civic-leaf hover:underline">
                Full timeline →
              </Link>
            </div>
            {timelineSource === 'mock' && (
              <p className="mt-2 rounded-md bg-civic-acacia/10 px-3 py-1.5 text-xs text-civic-clay">
                Showing sample timeline — connect the Go API (port 9000) for verified events from kenyalaw.org
              </p>
            )}
            {timelineEvents.length === 0 ? (
              <p className="mt-3 text-sm text-civic-stone">
                No verified timeline events yet. The Bill may have been recently published.
              </p>
            ) : (
              <TimelineView events={timelineEvents} limit={5} />
            )}
          </section>

          {/* Ask about this Bill */}
          <section>
            <h2 className="font-serif text-xl font-semibold text-civic-forest">Ask about this Bill</h2>
            <p className="mt-1 text-sm text-civic-stone">
              Every answer is evidence-grounded and cites its sources.
            </p>
            <BillAskPanel billId={bill.id} />
          </section>
        </div>

        {/* Sidebar */}
        <aside className="space-y-6">
          <div className="rounded-xl border border-civic-border bg-civic-paper p-5">
            <h3 className="text-sm font-semibold text-civic-ink">People</h3>
            <ul className="mt-3 space-y-1.5 text-sm text-civic-stone">
              <li>Sponsor: {bill.sponsor_name ?? '—'}</li>
              <li>Committee: {bill.committee_name ?? '—'}</li>
              <li>House: {bill.house_name ?? bill.house ?? '—'}</li>
            </ul>
          </div>

          <div className="rounded-xl border border-civic-border bg-civic-paper p-5">
            <h3 className="text-sm font-semibold text-civic-ink">Documents</h3>
            <p className="mt-2 text-sm text-civic-stone">
              {bill.version_count} version(s) · {bill.citation_count} citation(s)
            </p>
            <Link
              href={`/bills/${bill.id}/documents`}
              className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
            >
              View documents →
            </Link>
          </div>

          <div className="rounded-xl border border-civic-border bg-civic-paper p-5">
            <h3 className="text-sm font-semibold text-civic-ink">Follow this Bill</h3>
            <p className="mt-2 text-sm text-civic-stone">
              Get notified when this Bill changes stage, is amended, or a new document is published.
            </p>
            <button
              type="button"
              className="mt-3 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-civic-leaf px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-civic-leaf/90"
              data-bill-id={bill.id}
            >
              <Bell className="h-4 w-4" aria-hidden="true" />
              Follow
            </button>
          </div>

          <div className="rounded-xl border border-civic-border bg-civic-paper p-5 text-xs text-civic-stone">
            <p>
              <strong className="text-civic-ink">Disclaimer:</strong> Summaries are AI-generated and
              citation-validated. Not legal advice. Always consult source documents.
            </p>
          </div>
        </aside>
      </div>
    </article>
  );
}
