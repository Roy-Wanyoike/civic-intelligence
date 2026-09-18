import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { listAssumptions } from '@/lib/scenarios-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

const TYPE_LABELS: Record<string, string> = {
  OBSERVED_INPUT: 'Observed input',
  USER_DEFINED: 'User-defined',
  MODEL_ASSUMPTION: 'Model assumption',
  HISTORICAL_REFERENCE: 'Historical reference',
  ESTIMATE: 'Estimate',
  UNKNOWN: 'Unknown',
};

const CONFIDENCE_COLORS: Record<string, string> = {
  HIGH: 'text-emerald-700',
  MEDIUM: 'text-amber-700',
  LOW: 'text-rose-700',
  UNKNOWN: 'text-stone-500',
};

export default async function AssumptionsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let assumptions: Awaited<ReturnType<typeof listAssumptions>>['assumptions'] = [];
  let loadError: string | null = null;
  try {
    const resp = await listAssumptions(id);
    assumptions = resp.assumptions ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load assumptions';
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
            Assumptions
          </h1>
          <RealityBadge kind="ASSUMPTION" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          Every assumption is explicit. Externally sourced assumptions must
          carry traceable evidence; UNKNOWN assumptions must NOT carry a value.
        </p>
      </header>
      <div className="mb-6">
        <RealityDisclaimer kind="HYPOTHETICAL" />
      </div>
      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}
      <ul className="space-y-3">
        {assumptions.map((a) => (
          <li
            key={a.id}
            className="rounded-lg border border-amber-200 bg-amber-50 p-4"
          >
            <div className="flex items-start justify-between gap-3">
              <p className="text-sm font-semibold text-amber-900">
                {a.statement}
              </p>
              <RealityBadge kind="ASSUMPTION" />
            </div>
            <dl className="mt-3 grid gap-2 text-xs sm:grid-cols-3">
              <div>
                <dt className="font-semibold text-civic-stone">Type</dt>
                <dd className="text-amber-900">
                  {TYPE_LABELS[a.assumption_type] ?? a.assumption_type}
                </dd>
              </div>
              <div>
                <dt className="font-semibold text-civic-stone">Confidence</dt>
                <dd className={CONFIDENCE_COLORS[a.confidence] ?? 'text-stone-700'}>
                  {a.confidence}
                </dd>
              </div>
              <div>
                <dt className="font-semibold text-civic-stone">Value</dt>
                <dd className="text-amber-900">
                  {a.value !== undefined && a.value !== null
                    ? `${a.value}${a.unit ? ' ' + a.unit : ''}`
                    : '— (UNKNOWN)'}
                </dd>
              </div>
            </dl>
            {a.basis && (
              <p className="mt-2 text-xs text-amber-800">
                <span className="font-semibold">Basis:</span> {a.basis}
              </p>
            )}
            {a.source_evidence && a.source_evidence.length > 0 && (
              <p className="mt-2 text-xs text-amber-800">
                <span className="font-semibold">Evidence:</span>{' '}
                {a.source_evidence.length} reference(s)
              </p>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
