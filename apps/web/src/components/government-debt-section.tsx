// GovernmentDebtSection renders the PUBLIC DEBT & BORROWING section on a
// government detail page. Spec section 31.

import { getGovernmentDebtSummary } from '@/lib/debt-api';
import { RealityBadge } from '@/components/reality-labels';
import { ExternalLink } from 'lucide-react';

interface GovernmentDebtSectionProps {
  administrationId: string;
}

function formatKES(amount?: number): string {
  if (amount == null) return '—';
  return new Intl.NumberFormat('en-KE', {
    style: 'currency',
    currency: 'KES',
    notation: 'compact',
    maximumFractionDigits: 2,
  }).format(amount);
}

export async function GovernmentDebtSection({ administrationId }: GovernmentDebtSectionProps) {
  let summary: Awaited<ReturnType<typeof getGovernmentDebtSummary>> | null = null;
  let loadError: string | null = null;
  try {
    summary = await getGovernmentDebtSummary(administrationId);
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load';
  }

  if (loadError) {
    return (
      <section className="mt-6 rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Public Debt & Borrowing
        </h2>
        <p className="mt-2 text-xs text-stone-600">{loadError}</p>
      </section>
    );
  }

  if (!summary) return null;

  return (
    <section className="mt-6 rounded-lg border border-stone-200 bg-white p-5">
      <div className="flex items-center justify-between gap-2">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Public Debt & Borrowing
        </h2>
        <RealityBadge kind="FACT" size="md" />
      </div>
      <p className="mt-1 text-xs text-stone-600">
        Period: {summary.period} · Currency: {summary.currency}
      </p>

      <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-3">
          <dt className="text-xs font-semibold text-civic-stone">Debt at start</dt>
          <dd className="mt-1 font-serif text-lg font-semibold text-civic-forest">
            {formatKES(summary.debt_at_start)}
          </dd>
        </div>
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-3">
          <dt className="text-xs font-semibold text-civic-stone">Debt at end</dt>
          <dd className="mt-1 font-serif text-lg font-semibold text-civic-forest">
            {formatKES(summary.debt_at_end)}
          </dd>
        </div>
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-3">
          <dt className="text-xs font-semibold text-civic-stone">New borrowing</dt>
          <dd className="mt-1 font-serif text-lg font-semibold text-civic-forest">
            {formatKES(summary.new_borrowing)}
          </dd>
        </div>
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-3">
          <dt className="text-xs font-semibold text-civic-stone">Debt service</dt>
          <dd className="mt-1 font-serif text-lg font-semibold text-civic-forest">
            {formatKES(summary.debt_service)}
          </dd>
        </div>
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-3">
          <dt className="text-xs font-semibold text-civic-stone">Domestic debt</dt>
          <dd className="mt-1 font-serif text-lg font-semibold text-civic-forest">
            {formatKES(summary.domestic_debt)}
          </dd>
        </div>
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-3">
          <dt className="text-xs font-semibold text-civic-stone">External debt</dt>
          <dd className="mt-1 font-serif text-lg font-semibold text-civic-forest">
            {formatKES(summary.external_debt)}
          </dd>
        </div>
      </dl>

      {summary.source_urls && summary.source_urls.length > 0 && (
        <div className="mt-4">
          <h3 className="text-xs font-semibold uppercase text-civic-stone">Sources</h3>
          <ul className="mt-1 space-y-1">
            {summary.source_urls.map((url) => (
              <li key={url}>
                <a
                  href={url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
                >
                  {url.replace(/^https?:\/\//, '').slice(0, 60)}{url.length > 60 ? '…' : ''}
                  <ExternalLink className="h-3 w-3" aria-hidden="true" />
                </a>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3">
        <p className="text-xs text-amber-900 whitespace-pre-line">
          {summary.disclaimer}
        </p>
      </div>

      <p className="mt-3 text-xs text-stone-600">
        <strong>Methodology:</strong> Debt at start/end are observed public
        debt snapshots. New borrowing is the sum of CONTRACTED_DURING
        borrowing agreements in the period. Debt service is principal + interest.
        Changes in debt stock can also reflect exchange-rate movements,
        valuation changes, repayments, refinancing, arrears, and adjustments.
      </p>
    </section>
  );
}
