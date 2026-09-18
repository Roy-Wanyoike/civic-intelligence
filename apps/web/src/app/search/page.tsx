import Link from 'next/link';
import { Search as SearchIcon, ExternalLink, AlertTriangle } from 'lucide-react';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Search',
  description: 'Search Bills, Acts, regulations, and Constitution articles from across the Kenyan government — backed by the Go BFF /api/v1/search endpoint.',
};

export const dynamic = 'force-dynamic';
export const revalidate = 60;

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

/**
 * Search result item — matches the `searchItem` struct serialised by the Go
 * BFF in services/api/cmd/search.go. The BFF returns `type` (not `kind`)
 * for the discriminator field, with values: "bill" | "act" |
 * "constitution_article". The `score` field is internal to the BFF and is
 * not serialised (tagged `json:"-"` in Go).
 */
interface SearchItem {
  type: 'bill' | 'act' | 'constitution_article' | string;
  id: string;
  title: string;
  snippet?: string;
  url: string;
}

interface SearchResponse {
  q?: string;
  items: SearchItem[];
  total?: number;
  note?: string;
}

interface FetchResult {
  ok: true;
  data: SearchResponse;
  elapsedMs: number;
}

interface FetchError {
  ok: false;
  error: string;
  status?: number;
}

async function runSearch(q: string): Promise<FetchResult | FetchError> {
  const start = Date.now();
  try {
    const url = `${API_BASE}/api/v1/search?q=${encodeURIComponent(q)}`;
    const resp = await fetch(url, {
      headers: { Accept: 'application/json' },
      cache: 'no-store',
    });
    if (!resp.ok) {
      return {
        ok: false,
        error: `Search API returned HTTP ${resp.status}`,
        status: resp.status,
      };
    }
    const data = (await resp.json()) as SearchResponse;
    return { ok: true, data, elapsedMs: Date.now() - start };
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return { ok: false, error: msg };
  }
}

/** Human-readable labels for each result `type` value. */
const TYPE_LABELS: Record<string, string> = {
  bill: 'Bills',
  act: 'Acts',
  constitution_article: 'Constitution Articles',
};

const TYPE_ORDER: string[] = ['bill', 'act', 'constitution_article'];

function groupByType(items: SearchItem[]): Array<{ type: string; label: string; items: SearchItem[] }> {
  const buckets = new Map<string, SearchItem[]>();
  for (const it of items) {
    const arr = buckets.get(it.type) ?? [];
    arr.push(it);
    buckets.set(it.type, arr);
  }
  // Stable ordering: known types first (in TYPE_ORDER), then any extras
  // alphabetically.
  const known = TYPE_ORDER.filter((t) => buckets.has(t));
  const extras = Array.from(buckets.keys())
    .filter((t) => !TYPE_ORDER.includes(t))
    .sort();
  return [...known, ...extras].map((t) => ({
    type: t,
    label: TYPE_LABELS[t] ?? t.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()) + 's',
    items: buckets.get(t)!,
  }));
}

const SUGGESTED = [
  'Bills about housing',
  'Government action on AI',
  'What happened in Parliament this week?',
  'What changed in data protection legislation?',
  'Laws affecting small businesses',
];

