import Link from 'next/link';
import { ExternalLink, Filter, Gift } from 'lucide-react';
import { governmentGrants } from '@/lib/financial-data';
import { formatDate } from '@/lib/utils';
import { FilterSelect } from '@/components/filter-select';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Government Grants',
  description:
    'Track every grant the Government of Kenya has been promised or has received since September 2022 — evidence-backed, with source URLs.',
};

const STATUS_FILTERS = ['all', 'announced', 'pending', 'disbursed'] as const;
const DONOR_FILTERS = [
  'all',
  'World Bank (IDA)',
  'The Global Fund',
  'European Union (Global Gateway)',
] as const;
const SECTOR_FILTERS = [
  'all',
  'fiscal',
  'education',
  'health',
  'multi-sector',
  'digital',
] as const;

function formatUSD(amount: number | null | undefined): string {
  if (amount == null) return '\u2014';
  return new Intl.NumberFormat('en-KE', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(amount);
}

function formatKES(amount: number | null | undefined): string {
  if (amount == null) return '\u2014';
  return new Intl.NumberFormat('en-KE', {
    style: 'currency',
    currency: 'KES',
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(amount);
}

export default function GrantsPage({
  searchParams,
}: {
  searchParams: { donor?: string; sector?: string; status?: string; q?: string };
}) {
  let grants = governmentGrants.slice();
  const status = searchParams.status ?? 'all';
  const donor = searchParams.donor ?? 'all';
  const sector = searchParams.sector ?? 'all';
  const q = searchParams.q?.toLowerCase().trim();
  if (status !== 'all') grants = grants.filter((g) => g.status === status);
  if (donor !== 'all') grants = grants.filter((g) => g.donor === donor);
  if (sector !== 'all') grants = grants.filter((g) => g.sector === sector);
  if (q) {
    grants = grants.filter(
      (g) =>
        g.donor.toLowerCase().includes(q) ||
        g.purpose.toLowerCase().includes(q) ||
        g.sector.toLowerCase().includes(q),
    );
  }

  // Sort newest announcement first.
  grants.sort((a, b) => {
    const ax = a.announcement_date ? Date.parse(a.announcement_date) : 0;
    const bx = b.announcement_date ? Date.parse(b.announcement_date) : 0;
    return bx - ax;
  });

  const totalUSD = grants.reduce((sum, g) => sum + (g.amount_usd ?? 0), 0);

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Gift className="h-4 w-4" aria-hidden="true" />
          <span>Transparency Tracker</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Government Grants since September 2022
        </h1>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          Every grant the Government of Kenya has been awarded or has received since
          the Ruto administration took office on 13 September 2022. Each entry links
          to a credible public source &mdash; no fabricated amounts.
        </p>
        {grants.length > 0 && (
          <p className="mt-3 text-sm text-civic-ink">
            <span className="font-medium">{grants.length}</span> grants shown &middot;{' '}
            <span className="font-medium">{formatUSD(totalUSD)}</span> total
          </p>
        )}
      </header>

      <form
        className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4"
        aria-label="Filter Grants"
      >
        <FilterSelect
          label="Status"
          name="status"
          value={status}
          options={STATUS_FILTERS}
          basePath="/grants"
        />
        <FilterSelect
          label="Donor"
          name="donor"
          value={donor}
          options={DONOR_FILTERS}
          basePath="/grants"
        />
        <FilterSelect
          label="Sector"
          name="sector"
          value={sector}
          options={SECTOR_FILTERS}
          basePath="/grants"
        />
        <div>
          <label htmlFor="q" className="sr-only">
            Search within Grants
          </label>
          <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-paper px-3 py-2">
            <Filter className="h-4 w-4 text-civic-stone" aria-hidden="true" />
            <input
              id="q"
              type="search"
              name="q"
              defaultValue={searchParams.q ?? ''}
              placeholder="Search donor, purpose, sector"
              className="flex-1 bg-transparent text-sm focus:outline-none"
            />
          </div>
        </div>
      </form>

      {grants.length === 0 ? (
        <div className="rounded-lg border border-dashed border-civic-border p-12 text-center text-civic-stone">
          No grants match the current filters.
        </div>
      ) : (
        <ul className="grid gap-4">
          {grants.map((g) => (
            <li
              key={g.id}
              className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2 text-xs text-civic-stone">
                    <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium uppercase tracking-wide text-civic-ink">
                      {g.status}
                    </span>
                    <span>{g.grant_type}</span>
                    <span>&middot; sector: {g.sector.replace('_', ' ')}</span>
                    {g.announcement_date && (
                      <span>&middot; announced {formatDate(g.announcement_date)}</span>
                    )}
                  </div>
                  <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                    {g.donor}
                  </h2>
                  <p className="mt-1 line-clamp-3 text-sm text-civic-stone">
                    {g.purpose}
                  </p>
                </div>
                <div className="text-right">
                  <p className="font-mono text-base font-semibold text-civic-forest">
                    {formatUSD(g.amount_usd)}
                  </p>
                  {g.amount_kes && (
                    <p className="font-mono text-xs text-civic-stone">
                      {formatKES(g.amount_kes)}
                    </p>
                  )}
                  {g.disbursement_date && (
                    <p className="mt-1 text-xs text-civic-stone">
                      disbursed {formatDate(g.disbursement_date)}
                    </p>
                  )}
                </div>
              </div>
              {g.source_url && (
                <div className="mt-3 border-t border-civic-border pt-3">
                  <Link
                    href={g.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-xs font-medium text-civic-leaf hover:underline"
                  >
                    <ExternalLink className="h-3 w-3" aria-hidden="true" />
                    View source
                  </Link>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      <p className="mt-8 text-xs text-civic-stone">
        Source data is mirrored from <code>infrastructure/postgres/seed/003_loans_grants.sql</code>{' '}
        and verified against World Bank, EU Commission, Global Fund and AfDB publications.
        The Go BFF exposes <code>GET /api/v1/grants</code> and{' '}
        <code>GET /api/v1/grants/{'{id}'}</code>; the database read path will be wired in issue #19.
      </p>
    </div>
  );
}
