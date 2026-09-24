import { notFound } from 'next/navigation';
import Link from 'next/link';
import { Scale, ExternalLink } from 'lucide-react';
import { mockBills } from '@/lib/mock-data';
import { getJSON_ as getJSON, ApiError } from '@/lib/api';
import { formatDate } from '@/lib/utils';
import type { Metadata } from 'next';
import type { Amendment, AmendmentStatus, Bill } from '@/lib/types';

/**
 * Amendments tab (issue #293 / FEAT-15).
 *
 * Server component that fetches the parent Bill from `/api/v1/bills/{id}`
 * and the amendments list from `/api/v1/bills/{id}/amendments`. Both calls
 * fall back to mock data when the Go BFF is offline so the page still
 * renders during local dev. The amendments tracker is seed-only for
 * FEAT-15 — the live Hansard committee-stage ingestion path is pending
 * issue #19 + ADR-0011.
 *
 * The page renders each amendment as a card with a status badge
 * (proposed = amber, accepted = green, rejected = red) + the sponsor +
 * the committee sitting date + the one-line title + the plain-language
 * summary. Every card links to the official source document.
 */

// Force dynamic rendering — the amendments list is fetched from the
// live Go BFF. A static export would freeze the page at build time and
// miss any new amendments proposed during a sitting.
export const dynamic = 'force-dynamic';
export const revalidate = 60; // ISR-friendly cache, max 1 minute.

const API_BASE =
  process.env.API_BASE_URL ?? 'http://localhost:9000';

// fetchBill mirrors the helper in /bills/[id]/page.tsx. Returns the Bill
// from the live API or falls back to the mock roster on any error so
// the amendments page never 500s just because the BFF is offline.
async function fetchBill(billId: string): Promise<{ bill: Bill | null; source: 'api' | 'mock' }> {
  try {
    const bill = await getJSON<Bill>(`${API_BASE}/api/v1/bills/${encodeURIComponent(billId)}`);
    return { bill, source: 'api' };
  } catch (err) {
    if (err instanceof ApiError) {
      // eslint-disable-next-line no-console
      console.warn(`[bills/[id]/amendments] API error ${err.status} for ${billId}: ${err.message}`);
    } else {
      // eslint-disable-next-line no-console
      console.warn(`[bills/[id]/amendments] fetch failed for ${billId}`, err);
    }
    const bill = mockBills.find((b) => b.id === billId) ?? null;
    return { bill, source: 'mock' };
  }
}

