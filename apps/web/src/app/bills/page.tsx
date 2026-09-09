import Link from 'next/link';
import { Filter } from 'lucide-react';
import { mockBills } from '@/lib/mock-data';
import { formatDate } from '@/lib/utils';
import { FilterSelect } from '@/components/filter-select';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Bills',
  description: 'Browse Kenyan Bills before Parliament — with plain-language explanations and verified timelines.',
};

const STATUS_FILTERS = ['all', 'in_progress', 'enacted', 'withdrawn'] as const;
const HOUSE_FILTERS = ['all', 'National Assembly', 'Senate'] as const;
const TOPIC_FILTERS = ['all', 'housing', 'data-protection', 'public-finance', 'education', 'transport', 'county-government'] as const;

export default function BillsPage({
  searchParams,
}: {
  searchParams: { status?: string; house?: string; topic?: string; q?: string };
}) {
  let bills = mockBills.slice();
  const status = searchParams.status ?? 'all';
  const house = searchParams.house ?? 'all';
  const topic = searchParams.topic ?? 'all';
  const q = searchParams.q?.toLowerCase().trim();
  if (status !== 'all') bills = bills.filter(b => b.status === status);
  if (house !== 'all') bills = bills.filter(b => b.house_name === house);
  if (topic !== 'all') bills = bills.filter(b => b.topics?.includes(topic));
  if (q) {
    bills = bills.filter(
      b => b.title.toLowerCase().includes(q) || b.identifier.toLowerCase().includes(q) || (b.purpose?.toLowerCase().includes(q) ?? false),
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Bills before Parliament</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Each Bill links to a plain-language explanation, verified timeline, and the original documents. No claim is published without an authoritative source.
        </p>
      </header>

      <form className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4" aria-label="Filter Bills">
        <FilterSelect label="Status" name="status" value={status} options={STATUS_FILTERS} />
        <FilterSelect label="House" name="house" value={house} options={HOUSE_FILTERS} />
        <FilterSelect label="Topic" name="topic" value={topic} options={TOPIC_FILTERS} />
        <div>
          <label htmlFor="q" className="sr-only">Search within Bills</label>
          <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-paper px-3 py-2">
            <Filter className="h-4 w-4 text-civic-stone" aria-hidden="true" />
            <input id="q" type="search" name="q" defaultValue={searchParams.q ?? ''} placeholder="Search within Bills" className="flex-1 bg-transparent text-sm focus:outline-none" />
          </div>
        </div>
      </form>

      {bills.length === 0 ? (
        <div className="rounded-lg border border-dashed border-civic-border p-12 text-center text-civic-stone">
          No Bills match the current filters.
        </div>
      ) : (
        <ul className="grid gap-4 md:grid-cols-2">
          {bills.map((b) => (
            <li key={b.id} className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf">
              <Link href={`/bills/${b.id}`} className="block">
                <div className="flex items-center justify-between gap-2 text-xs text-civic-stone">
                  <span>{b.identifier} · {b.year}</span>
                  <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium text-civic-ink">{b.status.replace('_', ' ')}</span>
                </div>
                <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">{b.title}</h2>
                {b.current_stage && (<p className="mt-2 text-sm text-civic-stone"><span className="font-medium text-civic-ink">Current stage:</span> {b.current_stage}</p>)}
                {b.purpose && (<p className="mt-2 line-clamp-2 text-sm text-civic-stone">{b.purpose}</p>)}
                <div className="mt-4 flex items-center justify-between text-xs text-civic-stone">
                  <span>Updated {formatDate(b.updated_at)}</span>
                  <span>{b.citation_count} citations</span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

