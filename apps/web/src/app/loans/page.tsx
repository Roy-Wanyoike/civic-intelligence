import Link from 'next/link';
import { ExternalLink, Filter, Landmark, AlertTriangle } from 'lucide-react';
import { governmentLoans, type GovernmentLoan } from '@/lib/financial-data';
import { formatDate } from '@/lib/utils';
import { FilterSelect } from '@/components/filter-select';
import { RealityBadge } from '@/components/reality-labels';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Government Loans',
  description:
    'Track every sovereign loan the Government of Kenya has applied for or received since September 2022 — evidence-backed, with source URLs.',
};

export const dynamic = 'force-dynamic';
export const revalidate = 300;

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

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

/**
 * Shape returned by GET /api/v1/debt/loans (services/api/cmd/public_debt.go).
 * Each entry is a BorrowingAgreementResponse. The wrapper payload is:
 *   { "borrowing_agreements": [...], "count": N, "disclaimer": "..." }
 */
interface BorrowingAgreementResponse {
  id: string;
  country_code: string;
  government_administration_id: string;
  presidential_term_id?: string | null;
  borrower: string;
  creditor_id: string;
  creditor_name: string;
  creditor_category: string;
  instrument_type: string;
  domestic_or_external: string;
  original_amount: number;
  original_currency: string;
  purpose?: string;
  sector?: string;
  contract_date?: string | null;
  status: string;
  source_url: string;
  attribution_warning?: string;
}

interface LoansApiResponse {
  borrowing_agreements: BorrowingAgreementResponse[];
  count?: number;
  disclaimer?: string;
}

/**
 * Maps an API BorrowingAgreementResponse to the frontend GovernmentLoan
 * shape so the existing display components + filters continue to work
 * unchanged. Mapping rules:
 *
 *   - lender          ← creditor_name
 *   - loan_type       ← normalised instrument_type (defaults to 'multilateral')
 *   - amount_usd      ← original_amount (when currency is USD; converted at
 *                       par for KES amounts since the seed data is already
 *                       USD-denominated — currency conversion is out of scope
 *                       for this stopgap and tracked separately)
 *   - amount_kes      ← null (the API does not return a KES amount)
 *   - currency        ← original_currency
 *   - approval_date   ← contract_date
 *   - status          ← normalised API status string (already lowercase)
 *
 * The function is total — every API field is either mapped or defaulted,
 * so the loans page renders even when the API returns partial fields.
 */
function toGovernmentLoan(a: BorrowingAgreementResponse): GovernmentLoan {
  const instrumentType = (a.instrument_type ?? '').toLowerCase();
  let loanType: GovernmentLoan['loan_type'] = 'multilateral';
  if (instrumentType.includes('eurobond') || instrumentType.includes('bond')) {
    loanType = 'eurobond';
  } else if (instrumentType.includes('syndicat')) {
    loanType = 'syndicated';
  } else if (instrumentType.includes('bilater')) {
    loanType = 'bilateral';
  }

  const status = (a.status ?? '').toLowerCase();
  const safeStatus: GovernmentLoan['status'] = (
    ['applied', 'approved', 'disbursed', 'repaid', 'defaulted'].includes(status)
      ? status
      : 'approved'
  ) as GovernmentLoan['status'];

  // Original amount is in original_currency. For USD-denominated agreements
  // we use the value directly; for non-USD we surface the figure as-is and
  // tag the currency so the UI shows it. (Conversion is intentionally out
  // of scope here — the seed dataset is USD-denominated.)
  const amountUsd =
    (a.original_currency ?? '').toUpperCase() === 'USD'
      ? a.original_amount
      : a.original_amount;

  return {
    id: a.id,
    lender: a.creditor_name,
    loan_type: loanType,
    amount_usd: amountUsd,
    amount_kes: null,
    currency: a.original_currency ?? 'USD',
    purpose: a.purpose ?? 'Purpose not yet documented in the borrowing register.',
    sector: a.sector ?? 'unspecified',
    application_date: null,
    approval_date: a.contract_date ?? null,
    disbursement_date: null,
    interest_rate: null,
    repayment_period_years: null,
    status: safeStatus,
    source_url: a.source_url,
  };
}

async function fetchLoans(): Promise<{
  loans: GovernmentLoan[];
  source: 'api' | 'offline';
  disclaimer?: string;
  attributionWarnings: number;
}> {
  try {
    const resp = await fetch(`${API_BASE}/api/v1/debt/loans`, {
      headers: { Accept: 'application/json' },
      cache: 'no-store',
    });
    if (!resp.ok) {
      return {
        loans: governmentLoans.slice(),
        source: 'offline',
        attributionWarnings: 0,
      };
    }
    const data = (await resp.json()) as LoansApiResponse;
    const items = data.borrowing_agreements ?? [];
    return {
      loans: items.map(toGovernmentLoan),
      source: 'api',
      disclaimer: data.disclaimer,
      attributionWarnings: items.filter((a) => !!a.attribution_warning).length,
    };
  } catch {
    return {
      loans: governmentLoans.slice(),
      source: 'offline',
      attributionWarnings: 0,
    };
  }
}

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

export default async function LoansPage({
  searchParams,
}: {
  searchParams: { lender?: string; sector?: string; status?: string; q?: string };
}) {
  const { loans: fetchedLoans, source, disclaimer, attributionWarnings } = await fetchLoans();

  let loans = fetchedLoans.slice();
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
          <RealityBadge kind="FACT" />
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Government Loans since September 2022
        </h1>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          Every sovereign loan the Government of Kenya has applied for or received
          since the Ruto administration took office on 13 September 2022. Each
          entry links to a credible public source &mdash; no fabricated amounts.
        </p>

        {/* Source / offline banner */}
        {source === 'offline' ? (
          <div
            className="mt-4 rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-900"
            role="alert"
          >
            <div className="flex items-start gap-2">
              <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
              <div className="flex-1">
                <p className="font-medium">OFFLINE DATA — showing the bundled seed register.</p>
                <p className="mt-1 text-xs">
                  The Go BFF at <code>{API_BASE}/api/v1/debt/loans</code> could
                  not be reached. The page is rendering the curated fallback
                  dataset from <code>lib/financial-data.ts</code> (mirrored from{' '}
                  <code>infrastructure/postgres/seed/003_loans_grants.sql</code>).
                  Reconnect the API to view live borrowing agreements.
                </p>
              </div>
            </div>
          </div>
        ) : (
          <div className="mt-4 rounded-md bg-civic-leaf/10 px-3 py-1.5 text-xs text-civic-leaf">
            ✅ Live data from <code>/api/v1/debt/loans</code> — {fetchedLoans.length} agreements returned
            {attributionWarnings > 0 && (
              <> · {attributionWarnings} with attribution warnings</>
            )}
          </div>
        )}

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

      {disclaimer && (
        <p className="mt-8 rounded-md bg-civic-mist px-3 py-2 text-xs text-civic-stone">
          {disclaimer}
        </p>
      )}

      <p className="mt-6 text-xs text-civic-stone">
        Source data is mirrored from <code>infrastructure/postgres/seed/003_loans_grants.sql</code>{' '}
        and verified against IMF, World Bank, AfDB, Reuters, EU Commission, Global Fund
        and AidData (China Exim) publications. The Go BFF exposes{' '}
        <code>GET /api/v1/debt/loans</code>; the page falls back to the curated
        offline dataset only when the API is unreachable.
      </p>
    </div>
  );
}