// fetchBillAmendments returns the amendments list for a Bill. Falls back
// to an empty list when the API is unreachable — the page renders an
// explanatory note instead of a stack trace.
async function fetchBillAmendments(
  billId: string,
): Promise<{ amendments: Amendment[]; source: 'api' | 'mock'; note?: string }> {
  try {
    const r = await getJSON<{
      amendments: Amendment[];
      total: number;
      source: string;
      note?: string;
    }>(`${API_BASE}/api/v1/bills/${encodeURIComponent(billId)}/amendments`);
    return {
      amendments: r.amendments ?? [],
      source: 'api',
      note: r.note,
    };
  } catch (err) {
    if (err instanceof ApiError) {
      // eslint-disable-next-line no-console
      console.warn(`[bills/[id]/amendments] amendments API error ${err.status} for ${billId}: ${err.message}`);
    } else {
      // eslint-disable-next-line no-console
      console.warn(`[bills/[id]/amendments] amendments fetch failed for ${billId}`, err);
    }
    return { amendments: [], source: 'mock' };
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const { bill } = await fetchBill(id);
  const mockBill = mockBills.find((b) => b.id === id);
  return { title: `Amendments · ${bill?.title ?? mockBill?.title ?? 'Bill'}` };
}

// statusBadge returns the Tailwind classes + human label for an
// amendment's lifecycle status. Proposed amendments surface as amber
// (in-flight, attention needed); accepted as green (carried); rejected
// as red (defeated). The colour mapping mirrors the platform's
// evidence-grounded palette — green = verified fact, amber = needs
// attention, red = contested.
function statusBadge(status: AmendmentStatus): { label: string; className: string } {
  switch (status) {
    case 'proposed':
      return {
        label: 'Proposed',
        className: 'bg-amber-100 text-amber-900 border-amber-300',
      };
    case 'accepted':
      return {
        label: 'Accepted',
        className: 'bg-civic-leaf/15 text-civic-leaf border-civic-leaf/30',
      };
    case 'rejected':
      return {
        label: 'Rejected',
        className: 'bg-red-100 text-red-900 border-red-300',
      };
    default:
      // Defensive: the API contract guarantees one of the three enum
      // values above, but we render an unknown-status badge rather than
      // crashing if the API ever returns a new value before this code
      // is updated.
      return {
        label: String(status) || 'Unknown',
        className: 'bg-civic-mist text-civic-stone border-civic-border',
      };
  }
}

export default async function BillAmendmentsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const { bill: liveBill } = await fetchBill(id);
  const { amendments, source, note } = await fetchBillAmendments(liveBill?.id ?? id);

  // Pick a bill to render the header against — prefer the live record,
  // fall back to mock so the page never 404s just because the API is down.
  const bill = liveBill ?? mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  // Sort amendments by proposed_at (oldest first) so the page reads in
  // chronological order — the same convention as the timeline page.
  const sortedAmendments = [...amendments].sort((a, b) => {
    const ta = Date.parse(a.proposed_at);
    const tb = Date.parse(b.proposed_at);
    if (Number.isNaN(ta) && Number.isNaN(tb)) return 0;
    if (Number.isNaN(ta)) return 1;
    if (Number.isNaN(tb)) return -1;
    return ta - tb;
  });

  return (
    <article className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Amendments</li>
        </ol>
      </nav>

      <header>
        <div className="flex items-center gap-2">
          <Scale className="h-6 w-6 text-civic-clay" aria-hidden="true" />
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">
            Amendments
          </h1>
        </div>
        <p className="mt-2 text-sm text-civic-stone">
          Every amendment proposed against{' '}
          <strong className="text-civic-ink">{bill.title}</strong> during
          its committee stage or third reading. Each amendment carries a
          lifecycle status — <em>proposed</em> (tabled, not yet voted),{' '}
          <em>accepted</em> (carried), or <em>rejected</em> (defeated) —
          and a one-line summary of the substantive change.
        </p>
        {source === 'api' && note && (
          <p className="mt-2 rounded-md bg-civic-acacia/10 px-3 py-1.5 text-xs text-civic-clay">
            {note}
          </p>
        )}
        {source === 'mock' && (
          <p className="mt-2 rounded-md bg-civic-acacia/10 px-3 py-1.5 text-xs text-civic-clay">
            Could not reach the Go BFF (port 9000) — showing an empty
            amendments list. Start the API to see the seed amendments.
          </p>
        )}
        {source === 'api' && sortedAmendments.length > 0 && (
          <p className="mt-2 rounded-md bg-civic-leaf/10 px-3 py-1.5 text-xs text-civic-leaf">
            ✅ Live data — {sortedAmendments.length} amendment(s) tracked
          </p>
        )}
      </header>

      {sortedAmendments.length === 0 ? (
        <p className="mt-8 rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
          No amendments have been proposed against this Bill yet. Once the
          Bill enters committee stage, proposed amendments will appear
          here.
        </p>
      ) : (
        <ul className="mt-8 space-y-4">
          {sortedAmendments.map((a) => {
            const badge = statusBadge(a.status);
            return (
              <li
                key={a.id}
                className="rounded-xl border border-civic-border bg-civic-paper p-5 shadow-sm"
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold ${badge.className}`}
                      >
                        {badge.label}
                      </span>
                      <span className="text-xs text-civic-stone">
                        Proposed {formatDate(a.proposed_at)}
                      </span>
                    </div>
                    <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                      {a.title}
                    </h2>
                    <p className="mt-2 text-sm leading-relaxed text-civic-stone">
                      {a.summary}
                    </p>
                    <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-civic-stone">
                      <span>
                        Sponsor:{' '}
                        <Link
                          href={`/people/${encodeURIComponent(a.proposed_by)}`}
                          className="font-medium text-civic-leaf hover:underline"
                        >
                          {a.proposed_by}
                        </Link>
                      </span>
                      {a.source_url && (
                        <a
                          href={a.source_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1 font-medium text-civic-leaf hover:underline"
                        >
                          Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                        </a>
                      )}
                    </div>
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}

      <p className="mt-8 text-xs text-civic-stone">
        Last updated {formatDate(bill.updated_at ?? bill.publication_date)} ·{' '}
        {sortedAmendments.length} amendment(s) tracked.
      </p>
    </article>
  );
}
