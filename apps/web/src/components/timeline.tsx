import { formatDate, confidenceClass } from '@/lib/utils';
import type { BillEvent } from '@/lib/types';
import { ExternalLink } from 'lucide-react';

export function TimelineView({ events, limit }: { events: BillEvent[]; limit?: number }) {
  if (events.length === 0) {
    return (
      <p className="mt-4 text-sm text-civic-stone">
        No verified events have been recorded yet. Information could not be
        verified from available authoritative sources.
      </p>
    );
  }

  const ordered = [...events].sort((a, b) => {
    if (!a.date) return 1;
    if (!b.date) return -1;
    return +new Date(a.date) - +new Date(b.date);
  }).reverse();
  const shown = limit ? ordered.slice(0, limit) : ordered;

  return (
    <ol className="mt-4 space-y-4" aria-label="Bill timeline">
      {shown.map((ev) => (
        <li
          key={ev.id}
          className="relative border-l-2 border-civic-border pl-4 pb-1"
        >
          <div className="absolute -left-[5px] top-1 h-2 w-2 rounded-full bg-civic-leaf" aria-hidden="true" />
          <div className="flex flex-wrap items-baseline gap-2">
            <time className="text-sm font-medium text-civic-ink" dateTime={ev.date ?? undefined}>
              {formatDate(ev.date)}
              {ev.date_is_approximate && (
                <span className="ml-1 text-xs text-civic-stone">(approximate)</span>
              )}
            </time>
            <span className={`rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${confidenceClass(ev.confidence)}`}>
              {ev.confidence}
            </span>
            {ev.house && <span className="text-xs text-civic-stone">{ev.house}</span>}
          </div>
          <p className="mt-1 text-sm text-civic-ink">{ev.description}</p>
          {ev.note && (
            <p className="mt-1 text-xs italic text-civic-stone">{ev.note}</p>
          )}
          {ev.source_url && (
            <a
              href={ev.source_url}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-1 inline-flex items-center gap-0.5 text-xs text-civic-leaf hover:underline"
            >
              Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
            </a>
          )}
        </li>
      ))}
    </ol>
  );
}
