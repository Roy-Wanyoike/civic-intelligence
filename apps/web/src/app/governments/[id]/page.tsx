import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, Landmark, Calendar, ExternalLink } from 'lucide-react';
import type { Metadata } from 'next';
import { getAdministration } from '@/lib/government-api';
import { RealityBadge } from '@/components/reality-labels';

function formatDate(d?: string): string {
  if (!d) return 'Present';
  return new Date(d).toLocaleDateString('en-KE', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  let title = 'Administration';
  try {
    const resp = await getAdministration(id);
    title = resp.administration.name;
  } catch { /* ignore */ }
  return { title: `${title} — Government` };
}

export default async function AdministrationDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let data: Awaited<ReturnType<typeof getAdministration>> | null = null;
  try {
    data = await getAdministration(id);
  } catch {
    notFound();
  }

  const admin = data.administration;
  const president = data.president;
  const terms = data.terms;

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <Link
        href="/governments"
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> All governments
      </Link>

      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Landmark className="h-4 w-4" aria-hidden="true" />
          <span>{admin.country_code} · Administration</span>
        </div>
        <div className="mt-2 flex items-start justify-between gap-3">
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">
            {admin.name}
          </h1>
          <RealityBadge kind="FACT" size="md" />
        </div>
        <p className="mt-2 text-sm text-civic-stone">
          President: <span className="font-semibold text-civic-forest">{president.display_name}</span>
        </p>
      </header>

      <section className="mb-6 rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">Overview</h2>
        <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-xs font-semibold text-civic-stone">President</dt>
            <dd className="text-stone-800">{president.full_name}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Government system</dt>
            <dd className="text-stone-800">{admin.government_system}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Start date</dt>
            <dd className="text-stone-800">{formatDate(admin.start_date)}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">End date</dt>
            <dd className="text-stone-800">{formatDate(admin.end_date)}</dd>
          </div>
        </dl>
        {admin.source_url && (
          <a
            href={admin.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
          >
            Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        )}
      </section>

      <section className="rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Presidential Terms
        </h2>
        <p className="mt-1 text-xs text-stone-600">
          A president may have multiple terms. The data model supports N terms;
          it does not assume every administration has exactly two.
        </p>
        <ol className="mt-4 space-y-3">
          {terms.map((t) => (
            <li
              key={t.id}
              className="rounded-lg border border-stone-200 bg-stone-50 p-4"
            >
              <div className="flex items-start justify-between gap-2">
                <div>
                  <h3 className="text-sm font-semibold text-civic-forest">
                    Term {t.term_number}
                  </h3>
                  <p className="mt-1 text-xs text-stone-700">
                    <Calendar className="mr-1 inline h-3 w-3" aria-hidden="true" />
                    {formatDate(t.start_date)} → {formatDate(t.end_date)}
                  </p>
                </div>
                <span
                  className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${
                    t.status === 'CURRENT'
                      ? 'bg-emerald-100 text-emerald-800'
                      : t.status === 'INTERRUPTED'
                        ? 'bg-amber-100 text-amber-800'
                        : 'bg-stone-200 text-stone-700'
                  }`}
                >
                  {t.status}
                </span>
              </div>
              <Link
                href={`/governments/${admin.id}/terms/${t.id}`}
                className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-civic-forest hover:underline"
              >
                View term detail →
              </Link>
            </li>
          ))}
        </ol>
      </section>

      <section className="mt-6 rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          What this page does NOT do
        </h2>
        <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-stone-700">
          <li>It does not rank administrations.</li>
          <li>It does not score presidential performance.</li>
          <li>It does not infer political attribution from dates alone.</li>
          <li>It does not modify or reinterpret constitutional text.</li>
        </ul>
      </section>
    </div>
  );
}
