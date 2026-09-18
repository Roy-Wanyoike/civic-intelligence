import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, Calendar } from 'lucide-react';
import type { Metadata } from 'next';
import { getJSON_ as getJSON } from '@/lib/api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Post-Assent Events',
  description: 'Every event tracked after a Bill became an Act: commencement, regulations, court challenges, amendments, repeal.',
};

interface EventItem {
  id: string;
  act_id: string;
  event_type: string;
  event_date: string;
  title: string;
  description?: string;
  source_url?: string;
}

interface EventsResponse {
  act_id: string;
  events: EventItem[];
  count: number;
  disclaimer: string;
}

const TYPE_COLORS: Record<string, string> = {
  COMMENCEMENT: 'border-emerald-200 bg-emerald-50 text-emerald-900',
  REGULATION: 'border-sky-200 bg-sky-50 text-sky-900',
  IMPLEMENTATION: 'border-stone-200 bg-stone-50 text-stone-800',
  COURT_CHALLENGE: 'border-amber-200 bg-amber-50 text-amber-900',
  JUDICIAL_DECISION: 'border-violet-200 bg-violet-50 text-violet-900',
  AMENDMENT: 'border-rose-200 bg-rose-50 text-rose-900',
  REPEAL: 'border-stone-400 bg-stone-100 text-stone-800',
};

function formatDate(d?: string): string {
  if (!d) return '—';
  return new Date(d).toLocaleDateString('en-KE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

export default async function EventsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let events: EventsResponse | null = null;
  try {
    events = await getJSON<EventsResponse>(`/api/v1/acts/${id}/events`);
  } catch {
    notFound();
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <Link
        href={`/acts/${id}`}
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to act
      </Link>
      <header className="mt-4 mb-6">
        <h1 className="font-serif text-2xl font-semibold text-civic-forest">
          Post-Assent Events
        </h1>
        <p className="mt-1 text-sm text-civic-stone">
          Every event tracked after this Bill became an Act. Events sourced
          from authoritative material (Kenya Law, Kenya Gazette).
        </p>
      </header>
      <div className="mb-6">
        <RealityDisclaimer kind="FACT" />
      </div>
      {events.events.length === 0 ? (
        <div className="rounded-lg border border-stone-200 bg-stone-50 p-8 text-center text-sm text-stone-700">
          No post-assent events tracked yet. This does NOT mean nothing
          happened — only that the platform has not yet ingested the
          authoritative record.
        </div>
      ) : (
        <ol className="space-y-3">
          {events.events.map((e) => {
            const color = TYPE_COLORS[e.event_type] || 'border-stone-200 bg-white';
            return (
              <li
                key={e.id}
                className={`rounded-lg border p-4 ${color}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <h3 className="font-serif text-base font-semibold">
                      {e.title}
                    </h3>
                    <p className="mt-1 text-xs">
                      <Calendar className="mr-1 inline h-3 w-3" aria-hidden="true" />
                      {formatDate(e.event_date)}
                    </p>
                  </div>
                  <RealityBadge kind="FACT" />
                </div>
                {e.description && (
                  <p className="mt-2 text-sm">{e.description}</p>
                )}
                {e.source_url && (
                  <a
                    href={e.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-2 inline-block text-xs underline"
                  >
                    Source
                  </a>
                )}
                <div className="mt-2">
                  <span className="rounded-full bg-white/60 px-2 py-0.5 text-[10px] font-semibold uppercase">
                    {e.event_type}
                  </span>
                </div>
              </li>
            );
          })}
        </ol>
      )}
      <p className="mt-6 text-xs text-stone-600">{events.disclaimer}</p>
    </div>
  );
}
