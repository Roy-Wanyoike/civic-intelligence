import Link from 'next/link';
import { Landmark, TrendingUp, ExternalLink } from 'lucide-react';
import type { Metadata } from 'next';
import { getDebtDashboard, getDebtTimeline, type DebtTrendPoint } from '@/lib/debt-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';
import { DebtTrendChart } from './debt-trend-chart';

export const metadata: Metadata = {
  title: 'Kenya Public Debt',
  description: 'Explore Kenya\'s public debt and borrowing history. Every figure is sourced from CBK or Treasury. The platform does not rank governments.',
};

function formatKES(amount?: number): string {
  if (amount == null) return '—';
  return new Intl.NumberFormat('en-KE', {
    style: 'currency',
    currency: 'KES',
    notation: 'compact',
    maximumFractionDigits: 2,
  }).format(amount);
}

function formatDate(d?: string): string {
  if (!d) return '—';
  return new Date(d).toLocaleDateString('en-KE', { year: 'numeric', month: 'short', day: 'numeric' });
}

export default async function DebtDashboardPage() {
  let dashboard: Awaited<ReturnType<typeof getDebtDashboard>> | null = null;
  let timeline: DebtTrendPoint[] = [];
  let loadError: string | null = null;
  try {
    const [d, t] = await Promise.all([getDebtDashboard(), getDebtTimeline()]);
    dashboard = d;
    timeline = t.points ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load';
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Landmark className="h-4 w-4" aria-hidden="true" />
          <span>Kenya · Public Debt & Borrowing</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Kenya Public Debt
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Explore Kenya&apos;s public debt and borrowing history. Every figure
          is sourced from the Central Bank of Kenya or the National Treasury.
          The platform <strong>does not</strong> rank governments, calculate
          debt performance scores, or attribute sovereign borrowing personally
          to a president.
        </p>
      </header>

      <div className="mb-8">
        <RealityDisclaimer kind="FACT" />
      </div>

      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}

      {dashboard && (
        <>
          <section className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="text-xs font-semibold uppercase text-civic-stone">Total Public Debt</h2>
              <p className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                {formatKES(dashboard.total_debt_stock)}
              </p>
              <p className="mt-1 text-xs text-stone-600">As of {formatDate(dashboard.as_of)}</p>
            </div>
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="text-xs font-semibold uppercase text-civic-stone">Domestic Debt</h2>
              <p className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                {formatKES(dashboard.domestic_debt)}
              </p>
            </div>
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="text-xs font-semibold uppercase text-civic-stone">External Debt</h2>
              <p className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                {formatKES(dashboard.external_debt)}
              </p>
            </div>
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="text-xs font-semibold uppercase text-civic-stone">Debt to GDP</h2>
              <p className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                {dashboard.debt_to_gdp ? `${dashboard.debt_to_gdp.toFixed(1)}%` : '—'}
              </p>
            </div>
          </section>

          <section className="mb-8 rounded-lg border border-stone-200 bg-white p-5">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5 text-civic-forest" aria-hidden="true" />
                <h2 className="font-serif text-lg font-semibold text-civic-forest">
                  Public Debt Stock Over Time
                </h2>
                <RealityBadge kind="FACT" />
              </div>
              <a
                href={dashboard.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
              >
                Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
              </a>
            </div>
            <p className="mt-1 text-xs text-stone-600">
              Metric: outstanding public debt stock. Unit: KES. Source: CBK Monthly Economic Indicators.
              Each point is an immutable observation.
            </p>
            <DebtTrendChart points={timeline} />
            <p className="mt-4 text-xs text-stone-600">
              <strong>Definition:</strong> Debt stock is the outstanding
              public debt recorded at the specified observation date. Changes
              in debt stock can reflect exchange-rate movements, valuation
              changes, repayments, refinancing, arrears, adjustments, and
              disbursement timing — <strong>NOT</strong> just new borrowing.
            </p>
          </section>

          <section className="mb-8 rounded-lg border border-stone-200 bg-white p-5">
            <h2 className="font-serif text-lg font-semibold text-civic-forest">
              Government Debt Summaries
            </h2>
            <p className="mt-1 text-xs text-stone-600">
              Each summary covers a presidential administration. The platform
              does NOT rank governments or attribute borrowing personally to
              a president.
            </p>
            <ul className="mt-3 space-y-2">
              <li className="rounded-lg border border-stone-200 bg-stone-50 p-3">
                <Link
                  href="/governments/admin-uhuru-kenyatta"
                  className="text-sm font-semibold text-civic-forest hover:underline"
                >
                  Uhuru Kenyatta Administration (2013-2022) →
                </Link>
                <p className="mt-1 text-xs text-stone-600">
                  Debt grew from KSh 1.96T to KSh 7.71T during this period.
                </p>
              </li>
              <li className="rounded-lg border border-stone-200 bg-stone-50 p-3">
                <Link
                  href="/governments/admin-william-ruto"
                  className="text-sm font-semibold text-civic-forest hover:underline"
                >
                  William Ruto Administration (2022-Present) →
                </Link>
                <p className="mt-1 text-xs text-stone-600">
                  Debt grew from KSh 7.71T to KSh 10.59T during this period.
                </p>
              </li>
            </ul>
          </section>

          <section className="rounded-lg border border-amber-200 bg-amber-50 p-5">
            <div className="flex items-center gap-2">
              <h2 className="font-serif text-lg font-semibold text-amber-900">
                Attribution Rule
              </h2>
              <RealityBadge kind="LIMITATION" />
            </div>
            <p className="mt-2 text-sm text-amber-900">
              The platform never says &ldquo;President X borrowed KSh X.&rdquo;
              Instead it says: <em>&ldquo;The Government of Kenya recorded
              KSh X in borrowing during this period.&rdquo;</em>
            </p>
            <p className="mt-2 text-xs text-amber-800">
              The legal borrower is the Republic of Kenya, not an individual.
              Borrowing is attributed by <strong>CONTRACTED_DURING</strong>,
              not by when repayment occurs. The platform distinguishes
              CONTRACTED_DURING, DISBURSED_DURING, REPAID_DURING,
              OUTSTANDING_DURING, and REFINANCED_DURING to prevent misleading
              historical attribution.
            </p>
          </section>
        </>
      )}
    </div>
  );
}
