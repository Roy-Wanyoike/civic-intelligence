import { notFound } from 'next/navigation';
import Link from 'next/link';
import { GitCompare, Info } from 'lucide-react';
import { mockBills } from '@/lib/mock-data';
import type { Metadata } from 'next';
import type { Bill } from '@/lib/types';

// Force dynamic rendering — the available version set is fetched from the
// Go BFF (which in turn discovers Bills from new.kenyalaw.org). A static
// export would freeze the page at build time.
export const dynamic = 'force-dynamic';
export const revalidate = 60;

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

interface BillChange {
  kind: 'addition' | 'removal' | 'modification' | string;
  section?: string;
  description?: string;
  before?: string;
  after?: string;
  source_url?: string;
  confidence?: 'high' | 'medium' | 'low' | 'unknown';
}

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

async function fetchBillChanges(
  billId: string,
): Promise<{ changes: BillChange[]; note: string | null; source: string }> {
  try {
    const resp = await fetch(`${API_BASE}/api/v1/bills/${encodeURIComponent(billId)}/changes`, {
      next: { revalidate: 60 },
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    return {
      changes: data.changes ?? [],
      note: data.note ?? null,
      source: data.source ?? 'api',
    };
  } catch {
    // Fall back to empty changes + a transparent note when the API is
    // unavailable (local dev without the Go backend running). This keeps
    // the page shape identical regardless of whether the BFF is up.
    return {
      changes: [],
      note: 'Version comparison requires multiple Bill versions in the database',
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
  const { bill } = await fetchBill(id);
  const mockBill = mockBills.find((b) => b.id === id);
  return { title: `Compare · ${bill?.title ?? mockBill?.title ?? 'Bill'}` };
}

export default async function BillComparePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const { bill: liveBill } = await fetchBill(id);
  const { changes, note, source } = await fetchBillChanges(liveBill?.id ?? id);

  // Pick a bill to render the header against — prefer the live record,
  // fall back to mock so the page never 404s just because the API is down.
  const bill = liveBill ?? mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  const hasChanges = changes.length > 0;

  return (
    <article className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Compare</li>
        </ol>
      </nav>

      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Compare versions</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Pick two versions to see exactly what changed. Additions, removals, and
          modifications are shown with a plain-language explanation. We never
          speculate on <em>why</em> a change was made unless the source explicitly
          states the reason.
        </p>
        {source === 'mock' && (
          <p className="mt-2 rounded-md bg-civic-acacia/10 px-3 py-1.5 text-xs text-civic-clay">
            Showing sample data — connect the Go API (port 9000) for live Bill versions
          </p>
        )}
        {source !== 'mock' && (
          <p className="mt-2 rounded-md bg-civic-leaf/10 px-3 py-1.5 text-xs text-civic-leaf">
            ✅ Live data from {source} — {changes.length} change(s) detected
          </p>
        )}
      </header>

      <div className="mt-8 grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <GitCompare className="h-6 w-6 text-civic-leaf" aria-hidden="true" />
          <h2 className="mt-2 font-serif text-lg font-semibold">Side-by-side</h2>
          <p className="mt-1 text-sm text-civic-stone">
            View both versions in parallel with additions and removals highlighted inline.
          </p>
        </div>
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <GitCompare className="h-6 w-6 text-civic-acacia" aria-hidden="true" />
          <h2 className="mt-2 font-serif text-lg font-semibold">Clause-level diff</h2>
          <p className="mt-1 text-sm text-civic-stone">
            Grouped by clause, with affected clauses listed at the top.
          </p>
        </div>
      </div>

      {hasChanges ? (
        <section className="mt-8" aria-label="Detected changes">
          <h2 className="font-serif text-xl font-semibold text-civic-forest">Detected changes</h2>
          <ul className="mt-4 space-y-3">
            {changes.map((c, i) => (
              <li key={i} className="rounded-lg border border-civic-border bg-civic-paper p-4">
                <div className="flex items-center gap-2 text-xs">
                  <span className="rounded-full bg-civic-mist px-2 py-0.5 font-semibold text-civic-ink">{c.kind}</span>
                  {c.section && <span className="text-civic-stone">{c.section}</span>}
                  {c.confidence && (
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${
                      c.confidence === 'high' ? 'bg-civic-leaf/20 text-civic-leaf' :
                      c.confidence === 'medium' ? 'bg-civic-acacia/20 text-civic-clay' :
                      'bg-civic-stone/20 text-civic-stone'
                    }`}>{c.confidence}</span>
                  )}
                </div>
                {c.description && <p className="mt-2 text-sm text-civic-ink">{c.description}</p>}
                {(c.before || c.after) && (
                  <dl className="mt-2 grid gap-2 text-sm sm:grid-cols-2">
                    {c.before && (
                      <div>
                        <dt className="text-xs font-semibold uppercase text-civic-clay">Before</dt>
                        <dd className="text-civic-stone line-through">{c.before}</dd>
                      </div>
                    )}
                    {c.after && (
                      <div>
                        <dt className="text-xs font-semibold uppercase text-civic-leaf">After</dt>
                        <dd className="text-civic-ink">{c.after}</dd>
                      </div>
                    )}
                  </dl>
                )}
                {c.source_url && (
                  <a href={c.source_url} target="_blank" rel="noopener noreferrer" className="mt-2 inline-flex items-center gap-0.5 text-xs text-civic-leaf hover:underline">
                    Source <GitCompare className="h-3 w-3" aria-hidden="true" />
                  </a>
                )}
              </li>
            ))}
          </ul>
        </section>
      ) : (
        <section className="mt-8" aria-label="No changes available">
          <div className="rounded-lg border border-dashed border-civic-border bg-civic-paper p-8 text-center">
            <Info className="mx-auto h-8 w-8 text-civic-stone" aria-hidden="true" />
            <h2 className="mt-3 font-serif text-lg font-semibold text-civic-forest">No version changes available yet</h2>
            {note && <p className="mt-2 text-sm text-civic-stone">{note}</p>}
            <p className="mt-4 text-xs text-civic-stone">
              Select two versions on the <Link href={`/bills/${bill.id}/versions`} className="text-civic-leaf hover:underline">Versions</Link> page to begin a comparison once multiple versions are stored.
            </p>
          </div>
        </section>
      )}
    </article>
  );
}
