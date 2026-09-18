import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { getTimeline } from '@/lib/scenarios-api';
import { RealityDisclaimer, TimelineLabel } from '@/components/reality-labels';

export default async function TimelinePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let timeline: Awaited<ReturnType<typeof getTimeline>>['timeline'] = [];
  let loadError: string | null = null;
  try {
    const resp = await getTimeline(id);
    timeline = resp.timeline ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load timeline';
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
        <h1 className="font-serif text-2xl font-semibold text-civic-forest">
          Timeline
        </h1>
        <p className="mt-1 text-sm text-civic-stone">
          Every event on the scenario timeline is labelled OBSERVED, ASSUMED,
          MODELED, or UNKNOWN so you can always tell where reality ends and
          the hypothetical begins.
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
      <ol className="relative space-y-6 border-l-2 border-stone-200 pl-6">
        {timeline.map((ev, i) => (
          <li key={i} className="relative">
            <span
              className="absolute -left-[1.65rem] flex h-4 w-4 items-center justify-center rounded-full border-2 border-white bg-civic-forest"
              aria-hidden="true"
            />
            <div className="rounded-lg border border-stone-200 bg-white p-4">
              <div className="flex items-center justify-between gap-2">
                <h3 className="font-serif text-base font-semibold text-civic-forest">
                  {ev.title}
                </h3>
                <TimelineLabel kind={ev.label} />
              </div>
              <p className="mt-1 text-xs text-stone-600">
                {new Date(ev.timestamp).toLocaleString()}
              </p>
              {ev.description && (
                <p className="mt-2 text-sm text-stone-700">{ev.description}</p>
              )}
            </div>
          </li>
        ))}
        {timeline.length === 0 && !loadError && (
          <li className="rounded-lg border border-stone-200 bg-stone-50 p-4 text-sm text-stone-700">
            No timeline events recorded.
          </li>
        )}
      </ol>
    </div>
  );
}
