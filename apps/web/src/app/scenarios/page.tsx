import Link from 'next/link';
import { Sparkles, Plus, ArrowRight } from 'lucide-react';
import type { Metadata } from 'next';
import { listScenarios, type Scenario } from '@/lib/scenarios-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Scenarios — What If?',
  description:
    'Explore hypothetical civic scenarios. Every scenario is HYPOTHETICAL — never confused with observed civic fact.',
};

const TYPE_LABELS: Record<Scenario['type'], string> = {
  POLICY: 'Policy',
  LEGISLATIVE: 'Legislative',
  IMPLEMENTATION: 'Implementation',
  COMPARATIVE: 'Comparative',
  HISTORICAL_COUNTERFACTUAL: 'Historical Counterfactual',
  INSTITUTIONAL: 'Institutional',
  ECONOMIC_SOCIAL: 'Economic / Social Impact',
  INFRASTRUCTURE: 'Infrastructure',
};

const STATUS_LABELS: Record<Scenario['status'], string> = {
  DRAFT: 'Draft',
  CONFIGURED: 'Configured',
  VALIDATING: 'Validating',
  READY: 'Ready',
  RUNNING: 'Running',
  COMPLETED: 'Completed',
  REVIEW_REQUIRED: 'Review Required',
  ARCHIVED: 'Archived',
  FAILED: 'Failed',
};

export default async function ScenariosPage() {
  let scenarios: Scenario[] = [];
  let disclaimer = '';
  let loadError: string | null = null;
  try {
    const resp = await listScenarios();
    scenarios = resp.scenarios ?? [];
    disclaimer = resp.disclaimer ?? '';
  } catch (err) {
    loadError = err instanceof Error ? err.message : 'Failed to load scenarios';
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Sparkles className="h-4 w-4" aria-hidden="true" />
          <span>Scenario Explorer</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          What If?
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Explore possible outcomes of civic events and policy scenarios. Every
          scenario is <strong>hypothetical</strong>; results are{' '}
          <strong>simulated</strong>. Reality is observed; scenarios are
          constructed; assumptions are explicit; results are simulated;
          evidence remains traceable.
        </p>
      </header>

      <div className="mb-8">
        <RealityDisclaimer kind="HYPOTHETICAL" />
      </div>

      <div className="mb-6 flex items-center justify-between">
        <h2 className="font-serif text-xl font-semibold text-civic-forest">
          Available Scenarios
        </h2>
        <Link
          href="/scenarios/new"
          className="inline-flex items-center gap-2 rounded-lg bg-civic-forest px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-forest/90"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          New Scenario
        </Link>
      </div>

      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          Failed to load scenarios: {loadError}
        </div>
      )}

      {disclaimer && !loadError && scenarios.length === 0 ? (
        <div className="rounded-lg border border-stone-300 bg-stone-50 p-8 text-center text-sm text-stone-700">
          No scenarios yet. Create one to explore a &ldquo;What If?&rdquo;
          question.
        </div>
      ) : null}

      <ul className="grid gap-4 sm:grid-cols-2">
        {scenarios.map((s) => (
          <li
            key={s.id}
            className="rounded-lg border border-stone-200 bg-white p-5 transition hover:border-civic-forest/40 hover:shadow-sm"
          >
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
                  <span>{TYPE_LABELS[s.type] ?? s.type}</span>
                  <span aria-hidden="true">·</span>
                  <span>{s.jurisdiction}</span>
                </div>
                <h3 className="mt-1 font-serif text-lg font-semibold text-civic-forest">
                  <Link
                    href={`/scenarios/${s.id}`}
                    className="hover:underline"
                  >
                    {s.name}
                  </Link>
                </h3>
              </div>
              <RealityBadge kind="HYPOTHETICAL" />
            </div>
            <p className="mt-2 line-clamp-3 text-sm text-stone-700">
              {s.description}
            </p>
            <dl className="mt-4 grid grid-cols-2 gap-2 text-xs">
              <div>
                <dt className="font-semibold text-civic-stone">Status</dt>
                <dd className="text-stone-800">
                  {STATUS_LABELS[s.status] ?? s.status}
                </dd>
              </div>
              <div>
                <dt className="font-semibold text-civic-stone">Assumptions</dt>
                <dd className="text-stone-800">{s.assumptions?.length ?? 0}</dd>
              </div>
            </dl>
            <div className="mt-4 flex items-center gap-2">
              <Link
                href={`/scenarios/${s.id}`}
                className="inline-flex items-center gap-1 text-sm font-semibold text-civic-forest hover:underline"
              >
                Explore <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
              </Link>
            </div>
          </li>
        ))}
      </ul>

      <section className="mt-12 rounded-lg border border-stone-200 bg-stone-50 p-6">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Visual Language
        </h2>
        <p className="mt-1 text-sm text-stone-700">
          Every page in the scenario explorer uses six persistent labels so
          you can always tell what you are looking at:
        </p>
        <ul className="mt-3 flex flex-wrap gap-3">
          <li><RealityBadge kind="FACT" /></li>
          <li><RealityBadge kind="EVIDENCE" /></li>
          <li><RealityBadge kind="ASSUMPTION" /></li>
          <li><RealityBadge kind="SIMULATION" /></li>
          <li><RealityBadge kind="UNKNOWN" /></li>
          <li><RealityBadge kind="LIMITATION" /></li>
        </ul>
      </section>
    </div>
  );
}
