import Link from 'next/link';
import { ArrowLeft, ExternalLink } from 'lucide-react';
import { listEvidence } from '@/lib/scenarios-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export default async function EvidencePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let source: Awaited<ReturnType<typeof listEvidence>> = {
    source_evidence: [],
    baseline_evidence: [],
    assumption_evidence_count: 0,
  };
  let loadError: string | null = null;
  try {
    source = await listEvidence(id);
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load evidence';
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
            Evidence
          </h1>
          <RealityBadge kind="EVIDENCE" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          Every externally sourced assumption must be traceable to authoritative
          material. The platform rejects assumptions that claim to be observed
          without evidence.
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
      <section className="mb-6 rounded-lg border border-sky-200 bg-sky-50 p-5">
        <h2 className="font-serif text-lg font-semibold text-sky-900">
          Baseline Evidence
        </h2>
        <p className="mt-1 text-sm text-sky-900">
          Evidence supporting the OBSERVED baseline.
        </p>
        <ul className="mt-3 space-y-2">
          {source.baseline_evidence && source.baseline_evidence.length > 0 ? (
            source.baseline_evidence.map((e) => (
              <li
                key={e.id}
                className="rounded-lg border border-sky-200 bg-white p-3 text-sm"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-semibold text-sky-900">{e.kind}</span>
                  <RealityBadge kind="FACT" />
                </div>
                <p className="mt-1 text-xs text-stone-700">{e.id}</p>
                {e.source_url && (
                  <a
                    href={e.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-1 inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
                  >
                    Open source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                  </a>
                )}
              </li>
            ))
          ) : (
            <li className="rounded-lg border border-stone-200 bg-white p-3 text-sm text-stone-700">
              No baseline evidence references.
            </li>
          )}
        </ul>
      </section>
      <section className="rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Source Evidence
        </h2>
        <p className="mt-1 text-sm text-stone-700">
          Evidence attached directly to the scenario.
        </p>
        <p className="mt-2 text-xs text-stone-600">
          Total assumption evidence references: {source.assumption_evidence_count ?? 0}
        </p>
        <ul className="mt-3 space-y-2">
          {source.source_evidence && source.source_evidence.length > 0 ? (
            source.source_evidence.map((e) => (
              <li
                key={e.id}
                className="rounded-lg border border-stone-200 bg-stone-50 p-3 text-sm"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-semibold text-stone-800">{e.kind}</span>
                  <RealityBadge kind="EVIDENCE" />
                </div>
                <p className="mt-1 text-xs text-stone-700">{e.id}</p>
                {e.source_url && (
                  <a
                    href={e.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-1 inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
                  >
                    Open source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                  </a>
                )}
              </li>
            ))
          ) : (
            <li className="rounded-lg border border-stone-200 bg-stone-50 p-3 text-sm text-stone-700">
              No source evidence references.
            </li>
          )}
        </ul>
      </section>
    </div>
  );
}
