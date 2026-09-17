import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { listScenarios } from '@/lib/scenarios-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';
import { CompareForm } from './compare-form';

export default async function ComparisonPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let scenarios: Awaited<ReturnType<typeof listScenarios>>['scenarios'] = [];
  let loadError: string | null = null;
  try {
    const resp = await listScenarios();
    scenarios = resp.scenarios ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load scenarios';
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
            Compare Scenarios
          </h1>
          <RealityBadge kind="HYPOTHETICAL" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          Compare scenarios side-by-side along multiple dimensions. The
          platform describes differences only; it does NOT rank scenarios or
          recommend choices.
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
      <CompareForm currentId={id} scenarios={scenarios} />
    </div>
  );
}
