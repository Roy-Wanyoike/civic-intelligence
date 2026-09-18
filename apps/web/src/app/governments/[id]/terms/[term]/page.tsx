import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, Calendar } from 'lucide-react';
import type { Metadata } from 'next';
import { getAdministration } from '@/lib/government-api';
import { RealityBadge } from '@/components/reality-labels';

function formatDate(d?: string): string {
  if (!d) return 'Present';
  return new Date(d).toLocaleDateString('en-KE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string; term: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  let title = 'Presidential Term';
  try {
    const resp = await getAdministration(id);
    title = `${resp.administration.name} — Term`;
  } catch { /* ignore */ }
  return { title: `${title} — Government` };
}

export default async function PresidentialTermPage({
  params,
}: {
  params: Promise<{ id: string; term: string }>;
}) {
  const { id, term } = await params;
  let data: Awaited<ReturnType<typeof getAdministration>> | null = null;
  try {
    data = await getAdministration(id);
  } catch {
    notFound();
  }
  const termRecord = data.terms.find((t) => t.id === term);
  if (!termRecord) {
    notFound();
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <Link
        href={`/governments/${id}`}
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to administration
      </Link>
      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Calendar className="h-4 w-4" aria-hidden="true" />
          <span>Presidential Term</span>
        </div>
        <div className="mt-2 flex items-start justify-between gap-3">
          <h1 className="font-serif text-2xl font-semibold text-civic-forest">
            {data.administration.name} · Term {termRecord.term_number}
          </h1>
          <RealityBadge kind="FACT" size="md" />
        </div>
      </header>
      <section className="rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">Period</h2>
        <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Start</dt>
            <dd className="text-stone-800">{formatDate(termRecord.start_date)}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">End</dt>
            <dd className="text-stone-800">{formatDate(termRecord.end_date)}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Status</dt>
            <dd className="text-stone-800">{termRecord.status}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Term number</dt>
            <dd className="text-stone-800">{termRecord.term_number}</dd>
          </div>
        </dl>
        {termRecord.source_url && (
          <p className="mt-3 text-xs text-stone-600">
            Source: <a className="text-sky-700 hover:underline" href={termRecord.source_url} target="_blank" rel="noopener noreferrer">{termRecord.source_url}</a>
          </p>
        )}
      </section>
    </div>
  );
}
