import { notFound } from 'next/navigation';
import Link from 'next/link';
import { mockBills, mockTimeline } from '@/lib/mock-data';
import { TimelineView } from '@/components/timeline';
import { formatDate } from '@/lib/utils';
import type { Metadata } from 'next';
import type { Bill, BillEvent } from '@/lib/types';

// Force dynamic rendering — the timeline is fetched from the live Go BFF
// (which in turn discovers Bills from new.kenyalaw.org). A static export
// would freeze the timeline at build time and miss new publications.
export const dynamic = 'force-dynamic';
// Revalidate at most once per minute to keep the timeline fresh without
// hammering the upstream Kenyan source.
export const revalidate = 60;

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

async function fetchBill(billId: string): Promise<{ bill: Bill | null; source: string }> {
  try {
    const resp = await fetch(`${API_BASE}/api/v1/bills/${encodeURIComponent(billId)}`, {
      next: { revalidate: 60 },
    });
    if (!resp.ok) return { bill: null, source: 'api' };
    const bill = (await resp.json()) as Bill;
    return { bill, source: 'api' };
  } catch {
    const bill = mockBills.find((b) => b.id === billId) ?? null;
    return { bill, source: 'mock' };
  }
}

async function fetchBillTimeline(billId: string): Promise<{ events: BillEvent[]; source: string }> {
  try {
    const resp = await fetch(`${API_BASE}/api/v1/bills/${encodeURIComponent(billId)}/timeline`, {
      next: { revalidate: 60 },
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    return { events: data.events ?? [], source: data.source ?? 'api' };
  } catch {
    // Fall back to mock data so the page still renders during local dev
    // when the Go BFF isn't running.
    return {
      events: mockTimeline.filter((e) => e.bill_id === billId),
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
  // Try the live API first; fall back to mock for build-time metadata.
  const { bill } = await fetchBill(id);
  const mockBill = mockBills.find((b) => b.id === id);
  return { title: `Timeline · ${bill?.title ?? mockBill?.title ?? 'Bill'}` };
}

export default async function BillTimelinePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const { bill: liveBill } = await fetchBill(id);
  const { events, source } = await fetchBillTimeline(liveBill?.id ?? id);

  // Pick a bill to render the header against — prefer the live record,
  // fall back to mock so the page never 404s just because the API is down.
  const bill = liveBill ?? mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  // Normalize the API response into the BillEvent shape the TimelineView
  // component already understands. The Go handler returns fields with the
  // same names, but defensive normalization keeps the page robust to small
  // schema drift.
  const normalizedEvents: BillEvent[] = events.map((e, i) => ({
    id: e.id ?? `ev-${i}`,
    bill_id: e.bill_id ?? bill.id,
    event_type: e.event_type,
    date: e.date,
    date_is_approximate: e.date_is_approximate ?? false,
    house: e.house ?? bill.house_name ?? bill.house ?? null,
    description: e.description,
    source_url: e.source_url,
    confidence: e.confidence ?? 'unknown',
    note: e.note ?? null,
  }));

  return (
    <article className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Timeline</li>
        </ol>
      </nav>

      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Timeline</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Verified events for <strong className="text-civic-ink">{bill.title}</strong>.
          Every event links to its source. Dates that could not be confirmed are
          preserved as approximate or unknown — never fabricated.
        </p>
        {source === 'mock' && (
          <p className="mt-2 rounded-md bg-civic-acacia/10 px-3 py-1.5 text-xs text-civic-clay">
            Showing sample timeline — connect the Go API (port 9000) for verified events from kenyalaw.org
          </p>
        )}
        {source !== 'mock' && (
          <p className="mt-2 rounded-md bg-civic-leaf/10 px-3 py-1.5 text-xs text-civic-leaf">
            ✅ Live data from {source} — {normalizedEvents.length} verified event(s)
          </p>
        )}
      </header>

      <TimelineView events={normalizedEvents} />

      <p className="mt-8 text-xs text-civic-stone">
        Last updated {formatDate(bill.updated_at ?? bill.publication_date)} · {normalizedEvents.length} verified event(s).
      </p>
    </article>
  );
}
