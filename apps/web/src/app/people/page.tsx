import Link from 'next/link';
import type { Metadata } from 'next';
import { Users, ArrowRight } from 'lucide-react';
import { listPeople } from '@/lib/people-api';
import { RealityBadge } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'People — MPs, Senators & Committee Members',
  description:
    'Members of Parliament, Senators, and committee members — connected to the Bills they sponsor and the proceedings they participate in. The platform does not rank politicians — we connect them to verifiable civic activity.',
};

export default async function PeoplePage() {
  let people: Awaited<ReturnType<typeof listPeople>>['items'] = [];
  let loadError: string | null = null;
  try {
    const resp = await listPeople();
    people = resp.items ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load';
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Users className="h-4 w-4" aria-hidden="true" />
          <span>People · Kenya</span>
          <RealityBadge kind="FACT" size="md" />
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">People</h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          MPs, Senators, and committee members connected to the Bills they
          sponsor, the speeches they make, and the committees they serve on.
          We do not rank politicians — we connect them to verifiable civic
          activity. Each MP has a factual scorecard that aggregates their
          parliamentary record from official sources.
        </p>
      </header>

      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}

      {people.length === 0 && !loadError ? (
        <p className="rounded-lg border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
          No people loaded yet.
        </p>
      ) : (
        <ul className="grid gap-3 md:grid-cols-2">
          {people.map((p) => (
            <li
              key={p.person_id}
              className="rounded-lg border border-civic-border bg-civic-paper p-4 transition hover:border-civic-leaf"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="font-serif text-base font-semibold text-civic-ink">
                    <Link
                      href={`/people/${p.person_id}/scorecard`}
                      className="hover:underline"
                    >
                      {p.name}
                    </Link>
                  </h2>
                  <p className="mt-1 text-xs text-civic-stone">{p.role}</p>
                  <p className="mt-2 text-xs text-civic-stone">
                    {p.constituency} · {p.party}
                  </p>
                </div>
                <RealityBadge kind="FACT" />
              </div>
              <Link
                href={`/people/${p.person_id}/scorecard`}
                className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-civic-forest hover:underline"
              >
                View scorecard <ArrowRight className="h-3 w-3" aria-hidden="true" />
              </Link>
            </li>
          ))}
        </ul>
      )}

      <p className="mt-8 rounded-lg border border-emerald-300 bg-emerald-50 p-4 text-xs text-emerald-900">
        <strong>Factual records only.</strong> The platform does not rank MPs,
        score their performance, or imply political approval. Each scorecard
        presents raw counts (bills sponsored, questions asked, votes recorded,
        statements made) and a single attendance rate — every metric carries a
        source URL so a citizen can verify each datum against the official
        parliamentary record.
      </p>
    </div>
  );
}