export default async function SearchPage({
  searchParams,
}: {
  searchParams: { q?: string };
}) {
  const rawQ = (searchParams.q ?? '').trim();
  const q = rawQ;
  const result = q ? await runSearch(q) : null;

  const items = result && result.ok ? result.data.items ?? [] : [];
  const total = result && result.ok ? result.data.total ?? items.length : 0;
  const elapsedMs = result && result.ok ? result.elapsedMs : 0;
  const note = result && result.ok ? result.data.note : undefined;
  const error = result && !result.ok ? result.error : null;
  const groups = groupByType(items);

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Search</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Search across Bills, Acts, regulations, Hansard, committee reports,
          Order Papers, and Gazette notices. Ask a question in plain language.
        </p>
      </header>

      <form className="mt-6" role="search">
        <label htmlFor="q" className="sr-only">
          Search query
        </label>
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
          <button
            type="submit"
            className="rounded-md bg-civic-leaf px-4 py-1.5 text-sm font-semibold text-white hover:bg-civic-leaf/90"
          >
            Search
          </button>
        </div>
      </form>

      {!q && (
        <div className="mt-6">
          <h2 className="text-sm font-semibold text-civic-ink">Try</h2>
          <ul className="mt-2 flex flex-wrap gap-2">
            {SUGGESTED.map((s) => (
              <li key={s}>
                <Link
                  href={`/search?q=${encodeURIComponent(s)}`}
                  className="rounded-full bg-civic-mist px-3 py-1 text-sm text-civic-stone hover:bg-civic-leaf/10 hover:text-civic-leaf"
                >
                  {s}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}

      {q && error && (
        <div
          className="mt-8 rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900"
          role="alert"
        >
          <div className="flex items-start gap-2">
            <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
            <div>
              <p className="font-medium">The search service is unavailable right now.</p>
              <p className="mt-1 text-xs">
                {error} — the Go BFF at <code>{API_BASE}</code> may be down or
                the query could not be parsed. Please try again in a few
                minutes.
              </p>
            </div>
          </div>
        </div>
      )}

      {q && !error && (
        <div className="mt-8">
          {/* Result count + search time */}
          <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-civic-stone">
            <p>
              {total === 0 ? (
                <>No results for &ldquo;{q}&rdquo;</>
              ) : (
                <>
                  <span className="font-medium text-civic-ink">{total}</span>{' '}
                  result{total === 1 ? '' : 's'} for &ldquo;{q}&rdquo;
                </>
              )}
            </p>
            <p className="text-xs">
              {elapsedMs > 0 && <>search time: {elapsedMs}ms · </>}
              source: <code>/api/v1/search</code>
            </p>
          </div>

          {/* No results empty state */}
          {total === 0 && (
            <div className="mt-4 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
              No verified results. Try a different query, or{' '}
              <Link href="/about" className="text-civic-leaf hover:underline">
                learn how to ask better questions
              </Link>
              .
            </div>
          )}

          {/* Grouped results */}
          {total > 0 && (
            <div className="mt-6 space-y-8">
              {groups.map((g) => (
                <section key={g.type} aria-label={g.label}>
                  <div className="flex items-center justify-between border-b border-civic-border pb-2">
                    <h2 className="text-sm font-semibold uppercase tracking-wide text-civic-ink">
                      {g.label}
                    </h2>
                    <span className="text-xs text-civic-stone">
                      {g.items.length} match{g.items.length === 1 ? '' : 'es'}
                    </span>
                  </div>
                  <ul className="mt-3 space-y-3">
                    {g.items.map((r) => (
                      <li
                        key={`${r.type}-${r.id}-${r.url}`}
                        className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf"
                      >
                        <a href={r.url} className="block" target="_blank" rel="noopener noreferrer">
                          <div className="flex items-center justify-between text-xs text-civic-stone">
                            <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium uppercase tracking-wide">
                              {r.type.replace('_', ' ')}
                            </span>
                            {r.id && <span className="font-mono">{r.id}</span>}
                          </div>
                          <h3 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                            {r.title}
                          </h3>
                          {r.snippet && (
                            <p className="mt-1 line-clamp-2 text-sm text-civic-stone">
                              {r.snippet}
                            </p>
                          )}
                          <span className="mt-2 inline-flex items-center gap-0.5 text-xs text-civic-leaf">
                            View <ExternalLink className="h-3 w-3" aria-hidden="true" />
                          </span>
                        </a>
                      </li>
                    ))}
                  </ul>
                </section>
              ))}

              {/* Stopgap note from the BFF */}
              {note && (
                <p className="rounded-md bg-civic-mist px-3 py-2 text-xs text-civic-stone">
                  {note}
                </p>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
