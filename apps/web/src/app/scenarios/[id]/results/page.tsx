import Link from 'next/link';
import type { Metadata } from 'next';
import { ArrowLeft } from 'lucide-react';
import { listResults } from '@/lib/scenarios-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Scenario Results',
  description:
    'The computed results of a policy simulation scenario — every result is tagged HYPOTHETICAL or MODELED so it is never confused with observed civic facts.',
  alternates: { canonical: '/scenarios/[id]/results' },
  robots: { index: false, follow: true },
};

function formatValue(v: unknown): string {
  if (v === null || v === undefined) return '—';
  if (typeof v === 'number') {
    if (Number.isInteger(v)) return v.toLocaleString();
    return v.toFixed(3);
  }
  if (typeof v === 'string') return v;
  return JSON.stringify(v);
}

export default async function ResultsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let results: Awaited<ReturnType<typeof listResults>>['results'] = [];
  let loadError: string | null = null;
  try {
    const resp = await listResults(id);
    results = resp.results ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load results';
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <Link
        href={`/scenarios/${id}`}
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to scenario
      </Link>
      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2">
          <h1 className="font-serif text-2xl font-semibold text-civic-forest">
            Results
          </h1>
          <RealityBadge kind="SIMULATION" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          Modeled outcomes. These are SIMULATED — they are NOT observed civic
          facts. Scenarios capable of uncertainty expose percentiles rather
          than false precision.
        </p>
      </header>
      <div className="mb-6">
        <RealityDisclaimer kind="SIMULATION" />
      </div>
      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}
      {results.length === 0 && !loadError && (
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-8 text-center text-sm text-stone-700">
          No published results yet. Run the scenario to produce results.
        </div>
      )}
      <ul className="space-y-4">
        {results.map((r) => {
          const s = r.summary;
          return (
            <li
              key={r.run_id}
              className="rounded-lg border border-violet-200 bg-violet-50 p-5"
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h3 className="font-serif text-base font-semibold text-violet-900">
                    {s?.headline_metric ? formatValue(s.headline_metric) : 'Modeled outcome'}
                  </h3>
                  <p className="mt-1 text-xs text-violet-900">
                    Run: <code>{r.run_id}</code>
                    {r.completed_at && (
                      <> · completed {new Date(r.completed_at).toLocaleString()}</>
                    )}
                    {r.duration && <> · {r.duration}</>}
                  </p>
                </div>
                <RealityBadge kind="SIMULATION" />
              </div>
              {s?.uncertainty && s.uncertainty.has_uncertainty && (
                <div className="mt-3 rounded-lg border border-violet-200 bg-white p-3">
                  <h4 className="text-xs font-semibold text-violet-900">
                    Uncertainty (Monte Carlo percentiles)
                  </h4>
                  <dl className="mt-2 grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
                    <div>
                      <dt className="text-stone-600">P10</dt>
                      <dd className="font-semibold text-stone-800">
                        {formatValue(s.uncertainty.p10)}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-stone-600">Median (P50)</dt>
                      <dd className="font-semibold text-stone-800">
                        {formatValue(s.uncertainty.median)}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-stone-600">P90</dt>
                      <dd className="font-semibold text-stone-800">
                        {formatValue(s.uncertainty.p90)}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-stone-600">Range</dt>
                      <dd className="font-semibold text-stone-800">
                        {formatValue(s.uncertainty.min)} – {formatValue(s.uncertainty.max)}
                      </dd>
                    </div>
                  </dl>
                  {s.uncertainty.notes && (
                    <p className="mt-2 text-xs text-stone-600">{s.uncertainty.notes}</p>
                  )}
                </div>
              )}
              {s?.limitations && s.limitations.length > 0 && (
                <div className="mt-3">
                  <h4 className="text-xs font-semibold text-rose-800">
                    <RealityBadge kind="LIMITATION" /> Limitations
                  </h4>
                  <ul className="mt-1 list-disc pl-5 text-xs text-stone-700">
                    {s.limitations.map((l, i) => (
                      <li key={i}>{l}</li>
                    ))}
                  </ul>
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
