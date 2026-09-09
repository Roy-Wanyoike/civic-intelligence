import Link from 'next/link';
import { ExternalLink, Filter, Landmark } from 'lucide-react';
import { governmentLoans } from '@/lib/financial-data';
import { formatDate } from '@/lib/utils';
import { FilterSelect } from '@/components/filter-select';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Government Loans',
  description:
    'Track every sovereign loan the Government of Kenya has applied for or received since September 2022 — evidence-backed, with source URLs.',
};

const STATUS_FILTERS = [
  'all',
  'applied',
  'approved',
  'disbursed',
  'repaid',
  'defaulted',
] as const;
const LENDER_FILTERS = [
  'all',
  'IMF',
  'World Bank (IBRD)',
  'World Bank (IBRD + IDA)',
  'Trade and Development Bank (TDB)',
  'European Investment Bank (via TDB)',
  'China Exim Bank',
  'African Development Bank (AfDB)',
  'International Capital Markets',
] as const;
const SECTOR_FILTERS = [
  'all',
  'fiscal',
  'climate',
  'infrastructure',
  'finance',
  'water',
  'debt_refinancing',
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

export default function LoansPage({
  searchParams,
}: {
  searchParams: { lender?: string; sector?: string; status?: string; q?: string };
}) {
  let loans = governmentLoans.slice();
  const status = searchParams.status ?? 'all';
  const lender = searchParams.lender ?? 'all';
  const sector = searchParams.sector ?? 'all';
  const q = searchParams.q?.toLowerCase().trim();
  if (status !== 'all') loans = loans.filter((l) => l.status === status);
  if (lender !== 'all') loans = loans.filter((l) => l.lender === lender);
  if (sector !== 'all') loans = loans.filter((l) => l.sector === sector);
  if (q) {
    loans = loans.filter(
      (l) =>
        l.lender.toLowerCase().includes(q) ||
        l.purpose.toLowerCase().includes(q) ||
        l.sector.toLowerCase().includes(q),
    );
  }

  // Sort newest approval first.
  loans.sort((a, b) => {
    const ax = a.approval_date ? Date.parse(a.approval_date) : 0;
    const bx = b.approval_date ? Date.parse(b.approval_date) : 0;
    return bx - ax;
  });

  const totalUSD = loans.reduce((sum, l) => sum + (l.amount_usd ?? 0), 0);

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Landmark className="h-4 w-4" aria-hidden="true" />
          <span>Transparency Tracker</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Government Loans since September 2022
        </h1>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          Every sovereign loan the Government of Kenya has applied for or received
          since the Ruto administration took office on 13 September 2022. Each
          entry links to a credible public source &mdash; no fabricated amounts.
        </p>
        {loans.length > 0 && (
          <p className="mt-3 text-sm text-civic-ink">
            <span className="font-medium">{loans.length}</span> loans shown &middot;{' '}
            <span className="font-medium">{formatUSD(totalUSD)}</span> total
          </p>
        )}
      </header>

      <form
        className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4"
        aria-label="Filter Loans"
      >
        <FilterSelect
          label="Status"
          name="status"
          value={status}
          options={STATUS_FILTERS}
          basePath="/loans"
        />
        <FilterSelect
          label="Lender"
          name="lender"
          value={lender}
          options={LENDER_FILTERS}
          basePath="/loans"
        />
        <FilterSelect
          label="Sector"
          name="sector"
          value={sector}
          options={SECTOR_FILTERS}
          basePath="/loans"
        />
        <div>
          <label htmlFor="q" className="sr-only">
            Search within Loans
          </label>
          <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-paper px-3 py-2">
            <Filter className="h-4 w-4 text-civic-stone" aria-hidden="true" />
            <input
              id="q"
              type="search"
              name="q"
              defaultValue={searchParams.q ?? ''}
              placeholder="Search lender, purpose, sector"
              className="flex-1 bg-transparent text-sm focus:outline-none"
            />
          </div>
        </div>
      </form>

      {loans.length === 0 ? (
        <div className="rounded-lg border border-dashed border-civic-border p-12 text-center text-civic-stone">
          No loans match the current filters.
        </div>
      ) : (
        <ul className="grid gap-4">
          {loans.map((l) => (
            <li
              key={l.id}
              className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2 text-xs text-civic-stone">
                    <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium uppercase tracking-wide text-civic-ink">
                      {l.status}
                    </span>
                    <span>{l.loan_type}</span>
                    <span>&middot; sector: {l.sector.replace('_', ' ')}</span>
                    {l.approval_date && (
                      <span>&middot; approved {formatDate(l.approval_date)}</span>
                    )}
                  </div>
                  <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                    {l.lender}
                  </h2>
                  <p className="mt-1 line-clamp-3 text-sm text-civic-stone">
                    {l.purpose}
                  </p>
                </div>
                <div className="text-right">
                  <p className="font-mono text-base font-semibold text-civic-forest">
                    {formatUSD(l.amount_usd)}
                  </p>
                  {l.amount_kes && (
                    <p className="font-mono text-xs text-civic-stone">
                      {formatKES(l.amount_kes)}
                    </p>
                  )}
                  {l.interest_rate != null && (
                    <p className="mt-1 text-xs text-civic-stone">
                      {l.interest_rate}% interest
                    </p>
                  )}
                  {l.repayment_period_years && (
                    <p className="text-xs text-civic-stone">
                      {l.repayment_period_years}-year repayment
                    </p>
                  )}
                </div>
              </div>
              {l.source_url && (
                <div className="mt-3 border-t border-civic-border pt-3">
                  <Link
                    href={l.source_url}
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
        and verified against IMF, World Bank, AfDB, Reuters, EU Commission, Global Fund
        and AidData (China Exim) publications. The Go BFF exposes{' '}
        <code>GET /api/v1/loans</code> and <code>GET /api/v1/loans/{'{id}'}</code>; the
        database read path will be wired in issue #19.
      </p>
    </div>
  );
}
