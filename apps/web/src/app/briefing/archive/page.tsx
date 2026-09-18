// Civic Daily Brief — Archive.
//
// Lists previously generated briefs (newest-first). Each entry links back to
// the per-brief detail view at /briefing?id={id} (which the page.tsx reads
// via the query string and uses to fetch a specific brief instead of
// today's). The archive is intentionally a list of summaries — full briefs
// are fetched on demand when a reader taps one.
//
// Server component — fetches /api/v1/brief/archive from the Go BFF with
// revalidate=300.

import Link from 'next/link';
import { Calendar, FileText } from 'lucide-react';
import { formatDate } from '@/lib/utils';
import type { Metadata } from 'next';
import type { BriefArchiveEntry } from '@/lib/types';

export const metadata: Metadata = {
  title: 'Civic Daily Brief — Archive',
  description: 'Previously generated Civic Daily Briefs.',
};

async function fetchArchive(): Promise<{ items: BriefArchiveEntry[]; total: number } | null> {
  const base = process.env.API_SERVICE_URL ?? 'http://localhost:9000';
  try {
    const resp = await fetch(`${base}/api/v1/brief/archive`, {
      next: { revalidate: 300 },
      headers: { Accept: 'application/json' },
    });
    if (!resp.ok) return null;
    return (await resp.json()) as { items: BriefArchiveEntry[]; total: number };
  } catch {
    return null;
  }
}

export default async function BriefArchivePage() {
  const data = await fetchArchive();

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2">
          <FileText className="h-6 w-6 text-civic-leaf" aria-hidden="true" />
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">Brief Archive</h1>
        </div>
        <p className="mt-2 text-sm text-civic-stone">
          Previously generated Civic Daily Briefs. Each entry is a snapshot of that day&rsquo;s verified civic developments.
        </p>
      </header>

      {!data || data.items.length === 0 ? (
        <div className="rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
          <p>No past briefs in the archive yet.</p>
          <p className="mt-2">
            <Link href="/briefing" className="text-civic-leaf hover:underline">
              View today&rsquo;s brief →
            </Link>
          </p>
        </div>
      ) : (
        <ul className="space-y-3">
          {data.items.map((entry) => (
            <li key={entry.id}>
              <Link
                href={`/briefing?id=${encodeURIComponent(entry.id)}`}
                className="block rounded-xl border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf hover:shadow-sm transition"
              >
                <div className="flex items-center gap-2 text-xs text-civic-stone">
                  <Calendar className="h-3 w-3" aria-hidden="true" />
                  <time dateTime={entry.date}>{formatDate(entry.date)}</time>
                  <span aria-hidden="true">·</span>
                  <span>{entry.evidence_count} citation{entry.evidence_count === 1 ? '' : 's'}</span>
                </div>
                <h2 className="mt-1.5 font-serif text-base font-semibold text-civic-ink">
                  {entry.headline}
                </h2>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <p className="mt-8 text-xs text-civic-stone">
        <Link href="/briefing" className="text-civic-leaf hover:underline">
          ← Back to today&rsquo;s brief
        </Link>
      </p>
    </div>
  );
}
