import Link from 'next/link';
import { Search as SearchIcon, ExternalLink } from 'lucide-react';
import { mockBills, mockBriefing } from '@/lib/mock-data';
import { formatDate } from '@/lib/utils';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Search',
  description: 'Search Bills, Acts, regulations, and civic documents from across the Kenyan government.',
};

export default function SearchPage({
  searchParams,
}: {
  searchParams: { q?: string; kind?: string };
}) {
  const q = (searchParams.q ?? '').toLowerCase().trim();

  // Build a tiny in-memory search over mock Bills and briefing items.
  // Real implementation: GET /api/v1/search?q=... (Postgres FTS + pgvector).
  const results: Array<{ kind: string; id: string; title: string; snippet: string; url: string; published_at: string | null }> = [];

  if (q) {
    for (const b of mockBills) {
      const haystack = [
        b.title,
        b.identifier,
        b.purpose ?? '',
        b.description,
        ...(b.topics ?? []),
      ].join(' ').toLowerCase();
      if (haystack.includes(q)) {
        results.push({
          kind: 'bill',
          id: b.id,
          title: b.title,
          snippet: b.purpose ?? b.description ?? '',
          url: `/bills/${b.id}`,
          published_at: b.updated_at,
        });
      }
    }
    for (const item of mockBriefing.items) {
      if (item.title.toLowerCase().includes(q) || item.description.toLowerCase().includes(q)) {
        results.push({
          kind: item.kind,
          id: item.bill_id ?? '',
          title: item.title,
          snippet: item.description,
          url: item.bill_id ? `/bills/${item.bill_id}` : '/briefing',
          published_at: mockBriefing.date,
        });
      }
    }
  }

  const suggested = [
    'Bills about housing',
    'Government action on AI',
    'What happened in Parliament this week?',
    'What changed in data protection legislation?',
    'Laws affecting small businesses',
  ];

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Search</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Search across Bills, Acts, regulations, Hansard, committee reports,
          Order Papers, and Gazette notices. Ask a question in plain language.
        </p>
      </header>

      <form className="mt-6">
        <label htmlFor="q" className="sr-only">Search query</label>
        <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-paper px-3 py-2">
          <SearchIcon className="h-4 w-4 text-civic-stone" aria-hidden="true" />
          <input
            id="q"
            type="search"
            name="q"
            defaultValue={searchParams.q ?? ''}
            placeholder="Search a Bill, topic, or ask a question…"
            autoFocus
            className="flex-1 bg-transparent text-sm focus:outline-none"
          />
          <button type="submit" className="rounded-md bg-civic-leaf px-4 py-1.5 text-sm font-semibold text-white hover:bg-civic-leaf/90">
            Search
          </button>
        </div>
      </form>

      {!q && (
        <div className="mt-6">
          <h2 className="text-sm font-semibold text-civic-ink">Try</h2>
          <ul className="mt-2 flex flex-wrap gap-2">
            {suggested.map((s) => (
              <li key={s}>
                <Link href={`/search?q=${encodeURIComponent(s)}`} className="rounded-full bg-civic-mist px-3 py-1 text-sm text-civic-stone hover:bg-civic-leaf/10 hover:text-civic-leaf">
                  {s}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}

      {q && (
        <div className="mt-8">
          <p className="text-sm text-civic-stone">
            {results.length} result(s) for &ldquo;{searchParams.q}&rdquo;
          </p>
          {results.length === 0 ? (
            <div className="mt-4 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
              No verified results. Try a different query, or{' '}
              <Link href="/about" className="text-civic-leaf hover:underline">learn how to ask better questions</Link>.
            </div>
          ) : (
            <ul className="mt-4 space-y-3">
              {results.map((r) => (
                <li key={`${r.kind}-${r.id}`} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
                  <Link href={r.url} className="block">
                    <div className="flex items-center justify-between text-xs text-civic-stone">
                      <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium uppercase">{r.kind.replace('_', ' ')}</span>
                      {r.published_at && <span>{formatDate(r.published_at)}</span>}
                    </div>
                    <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">{r.title}</h2>
                    {r.snippet && <p className="mt-1 line-clamp-2 text-sm text-civic-stone">{r.snippet}</p>}
                    <span className="mt-2 inline-flex items-center gap-0.5 text-xs text-civic-leaf">
                      View <ExternalLink className="h-3 w-3" aria-hidden="true" />
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
