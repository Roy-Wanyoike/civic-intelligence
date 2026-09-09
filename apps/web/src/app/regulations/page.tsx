import Link from 'next/link';
import { ExternalLink, FileText, Filter } from 'lucide-react';
import { formatDate } from '@/lib/utils';
import { FilterSelect } from '@/components/filter-select';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Regulations',
  description: 'Subsidiary legislation made under Kenyan Acts of Parliament — every regulation links to its official source.',
};

const STATUS_FILTERS = ['all', 'in_force', 'amended', 'repealed'] as const;

// regulationItem mirrors the JSON shape returned by GET /api/v1/regulations (services/api).
interface RegulationItem {
  id: string;
  title: string;
  parent_act: string;
  gazette_notice?: string;
  effective_date?: string;
  status: 'in_force' | 'amended' | 'repealed';
  source_url: string;
  country: string;
  summary?: string;
}

// Sample data mirroring services/api/cmd/main.go sampleRegulations — kept in sync
// so the page renders immediately on the static frontend without a server
// round-trip. When the BFF is wired (issue #19 follow-up), the page can fetch
// /api/v1/regulations at runtime; until then we ship the verified seed
// client-side.
const regulations: RegulationItem[] = [
  {
    id: 'ke-reg-data-protection-general-2021',
    title: 'Data Protection (General) Regulations, 2021',
    parent_act: 'Data Protection Act, 2019 (No. 24 of 2019)',
    gazette_notice: 'Legal Notice No. 184 of 2021',
    effective_date: '2022-02-14',
    source_url: 'https://www.kenyalaw.org/kl/index.php?id=11853',
    status: 'in_force',
    country: 'KE',
    summary:
      'Operationalises the Data Protection Act — sets out registration of data controllers and processors, data protection impact assessments, and data subject rights procedures.',
  },
  {
    id: 'ke-reg-data-protection-complaints-2021',
    title:
      'Data Protection (Complaints Handling Procedure and Enforcement) Regulations, 2021',
    parent_act: 'Data Protection Act, 2019 (No. 24 of 2019)',
    gazette_notice: 'Legal Notice No. 185 of 2021',
    effective_date: '2022-02-14',
    source_url: 'https://www.kenyalaw.org/kl/index.php?id=11854',
    status: 'in_force',
    country: 'KE',
    summary:
      'Establishes the complaints-handling and enforcement procedure before the Office of the Data Protection Commissioner, including investigation, undertakings, and penalties.',
  },
  {
    id: 'ke-reg-cbk-prudential-2013',
    title: 'Central Bank of Kenya Prudential Regulations (Banking Act)',
    parent_act: 'Banking Act, 2012 (Cap 488)',
    gazette_notice: 'Various Legal Notices',
    effective_date: '2013-06-14',
    source_url: 'https://www.centralbank.go.ke/regulations/',
    status: 'amended',
    country: 'KE',
    summary:
      'Suite of prudential regulations issued by the Central Bank of Kenya governing capital adequacy, liquidity, risk management, and corporate governance for deposit-taking institutions.',
  },
  {
    id: 'ke-reg-public-procurement-2020',
    title: 'Public Procurement and Asset Disposal Regulations, 2020',
    parent_act: 'Public Procurement and Asset Disposal Act, 2015 (No. 33A of 2015)',
    gazette_notice: 'Legal Notice No. 140 of 2020',
    effective_date: '2020-07-31',
    source_url: 'https://www.kenyalaw.org/kl/index.php?id=11256',
    status: 'amended',
    country: 'KE',
    summary:
      'Implements the Public Procurement and Asset Disposal Act — sets procedures for procurement by public entities, including preferences and reservations, electronic procurement, and review mechanisms.',
  },
  {
    id: 'ke-reg-elections-campaign-financing-2023',
    title: 'Elections (Campaign Financing) Regulations, 2023',
    parent_act: 'Elections Act, 2011 (No. 24 of 2011)',
    gazette_notice: 'Legal Notice No. 86 of 2023',
    effective_date: '2023-08-04',
    source_url: 'https://www.kenyalaw.org/kl/index.php?id=12994',
    status: 'in_force',
    country: 'KE',
    summary:
      'Regulates the sources, limits, and disclosure of campaign financing for candidates and political parties, giving effect to Part III of the Elections Act.',
  },
];

const STATUS_LABELS: Record<RegulationItem['status'], string> = {
  in_force: 'In force',
  amended: 'Amended',
  repealed: 'Repealed',
};

