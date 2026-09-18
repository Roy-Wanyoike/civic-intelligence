import Link from 'next/link';
import { ExternalLink, Scale, Filter } from 'lucide-react';
import { formatDate } from '@/lib/utils';
import { FilterSelect } from '@/components/filter-select';
import { getJSON_ as getJSON } from '@/lib/api';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Acts of Parliament',
  description: 'Browse enacted Acts of Parliament in Kenya — every Act links to its official Kenya Law source.',
};

const STATUS_FILTERS = ['all', 'in_force', 'amended', 'repealed'] as const;

// actItem mirrors the JSON shape returned by GET /api/v1/acts (services/api).
interface ActItem {
  id: string;
  title: string;
  citation: string;
  assent_date?: string;
  commencement_date?: string;
  source_url: string;
  status: 'in_force' | 'amended' | 'repealed';
  country: string;
  summary?: string;
}

const STATUS_LABELS: Record<ActItem['status'], string> = {
  in_force: 'In force',
  amended: 'Amended',
  repealed: 'Repealed',
};

function statusBadgeClass(status: ActItem['status']): string {
  switch (status) {
    case 'in_force':
      return 'bg-civic-mist text-civic-ink';
    case 'amended':
      return 'bg-amber-100 text-amber-900';
    case 'repealed':
      return 'bg-rose-100 text-rose-900';
    default:
      return 'bg-civic-mist text-civic-ink';
  }
}

export default async function ActsPage({
  searchParams,
}: {
  searchParams: { status?: string; q?: string };
}) {
  const status = searchParams.status ?? 'all';
  const q = searchParams.q?.toLowerCase().trim();

  // Fetch the canonical Acts list from the Go BFF (services/api).
  // The page is a server component, so this runs at request time on the
  // server — the browser never talks to the API directly.
  let acts: ActItem[] = [];
  let apiError = false;
  try {
    const resp = await getJSON<{ items: ActItem[]; total: number }>(
      '/api/v1/acts',
    );
    acts = Array.isArray(resp.items) ? resp.items : [];
  } catch {
    apiError = true;
  }

  let filtered = acts.slice();
  if (status !== 'all') filtered = filtered.filter((a) => a.status === status);
  if (q) {
    filtered = filtered.filter((a) => {
      const haystack = `${a.title} ${a.citation} ${a.summary ?? ''}`.toLowerCase();
      return haystack.includes(q);
    });
  }

  // Sort newest assent first.
  filtered.sort((a, b) => {
    const ax = a.assent_date ? Date.parse(a.assent_date) : 0;
    const bx = b.assent_date ? Date.parse(b.assent_date) : 0;
    return bx - ax;
  });

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Scale className="h-4 w-4" aria-hidden="true" />
          <span>Existing Law</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Acts of Parliament
        </h1>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          Enacted legislation from Kenya Law. Every Act links to its official
          source on <span className="font-medium">kenyalaw.org</span>. Bills
          that receive Presidential Assent become Acts and are listed here.
        </p>
        {filtered.length > 0 && (
          <p className="mt-3 text-sm text-civic-ink">
            <span className="font-medium">{filtered.length}</span> Acts shown
          </p>
        )}
      </header>

      {apiError ? (
        <div
          role="alert"
          className="mb-6 rounded-lg border border-rose-200 bg-rose-50 p-6 text-sm text-rose-900"
        >
          <h2 className="font-serif text-base font-semibold text-rose-900">
            We couldn&rsquo;t load the Acts list right now.
          </h2>
          <p className="mt-2">
            The page fetches the canonical Acts list from the Go BFF at{' '}
            <code className="rounded bg-rose-100 px-1 py-0.5 text-xs">
              /api/v1/acts
            </code>
            . That service is unreachable or returned an error. Please try
            again in a moment — if the problem persists, the BFF logs will
            have the underlying cause.
          </p>
        </div>
      ) : (
        <>
          <form
            className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
            aria-label="Filter Acts"
          >
            <FilterSelect
              label="Status"
              name="status"
              value={status}
              options={STATUS_FILTERS}
              basePath="/acts"
            />
            <div className="lg:col-span-2">
              <label htmlFor="q" className="sr-only">
                Search within Acts
              </label>
              <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-paper px-3 py-2">
                <Filter className="h-4 w-4 text-civic-stone" aria-hidden="true" />
                <input
                  id="q"
                  type="search"
                  name="q"
                  defaultValue={searchParams.q ?? ''}
                  placeholder="Search by title, citation, or summary"
                  className="flex-1 bg-transparent text-sm focus:outline-none"
                />
              </div>
            </div>
          </form>

          {filtered.length === 0 ? (
            <div className="rounded-lg border border-dashed border-civic-border p-12 text-center text-civic-stone">
              No Acts match the current filters.
            </div>
          ) : (
            <ul className="grid gap-4">
              {filtered.map((a) => (
                <li
                  key={a.id}
                  className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2 text-xs text-civic-stone">
                        <span
                          className={`rounded-full px-2 py-0.5 font-medium uppercase tracking-wide ${statusBadgeClass(a.status)}`}
                        >
                          {STATUS_LABELS[a.status]}
                        </span>
                        <span>&middot; {a.citation}</span>
                      </div>
                      <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                        {a.title}
                      </h2>
                      {a.summary && (
                        <p className="mt-1 line-clamp-3 text-sm text-civic-stone">
                          {a.summary}
                        </p>
                      )}
                      <div className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-xs text-civic-stone">
                        {a.assent_date && (
                          <span>
                            <span className="font-medium text-civic-ink">Assented:</span>{' '}
                            {formatDate(a.assent_date)}
                          </span>
                        )}
                        {a.commencement_date && (
                          <span>
                            <span className="font-medium text-civic-ink">Commenced:</span>{' '}
                            {formatDate(a.commencement_date)}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                  {a.source_url && (
                    <div className="mt-3 border-t border-civic-border pt-3">
                      <Link
                        href={a.source_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs font-medium text-civic-leaf hover:underline"
                      >
                        <ExternalLink className="h-3 w-3" aria-hidden="true" />
                        View on Kenya Law
                      </Link>
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      <p className="mt-8 text-xs text-civic-stone">
        This page fetches <code>GET /api/v1/acts</code> from the Go BFF at
        request time. The BFF returns the verified Kenya Law sample list
        (see <code>services/api/cmd/main.go</code>) until the legislation
        database read path is wired in (issue #19).
      </p>
    </div>
  );
}
