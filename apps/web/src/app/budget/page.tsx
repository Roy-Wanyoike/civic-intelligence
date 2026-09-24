import Link from 'next/link';
import { Landmark, ExternalLink, PieChart } from 'lucide-react';
import type { Metadata } from 'next';
import { getBudget } from '@/lib/budget-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';
import { BudgetTreemap } from './budget-treemap';

export const metadata: Metadata = {
  title: 'Kenya National Budget — FY 2026/27',
  description:
    "Explore Kenya's national budget allocations across all ministries. Every figure is sourced from the National Treasury. The platform does not rank ministries.",
};

/**
 * formatKES — format a number in KES billions as a compact currency
 * string (e.g. "KES 1,500B" or "KES 3.91T" for totals). Used for the
 * headline numbers + the per-row table amounts.
 */
function formatKES(amountBillions: number): string {
  if (amountBillions >= 1000) {
    return `KES ${(amountBillions / 1000).toLocaleString('en-KE', {
      maximumFractionDigits: 2,
    })}T`;
  }
  return `KES ${amountBillions.toLocaleString('en-KE', { maximumFractionDigits: 1 })}B`;
}

export default async function BudgetPage() {
  let budget: Awaited<ReturnType<typeof getBudget>> | null = null;
  let loadError: string | null = null;
  try {
    budget = await getBudget({ country: 'KE', year: '2026' });
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load';
  }

  // Compute the recurrent / development split for the headline KPIs.
  // Done client-side (rather than asking the API for it) so the page
  // can render the split even when the API hasn't been extended with
  // summary fields. Both splits are surfaced raw — the platform does
  // NOT derive a "priority score" from them (rule:
  // NO_POLITICAL_PERFORMANCE_SCORE).
  const recurrentTotal = budget?.items
    .filter((i) => i.category === 'recurrent')
    .reduce((sum, i) => sum + i.amount_kes_billions, 0) ?? 0;
  const developmentTotal = budget?.items
    .filter((i) => i.category === 'development')
    .reduce((sum, i) => sum + i.amount_kes_billions, 0) ?? 0;
  const recurrentShare = budget && budget.total > 0 ? (recurrentTotal / budget.total) * 100 : 0;
  const developmentShare = budget && budget.total > 0 ? (developmentTotal / budget.total) * 100 : 0;

  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Landmark className="h-4 w-4" aria-hidden="true" />
          <span>Kenya · National Budget</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Kenya National Budget — FY 2026/27
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Explore Kenya&apos;s national budget allocations across all
          ministries, state departments, and constitutional commissions.
          Every figure is sourced from the National Treasury&apos;s
          published Budget Statement. The platform <strong>does not</strong>{' '}
          rank ministries, compute priority scores, or attribute budget
          outcomes to a president.
        </p>
      </header>

      <div className="mb-8">
        <RealityDisclaimer kind="FACT" />
      </div>

      {loadError && (
        <div className="mb-8 rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          Failed to load budget data: {loadError}
        </div>
      )}

      {budget && (
        <>
          {/* Headline KPIs: total budget + recurrent / development split */}
          <section className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="text-xs font-semibold uppercase text-civic-stone">Total Budget</h2>
              <p className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                {formatKES(budget.total)}
              </p>
              <p className="mt-1 text-xs text-stone-600">
                {budget.items.length} ministry allocations · FY {budget.year}
              </p>
            </div>
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <div className="flex items-center gap-2">
                <span
                  className="inline-block h-3 w-4 rounded-sm"
                  style={{ backgroundColor: '#0c3b2e' }}
                  aria-hidden="true"
                />
                <h2 className="text-xs font-semibold uppercase text-civic-stone">Recurrent</h2>
                <RealityBadge kind="FACT" />
              </div>
              <p className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                {formatKES(recurrentTotal)}
              </p>
              <p className="mt-1 text-xs text-stone-600">
                {recurrentShare.toFixed(1)}% of total — salaries, operations, debt servicing
              </p>
            </div>
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <div className="flex items-center gap-2">
                <span
                  className="inline-block h-3 w-4 rounded-sm"
                  style={{ backgroundColor: '#d4a017' }}
                  aria-hidden="true"
                />
                <h2 className="text-xs font-semibold uppercase text-civic-stone">Development</h2>
                <RealityBadge kind="FACT" />
              </div>
              <p className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                {formatKES(developmentTotal)}
              </p>
              <p className="mt-1 text-xs text-stone-600">
                {developmentShare.toFixed(1)}% of total — capital projects, infrastructure
              </p>
            </div>
          </section>

          {/* Treemap */}
          <section className="mb-8 rounded-lg border border-stone-200 bg-white p-5">
            <div className="mb-4 flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <PieChart className="h-5 w-5 text-civic-forest" aria-hidden="true" />
                <h2 className="font-serif text-lg font-semibold text-civic-forest">
                  Budget Allocations by Ministry
                </h2>
                <RealityBadge kind="FACT" />
              </div>
              <a
                href={budget.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
              >
                Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
              </a>
            </div>
            <p className="mb-4 text-xs text-stone-600">
              Cells sized by allocation amount. Hover any cell for the ministry name,
              allocation, and percentage share of the total budget. On mobile, the
              treemap renders as an accessible table.
            </p>
            <BudgetTreemap items={budget.items} />
            <p className="mt-4 text-xs text-stone-600">
              <strong>Definition:</strong> Each allocation is the ministry&apos;s total
              voted provision for FY {budget.year}, tagged with its dominant category —
              recurrent for service-delivery ministries (Education, Health, Interior,
              Defence), development for capital-heavy ministries (Roads, Energy, Water).
              Every ministry has both recurrent and development components in the
              real budget; this view surfaces the dominant share for at-a-glance
              reading. When the verified Programme-Based Budget ingestion path ships,
              the treemap will split each ministry into its Vote R (recurrent) and
              Vote D (development) components.
            </p>
          </section>

          {/* Attribution rule */}
          <section className="rounded-lg border border-amber-200 bg-amber-50 p-5">
            <div className="flex items-center gap-2">
              <h2 className="font-serif text-lg font-semibold text-amber-900">
                Attribution Rule
              </h2>
              <RealityBadge kind="LIMITATION" />
            </div>
            <p className="mt-2 text-sm text-amber-900">
              The platform never says &ldquo;President X&apos;s budget&rdquo;.
              Instead it says: <em>&ldquo;The Government of Kenya allocated
              KES X to this ministry for FY {budget.year}.&rdquo;</em>
            </p>
            <p className="mt-2 text-xs text-amber-800">
              The legal spender is the Republic of Kenya, not an individual.
              Allocations are approved by Parliament through the Appropriation
              Bill and the County Allocation of Revenue Bill — the platform
              attributes spending to constitutional votes, not to a person.
            </p>
          </section>

          {/* Cross-links */}
          <section className="mt-8 flex flex-wrap gap-3 text-sm">
            <Link
              href="/debt"
              className="inline-flex items-center gap-1.5 rounded-md border border-civic-border px-3 py-2 text-civic-ink hover:border-civic-leaf hover:text-civic-leaf"
            >
              <Landmark className="h-4 w-4" aria-hidden="true" />
              Public Debt Dashboard
            </Link>
            <Link
              href="/governments"
              className="inline-flex items-center gap-1.5 rounded-md border border-civic-border px-3 py-2 text-civic-ink hover:border-civic-leaf hover:text-civic-leaf"
            >
              <Landmark className="h-4 w-4" aria-hidden="true" />
              Government Administrations
            </Link>
          </section>
        </>
      )}
    </div>
  );
}
