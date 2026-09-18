import Link from 'next/link';
import { Landmark, ArrowRight } from 'lucide-react';
import type { Metadata } from 'next';
import { listAdministrations, type AdministrationListItem } from '@/lib/government-api';
import { RealityBadge } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Governments — Kenya Administrations',
  description: 'Explore Kenyan presidential administrations and their terms. Every record is evidence-backed.',
};

function formatDate(d?: string): string {
  if (!d) return 'Present';
  return new Date(d).toLocaleDateString('en-KE', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

function formatRange(start: string, end?: string): string {
  const s = new Date(start).getFullYear();
  const e = end ? new Date(end).getFullYear() : 'Present';
  return `${s} – ${e}`;
}

export default async function GovernmentsPage() {
  let admins: AdministrationListItem[] = [];
  let loadError: string | null = null;
  try {
    const resp = await listAdministrations();
    admins = resp.administrations ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load';
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Landmark className="h-4 w-4" aria-hidden="true" />
          <span>Kenya · Government History</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Governments of Kenya
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Explore every presidential administration since independence. Every
          record is sourced from authoritative material — the platform does
          not infer dates or political attribution.
        </p>
      </header>

      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}

      <ol className="relative space-y-6 border-l-2 border-stone-200 pl-6">
        {admins.map((a) => (
          <li key={a.id} className="relative">
            <span
              className="absolute -left-[1.65rem] flex h-4 w-4 items-center justify-center rounded-full border-2 border-white bg-civic-forest"
              aria-hidden="true"
            />
            <div className="rounded-lg border border-stone-200 bg-white p-5">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
                    <span>{formatRange(a.start_date, a.end_date)}</span>
                    {a.is_current && (
                      <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-800">
                        Current
                      </span>
                    )}
                  </div>
                  <h2 className="mt-1 font-serif text-xl font-semibold text-civic-forest">
                    <Link href={`/governments/${a.id}`} className="hover:underline">
                      {a.name}
                    </Link>
                  </h2>
                  <p className="mt-1 text-sm text-stone-700">
                    President: <span className="font-semibold">{a.president_name}</span>
                  </p>
                </div>
                <RealityBadge kind="FACT" />
              </div>
              <dl className="mt-3 grid gap-2 text-xs sm:grid-cols-2">
                <div>
                  <dt className="font-semibold text-civic-stone">Start</dt>
                  <dd className="text-stone-800">{formatDate(a.start_date)}</dd>
                </div>
                <div>
                  <dt className="font-semibold text-civic-stone">End</dt>
                  <dd className="text-stone-800">{formatDate(a.end_date)}</dd>
                </div>
              </dl>
              <Link
                href={`/governments/${a.id}`}
                className="mt-3 inline-flex items-center gap-1 text-sm font-semibold text-civic-forest hover:underline"
              >
                Explore administration <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
              </Link>
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}
