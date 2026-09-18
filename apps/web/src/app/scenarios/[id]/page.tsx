import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, Sparkles } from 'lucide-react';
import type { Metadata } from 'next';
import { getScenario, type Scenario } from '@/lib/scenarios-api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';
import { ScenarioActions } from './scenario-actions';

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

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  let title = 'Scenario';
  try {
    const resp = await getScenario(id);
    title = resp.scenario.name;
  } catch {
    /* ignore */
  }
  return {
    title: `${title} — Scenario`,
    description: `Hypothetical civic scenario: ${title}.`,
  };
}

const TABS = [
  { slug: '', label: 'Overview' },
  { slug: 'assumptions', label: 'Assumptions' },
  { slug: 'evidence', label: 'Evidence' },
  { slug: 'results', label: 'Results' },
  { slug: 'comparison', label: 'Comparison' },
  { slug: 'timeline', label: 'Timeline' },
  { slug: 'methodology', label: 'Methodology' },
];

export default async function ScenarioDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let scenario: Scenario | null = null;
  try {
    const resp = await getScenario(id);
    scenario = resp.scenario;
  } catch {
    notFound();
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <Link
        href="/scenarios"
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to scenarios
      </Link>

      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Sparkles className="h-4 w-4" aria-hidden="true" />
          <span>{TYPE_LABELS[scenario.type] ?? scenario.type}</span>
          <span aria-hidden="true">·</span>
          <span>{scenario.jurisdiction}</span>
          <span aria-hidden="true">·</span>
          <span>{STATUS_LABELS[scenario.status] ?? scenario.status}</span>
        </div>
        <div className="mt-2 flex items-start justify-between gap-3">
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">
            {scenario.name}
          </h1>
          <RealityBadge kind="HYPOTHETICAL" size="md" />
        </div>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          {scenario.description}
        </p>
      </header>

      <div className="mb-6">
        <RealityDisclaimer kind="HYPOTHETICAL" />
      </div>

      <nav
        aria-label="Scenario sections"
        className="mb-6 flex flex-wrap gap-2 border-b border-stone-200 pb-2"
      >
        {TABS.map((t) => (
          <Link
            key={t.slug || 'overview'}
            href={t.slug ? `/scenarios/${id}/${t.slug}` : `/scenarios/${id}`}
            className="rounded-lg px-3 py-1.5 text-sm font-semibold text-stone-700 transition hover:bg-stone-100"
          >
            {t.label}
          </Link>
        ))}
      </nav>

      <section className="space-y-6">
        <div className="rounded-lg border border-stone-200 bg-white p-5">
          <h2 className="font-serif text-lg font-semibold text-civic-forest">
            What are we exploring?
          </h2>
          <p className="mt-1 text-sm text-stone-700">
            {scenario.description}
          </p>
        </div>

        <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-5">
          <div className="flex items-center gap-2">
            <h2 className="font-serif text-lg font-semibold text-emerald-900">
              Baseline
            </h2>
            <RealityBadge kind="FACT" />
          </div>
          <p className="mt-1 text-sm text-emerald-900">
            {scenario.baseline?.description ?? 'No baseline declared.'}
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="rounded-lg border border-amber-200 bg-amber-50 p-5">
            <div className="flex items-center gap-2">
              <h3 className="font-serif text-base font-semibold text-amber-900">
                Assumptions
              </h3>
              <RealityBadge kind="ASSUMPTION" />
            </div>
            <p className="mt-1 text-sm text-amber-900">
              {scenario.assumptions?.length ?? 0} explicit assumption(s)
              declared.
            </p>
            <Link
              href={`/scenarios/${id}/assumptions`}
              className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-amber-900 hover:underline"
            >
              Inspect assumptions →
            </Link>
          </div>

          <div className="rounded-lg border border-sky-200 bg-sky-50 p-5">
            <div className="flex items-center gap-2">
              <h3 className="font-serif text-base font-semibold text-sky-900">
                Evidence
              </h3>
              <RealityBadge kind="EVIDENCE" />
            </div>
            <p className="mt-1 text-sm text-sky-900">
              {scenario.source_evidence?.length ?? 0} source reference(s) plus
              baseline evidence.
            </p>
            <Link
              href={`/scenarios/${id}/evidence`}
              className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-sky-900 hover:underline"
            >
              Inspect evidence →
            </Link>
          </div>
        </div>

        <div className="rounded-lg border border-stone-200 bg-white p-5">
          <h3 className="font-serif text-base font-semibold text-civic-forest">
            Model
          </h3>
          <dl className="mt-2 grid gap-2 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Model ID</dt>
              <dd className="text-stone-800">{scenario.model_id}</dd>
            </div>
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Model version</dt>
              <dd className="text-stone-800">{scenario.model_version}</dd>
            </div>
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Time horizon</dt>
              <dd className="text-stone-800">
                {scenario.time_horizon?.start?.slice(0, 10)} →{' '}
                {scenario.time_horizon?.end?.slice(0, 10)}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Scenario version</dt>
              <dd className="text-stone-800">v{scenario.scenario_version}</dd>
            </div>
          </dl>
          <Link
            href={`/scenarios/${id}/methodology`}
            className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-civic-forest hover:underline"
          >
            Inspect methodology →
          </Link>
        </div>

        <ScenarioActions scenarioId={id} status={scenario.status} />
      </section>
    </div>
  );
}
