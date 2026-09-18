import Link from 'next/link';
import type { Metadata } from 'next';
import { MapPin, Users, ArrowRight, Search } from 'lucide-react';
import { listConstituencies } from '@/lib/people-api';
import { RealityBadge } from '@/components/reality-labels';
import { ConstituencySearch } from './search';

export const metadata: Metadata = {
  title: 'Constituencies — Kenya',
  description:
    'Browse Kenyan constituencies. Each constituency has a dashboard with its MP, bills affecting the area, budget, local projects, public participation opportunities and recent civic events.',
};

function formatPopulation(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}K`;
  return String(n);
}

export default async function ConstituenciesPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string; country?: string }>;
}) {
  const { q, country } = await searchParams;
  let items: Awaited<ReturnType<typeof listConstituencies>>['items'] = [];
  let total = 0;
  let loadError: string | null = null;
  try {
    const resp = await listConstituencies({
      country: country ?? 'KE',
      q: q?.trim() || undefined,
    });
    items = resp.items ?? [];
    total = resp.total;
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load';
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <MapPin className="h-4 w-4" aria-hidden="true" />
          <span>Constituencies · Kenya</span>
          <RealityBadge kind="FACT" size="md" />
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Constituencies
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Browse the 10 sample constituencies across Kenya. Each constituency
          has a dashboard with its MP (and link to the MP&apos;s scorecard),
          Bills affecting the area, budget allocated, local projects gazetted,
          public participation opportunities, recent civic events and
          government context.
        </p>
      </header>

      {/* Search + filter bar */}
      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <ConstituencySearch initialQuery={q ?? ''} />
        <p className="text-xs text-civic-stone">
          Showing <span className="font-semibold text-civic-ink">{items.length}</span>{' '}
          of {total} constituencies
        </p>
      </div>

      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}

      {items.length === 0 && !loadError ? (
        <div className="rounded-lg border border-civic-border bg-civic-mist p-8 text-center">
          <p className="text-sm text-civic-stone">
            No constituencies match &quot;{q}&quot;. Try a different search term.
          </p>
        </div>
      ) : (
        <ul className="grid gap-3 md:grid-cols-2">
          {items.map((c) => (
            <li
              key={c.id}
              className="rounded-lg border border-civic-border bg-civic-paper p-4 transition hover:border-civic-leaf"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="font-serif text-lg font-semibold text-civic-forest">
                    <Link href={`/constituencies/${c.id}`} className="hover:underline">
                      {c.name}
                    </Link>
                  </h2>
                  <p className="mt-0.5 text-xs text-civic-stone">
                    {c.county} County · {c.region}
                  </p>
                </div>
                <RealityBadge kind="FACT" />
              </div>
              <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
                <div>
                  <dt className="font-semibold text-civic-stone">MP</dt>
                  <dd className="text-civic-ink">{c.mp_name}</dd>
                </div>
                <div>
                  <dt className="font-semibold text-civic-stone">Party</dt>
                  <dd className="text-civic-ink">{c.mp_party}</dd>
                </div>
                <div>
                  <dt className="font-semibold text-civic-stone">Population</dt>
                  <dd className="inline-flex items-center gap-1 text-civic-ink">
                    <Users className="h-3 w-3" aria-hidden="true" />
                    {formatPopulation(c.population)}
                  </dd>
                </div>
              </dl>
              <Link
                href={`/constituencies/${c.id}`}
                className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-civic-forest hover:underline"
              >
                Open dashboard <ArrowRight className="h-3 w-3" aria-hidden="true" />
              </Link>
            </li>
          ))}
        </ul>
      )}

      <p className="mt-8 rounded-lg border border-emerald-300 bg-emerald-50 p-4 text-xs text-emerald-900">
        <strong>Factual civic records only.</strong> The dashboard does not
        rank constituencies, score development performance, or imply political
        approval. Every datum — budget line, project, gazette notice — carries
        a source URL so a citizen can verify each against the official record.
      </p>
    </div>
  );
}