function statusBadgeClass(status: RegulationItem['status']): string {
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

function sourceLabel(url: string): string {
  if (url.includes('centralbank.go.ke')) return 'Central Bank of Kenya';
  if (url.includes('kenyalaw.org')) return 'Kenya Law';
  return 'Official source';
}

export default function RegulationsPage({
  searchParams,
}: {
  searchParams: { status?: string; parent_act?: string; q?: string };
}) {
  const status = searchParams.status ?? 'all';
  const parentAct = searchParams.parent_act?.toLowerCase().trim() ?? '';
  const q = searchParams.q?.toLowerCase().trim();

  let filtered = regulations.slice();
  if (status !== 'all') filtered = filtered.filter((r) => r.status === status);
  if (parentAct) {
    filtered = filtered.filter((r) =>
      r.parent_act.toLowerCase().includes(parentAct),
    );
  }
  if (q) {
    filtered = filtered.filter((r) => {
      const haystack = `${r.title} ${r.parent_act} ${r.summary ?? ''}`.toLowerCase();
      return haystack.includes(q);
    });
  }

  // Sort newest effective_date first.
  filtered.sort((a, b) => {
    const ax = a.effective_date ? Date.parse(a.effective_date) : 0;
    const bx = b.effective_date ? Date.parse(b.effective_date) : 0;
    return bx - ax;
  });

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <FileText className="h-4 w-4" aria-hidden="true" />
          <span>Subsidiary Legislation</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Regulations
        </h1>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          Subsidiary legislation made under Kenyan Acts of Parliament —
          statutory instruments, gazette notices, and prudential rules issued by
          the Cabinet, regulators, and county governments. Each regulation
          links to its official source.
        </p>
        {filtered.length > 0 && (
          <p className="mt-3 text-sm text-civic-ink">
            <span className="font-medium">{filtered.length}</span> regulations
            shown
          </p>
        )}
      </header>

      <form
        className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
        aria-label="Filter Regulations"
      >
        <FilterSelect
          label="Status"
          name="status"
          value={status}
          options={STATUS_FILTERS}
          basePath="/regulations"
        />
        <div>
          <label htmlFor="parent_act" className="sr-only">
            Filter by parent Act
          </label>
          <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-paper px-3 py-2">
            <Filter className="h-4 w-4 text-civic-stone" aria-hidden="true" />
            <input
              id="parent_act"
              type="search"
              name="parent_act"
              defaultValue={searchParams.parent_act ?? ''}
              placeholder="Filter by parent Act (e.g. Data Protection)"
              className="flex-1 bg-transparent text-sm focus:outline-none"
            />
          </div>
        </div>
        <div>
          <label htmlFor="q" className="sr-only">
            Search within Regulations
          </label>
          <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-paper px-3 py-2">
            <Filter className="h-4 w-4 text-civic-stone" aria-hidden="true" />
            <input
              id="q"
              type="search"
              name="q"
              defaultValue={searchParams.q ?? ''}
              placeholder="Search title, parent Act, or summary"
              className="flex-1 bg-transparent text-sm focus:outline-none"
            />
          </div>
        </div>
      </form>

      {filtered.length === 0 ? (
        <div className="rounded-lg border border-dashed border-civic-border p-12 text-center text-civic-stone">
          No regulations match the current filters.
        </div>
      ) : (
        <ul className="grid gap-4">
          {filtered.map((r) => (
            <li
              key={r.id}
              className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2 text-xs text-civic-stone">
                    <span
                      className={`rounded-full px-2 py-0.5 font-medium uppercase tracking-wide ${statusBadgeClass(r.status)}`}
                    >
                      {STATUS_LABELS[r.status]}
                    </span>
                    {r.gazette_notice && (
                      <span>&middot; {r.gazette_notice}</span>
                    )}
                  </div>
                  <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                    {r.title}
                  </h2>
                  <p className="mt-1 text-xs text-civic-stone">
                    <span className="font-medium text-civic-ink">Parent Act:</span>{' '}
                    {r.parent_act}
                  </p>
                  {r.summary && (
                    <p className="mt-1 line-clamp-3 text-sm text-civic-stone">
                      {r.summary}
                    </p>
                  )}
                  <div className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-xs text-civic-stone">
                    {r.effective_date && (
                      <span>
                        <span className="font-medium text-civic-ink">Effective:</span>{' '}
                        {formatDate(r.effective_date)}
                      </span>
                    )}
                  </div>
                </div>
              </div>
              {r.source_url && (
                <div className="mt-3 border-t border-civic-border pt-3">
                  <Link
                    href={r.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-xs font-medium text-civic-leaf hover:underline"
                  >
                    <ExternalLink className="h-3 w-3" aria-hidden="true" />
                    View on {sourceLabel(r.source_url)}
                  </Link>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      <p className="mt-8 text-xs text-civic-stone">
        Sample data is mirrored from <code>services/api/cmd/main.go</code> (verified
        against kenyalaw.org and centralbank.go.ke). The Go BFF exposes{' '}
        <code>GET /api/v1/regulations</code> and{' '}
        <code>GET /api/v1/regulations/{'{id}'}</code>; the database read path will be
        wired in issue #19.
      </p>
    </div>
  );
}
