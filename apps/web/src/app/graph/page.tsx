import type { Metadata } from 'next';
import { Network } from 'lucide-react';
import { getGraphSummary, type GraphSummary } from '@/lib/graph-api';
import { RealityBadge } from '@/components/reality-labels';
import { GraphExplorer } from './graph-view';

export const metadata: Metadata = {
  title: 'Civic Knowledge Graph',
  description:
    'Trace how Bills, Acts, Institutions, People, Constitution Articles, and Government borrowing are connected. The platform\'s signature visual relationship explorer.',
};

// KnowledgeGraphPage is a server component that fetches the graph summary
// (total node + edge counts broken down by type) at request time and passes
// it to the interactive GraphExplorer client component. The explorer
// loads the actual node + edge payloads on demand as the user searches,
// selects, and traverses — keeping the initial server-render payload small
// while still surfacing every relationship in the seed data.
export default async function KnowledgeGraphPage() {
  let summary: GraphSummary | null = null;
  let loadError: string | null = null;
  try {
    summary = await getGraphSummary();
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load graph summary';
  }

  return (
    <div className="mx-auto max-w-[100rem] px-4 py-8 sm:px-6 lg:px-8">
      <header className="mb-6">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Network className="h-4 w-4" aria-hidden="true" />
          <span>Kenya · Civic Knowledge Graph</span>
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-3">
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">
            Civic Knowledge Graph
          </h1>
          <RealityBadge kind="FACT" />
        </div>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          Trace how Bills, Acts, Institutions, People, Constitution Articles,
          Administrations, and Government borrowing are connected. This is
          the platform&apos;s signature differentiator — no civic intelligence
          platform in Africa offers a visual graph explorer. Every node
          links back to the authoritative Kenya Law, CBK, or Treasury source.
        </p>
      </header>

      {loadError ? (
        <div
          role="alert"
          className="rounded-lg border border-rose-200 bg-rose-50 p-6 text-sm text-rose-900"
        >
          <h2 className="font-serif text-base font-semibold">
            We couldn&apos;t load the graph summary right now.
          </h2>
          <p className="mt-2">
            The page fetches the graph summary from{' '}
            <code className="rounded bg-rose-100 px-1 py-0.5 text-xs">
              /api/v1/graph
            </code>
            . The service is unreachable or returned an error. Please try
            again in a moment — the explorer below will still render with an
            empty graph so the UI remains explorable.
          </p>
          <p className="mt-2 text-xs text-rose-700">{loadError}</p>
        </div>
      ) : null}

      {summary ? (
        <div
          className="mb-6 grid grid-cols-2 gap-3 rounded-lg border border-civic-border bg-civic-paper p-4 sm:grid-cols-4 lg:grid-cols-6"
          aria-label="Graph summary"
        >
          <SummaryStat label="Total nodes" value={summary.total_nodes} />
          <SummaryStat label="Total edges" value={summary.total_edges} />
          <SummaryStat label="Bills" value={summary.node_types.bill ?? 0} />
          <SummaryStat label="Acts" value={summary.node_types.act ?? 0} />
          <SummaryStat
            label="Articles"
            value={summary.node_types.constitution_article ?? 0}
          />
          <SummaryStat
            label="Loans"
            value={summary.node_types.borrowing_agreement ?? 0}
          />
        </div>
      ) : null}

      <GraphExplorer summary={summary} />
    </div>
  );
}

function SummaryStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="text-center">
      <p className="font-serif text-2xl font-semibold text-civic-forest">
        {value}
      </p>
      <p className="text-[11px] uppercase tracking-wide text-civic-stone">
        {label}
      </p>
    </div>
  );
}
