import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { getMethodology } from '@/lib/scenarios-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export default async function MethodologyPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let methodology: Awaited<ReturnType<typeof getMethodology>> | null = null;
  let loadError: string | null = null;
  try {
    methodology = await getMethodology(id);
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load methodology';
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
            Methodology
          </h1>
          <RealityBadge kind="HYPOTHETICAL" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          The methodology, model, and limitations behind this scenario. Every
          published result identifies the model and limitations used to
          produce it.
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
      {methodology && (
        <div className="space-y-6">
          <section className="rounded-lg border border-stone-200 bg-white p-5">
            <h2 className="font-serif text-lg font-semibold text-civic-forest">
              Model
            </h2>
            <dl className="mt-2 grid gap-2 text-sm sm:grid-cols-2">
              <div>
                <dt className="text-xs font-semibold text-civic-stone">Model ID</dt>
                <dd className="text-stone-800">{methodology.model_id}</dd>
              </div>
              <div>
                <dt className="text-xs font-semibold text-civic-stone">Model version</dt>
                <dd className="text-stone-800">{methodology.model_version}</dd>
              </div>
              {methodology.methodology.engine_name && (
                <div>
                  <dt className="text-xs font-semibold text-civic-stone">Engine</dt>
                  <dd className="text-stone-800">{methodology.methodology.engine_name}</dd>
                </div>
              )}
              {methodology.methodology.engine_version && (
                <div>
                  <dt className="text-xs font-semibold text-civic-stone">Engine version</dt>
                  <dd className="text-stone-800">{methodology.methodology.engine_version}</dd>
                </div>
              )}
            </dl>
          </section>
          {methodology.methodology.description && (
            <section className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="font-serif text-lg font-semibold text-civic-forest">
                Description
              </h2>
              <p className="mt-2 text-sm text-stone-700">
                {methodology.methodology.description}
              </p>
            </section>
          )}
          {methodology.methodology.inputs && methodology.methodology.inputs.length > 0 && (
            <section className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="font-serif text-lg font-semibold text-civic-forest">
                Inputs
              </h2>
              <ul className="mt-2 list-disc pl-5 text-sm text-stone-700">
                {methodology.methodology.inputs.map((i, idx) => (
                  <li key={idx}>{i}</li>
                ))}
              </ul>
            </section>
          )}
          {methodology.methodology.outputs && methodology.methodology.outputs.length > 0 && (
            <section className="rounded-lg border border-stone-200 bg-white p-5">
              <h2 className="font-serif text-lg font-semibold text-civic-forest">
                Outputs
              </h2>
              <ul className="mt-2 list-disc pl-5 text-sm text-stone-700">
                {methodology.methodology.outputs.map((o, idx) => (
                  <li key={idx}>{o}</li>
                ))}
              </ul>
            </section>
          )}
          <section className="rounded-lg border border-rose-200 bg-rose-50 p-5">
            <div className="flex items-center gap-2">
              <h2 className="font-serif text-lg font-semibold text-rose-900">
                Limitations
              </h2>
              <RealityBadge kind="LIMITATION" />
            </div>
            <ul className="mt-2 list-disc pl-5 text-sm text-rose-900">
              {methodology.limitations.map((l, i) => (
                <li key={i}>{l}</li>
              ))}
            </ul>
          </section>
        </div>
      )}
    </div>
  );
}
