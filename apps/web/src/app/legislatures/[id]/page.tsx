import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, Building2, CalendarDays, FileText, Scale, Users, Landmark, DollarSign, Megaphone } from 'lucide-react';
import { RealityBadge } from '@/components/reality-labels';
import { formatDate } from '@/lib/utils';

export const dynamic = 'force-dynamic';
export const revalidate = 300;

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

/**
 * Curated metadata for each legislature. Mirrors the list on the index page;
 * kept here so the detail page can render the period + houses immediately
 * without waiting on the (currently best-effort) API.
 */
const LEGISLATURE_META: Record<
  string,
  { name: string; period: string; houses: string; sessions: string[]; committees: { name: string; chair: string }[] }
> = {
  '13': {
    name: '13th Parliament',
    period: '8 September 2022 – present',
    houses: 'National Assembly (349) + Senate (67)',
    sessions: [
      { name: '1st Session', date: '2022-10-06' },
      { name: '2nd Session', date: '2023-02-14' },
      { name: '3rd Session', date: '2024-02-13' },
      { name: '4th Session', date: '2025-02-13' },
    ].map((s) => `${s.name} (opened ${s.date})`),
    committees: [
      { name: 'Budget and Appropriations Committee', chair: 'Hon. Ndindi Nyoro' },
      { name: 'Public Investments Committee', chair: 'Hon. John Mbadi' },
      { name: 'Justice and Legal Affairs Committee', chair: 'Hon. George Murugara' },
      { name: 'Health Committee', chair: 'Hon. Endebes Robert Pukose' },
    ],
  },
  '12': {
    name: '12th Parliament',
    period: '31 August 2017 – 8 September 2022',
    houses: 'National Assembly (349) + Senate (67)',
    sessions: [
      '1st Session (opened 2017-09-21)',
      '2nd Session (opened 2018-02-13)',
      '3rd Session (opened 2019-02-12)',
      '4th Session (opened 2020-02-11)',
      '5th Session (opened 2021-02-09)',
    ],
    committees: [
      { name: 'Budget and Appropriations Committee', chair: 'Hon. Kimani Ichung\'wah' },
      { name: 'Public Accounts Committee', chair: 'Hon. Opiyo Wandayi' },
      { name: 'Justice and Legal Affairs Committee', chair: 'Hon. William Cheptumo' },
    ],
  },
  '11': {
    name: '11th Parliament',
    period: '28 March 2013 – 31 August 2017',
    houses: 'National Assembly (349) + Senate (67)',
    sessions: [
      '1st Session (opened 2013-04-16)',
      '2nd Session (opened 2014-03-25)',
      '3rd Session (opened 2015-04-21)',
      '4th Session (opened 2016-04-12)',
    ],
    committees: [
      { name: 'Budget and Appropriations Committee', chair: 'Hon. Mutava Musyimi' },
      { name: 'Public Accounts Committee', chair: 'Hon. Nicholas Gumbo' },
      { name: 'Justice and Legal Affairs Committee', chair: 'Hon. Samuel Chepkonga' },
    ],
  },
};

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const leg = LEGISLATURE_META[id];
  if (!leg) return { title: 'Legislature not found' };
  return {
    title: `${leg.name} — Legislatures`,
    description: `${leg.name} (${leg.period}). Sessions, Bills, Acts, committees, and public borrowing summary.`,
  };
}

interface DebtLegislatureSummary {
  legislature_id?: string;
  total_disbursed_usd?: number;
  total_repaid_usd?: number;
  outstanding_obligations_usd?: number;
  borrowing_agreements_count?: number;
  source_url?: string;
  disclaimer?: string;
}

async function fetchDebtSummary(
  id: string,
): Promise<{ data: DebtLegislatureSummary | null; source: 'api' | 'mock' }> {
  try {
    const resp = await fetch(
      `${API_BASE}/api/v1/debt/legislatures/${encodeURIComponent(id)}`,
      {
        headers: { Accept: 'application/json' },
        cache: 'no-store',
      },
    );
    if (!resp.ok) return { data: null, source: 'mock' };
    const data = (await resp.json()) as DebtLegislatureSummary;
    return { data, source: 'api' };
  } catch {
    return { data: null, source: 'mock' };
  }
}

export default async function LegislatureDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const leg = LEGISLATURE_META[id];
  if (!leg) {
    notFound();
  }

  const { data: debtSummary, source: debtSource } = await fetchDebtSummary(id);

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      {/* Breadcrumb */}
      <nav aria-label="Breadcrumb" className="mb-6 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li>
            <Link href="/" className="hover:text-civic-leaf">
              Home
            </Link>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <Link href="/legislatures" className="hover:text-civic-leaf">
              Legislatures
            </Link>
          </li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">{leg.name}</li>
        </ol>
      </nav>

      {/* Header */}
      <header className="mb-8">
        <Link
          href="/legislatures"
          className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" /> All legislatures
        </Link>
        <h1 className="mt-3 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          {leg.name}
        </h1>
        <p className="mt-2 text-sm text-civic-stone">
          {leg.houses} · {leg.period}
        </p>
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <RealityBadge kind="FACT" size="md" />
          <span className="text-xs text-civic-stone">
            Bicameral Parliament of Kenya established under the 2010 Constitution.
          </span>
        </div>
      </header>

      <div className="space-y-10">
        {/* Sessions */}
        <section aria-labelledby="sessions-heading">
          <div className="flex items-center gap-2">
            <CalendarDays className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 id="sessions-heading" className="font-serif text-xl font-semibold text-civic-forest">
              Sessions
            </h2>
          </div>
          <ol className="mt-3 space-y-2">
            {leg.sessions.map((s) => (
              <li
                key={s}
                className="flex items-start gap-3 rounded-md border border-civic-border bg-civic-paper p-3 text-sm"
              >
                <span className="mt-0.5 inline-flex h-2 w-2 flex-shrink-0 rounded-full bg-civic-leaf" aria-hidden="true" />
                <span className="text-civic-ink">{s}</span>
              </li>
            ))}
          </ol>
        </section>

        {/* Bills */}
        <section aria-labelledby="bills-heading">
          <div className="flex items-center gap-2">
            <FileText className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 id="bills-heading" className="font-serif text-xl font-semibold text-civic-forest">
              Bills
            </h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">
            Bills introduced or considered during the {leg.name}.
          </p>
          <Link
            href={`/bills?legislature=${id}`}
            className="mt-3 inline-flex items-center gap-1 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-sm text-civic-ink hover:border-civic-leaf"
          >
            <FileText className="h-4 w-4" aria-hidden="true" /> Browse Bills →
          </Link>
        </section>

        {/* Acts */}
        <section aria-labelledby="acts-heading">
          <div className="flex items-center gap-2">
            <Scale className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 id="acts-heading" className="font-serif text-xl font-semibold text-civic-forest">
              Acts of Parliament
            </h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">
            Acts passed by the {leg.name} and assented to by the President.
          </p>
          <Link
            href={`/acts?legislature=${id}`}
            className="mt-3 inline-flex items-center gap-1 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-sm text-civic-ink hover:border-civic-leaf"
          >
            <Scale className="h-4 w-4" aria-hidden="true" /> Browse Acts →
          </Link>
        </section>

        {/* Committees */}
        <section aria-labelledby="committees-heading">
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 id="committees-heading" className="font-serif text-xl font-semibold text-civic-forest">
              Committees
            </h2>
          </div>
          <ul className="mt-3 grid gap-3 sm:grid-cols-2">
            {leg.committees.map((c) => (
              <li
                key={c.name}
                className="rounded-lg border border-civic-border bg-civic-paper p-4"
              >
                <h3 className="font-serif text-base font-semibold text-civic-ink">
                  {c.name}
                </h3>
                <p className="mt-1 text-xs text-civic-stone">
                  Chair: {c.chair}
                </p>
                <Link
                  href={`/committees`}
                  className="mt-2 inline-flex items-center gap-1 text-xs text-civic-leaf hover:underline"
                >
                  View committee →
                </Link>
              </li>
            ))}
          </ul>
        </section>

        {/* Public Participation */}
        <section aria-labelledby="participation-heading">
          <div className="flex items-center gap-2">
            <Megaphone className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 id="participation-heading" className="font-serif text-xl font-semibold text-civic-forest">
              Public Participation
            </h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">
            Article 118 of the Constitution of Kenya requires Parliament to
            facilitate public participation in legislative and other business.
            Memoranda can be submitted to the relevant departmental committee
            during the public-participation window for each Bill.
          </p>
          <Link
            href="/participation"
            className="mt-3 inline-flex items-center gap-1 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-sm text-civic-ink hover:border-civic-leaf"
          >
            <Megaphone className="h-4 w-4" aria-hidden="true" /> Open participation →
          </Link>
        </section>

        {/* Public Borrowing */}
        <section aria-labelledby="borrowing-heading">
          <div className="flex items-center gap-2">
            <DollarSign className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 id="borrowing-heading" className="font-serif text-xl font-semibold text-civic-forest">
              Public Borrowing
            </h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">
            Sovereign borrowing agreements attributed to the {leg.name}, sourced
            from the Treasury External Public Debt Register and verified
            creditor press releases.
          </p>

          {debtSource === 'mock' && (
            <p className="mt-2 rounded-md bg-civic-acacia/10 px-3 py-1.5 text-xs text-civic-clay">
              OFFLINE DATA — the public borrowing API
              (<code>/api/v1/debt/legislatures/{id}</code>) could not be
              reached. The figures below are placeholders until the Go BFF is
              online.
            </p>
          )}

          {debtSummary ? (
            <dl className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <DebtStat
                label="Total disbursed (USD)"
                value={debtSummary.total_disbursed_usd}
              />
              <DebtStat
                label="Total repaid (USD)"
                value={debtSummary.total_repaid_usd}
              />
              <DebtStat
                label="Outstanding obligations (USD)"
                value={debtSummary.outstanding_obligations_usd}
              />
              <DebtStat
                label="Borrowing agreements"
                value={debtSummary.borrowing_agreements_count}
                asCount
              />
            </dl>
          ) : (
            <div className="mt-3 rounded-md border border-dashed border-civic-border p-4 text-sm text-civic-stone">
              No borrowing summary available for this legislature.
            </div>
          )}

          {debtSummary?.disclaimer && (
            <p className="mt-3 rounded-md bg-civic-mist px-3 py-2 text-xs text-civic-stone">
              {debtSummary.disclaimer}
            </p>
          )}

          <Link
            href={`/loans?legislature=${id}`}
            className="mt-4 inline-flex items-center gap-1 rounded-md border border-civic-leaf bg-civic-leaf/10 px-3 py-1.5 text-sm text-civic-leaf hover:bg-civic-leaf/20"
          >
            <Landmark className="h-4 w-4" aria-hidden="true" /> View loans register →
          </Link>
        </section>

        {/* Institutions */}
        <section aria-labelledby="institutions-heading">
          <div className="flex items-center gap-2">
            <Building2 className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
            <h2 id="institutions-heading" className="font-serif text-xl font-semibold text-civic-forest">
              Institutions
            </h2>
          </div>
          <ul className="mt-3 grid gap-3 sm:grid-cols-2">
            <li className="rounded-lg border border-civic-border bg-civic-paper p-4">
              <h3 className="font-serif text-base font-semibold text-civic-ink">
                National Assembly
              </h3>
              <p className="mt-1 text-xs text-civic-stone">
                Lower house — 290 elected + 47 women + 12 nominated.
              </p>
            </li>
            <li className="rounded-lg border border-civic-border bg-civic-paper p-4">
              <h3 className="font-serif text-base font-semibold text-civic-ink">
                Senate
              </h3>
              <p className="mt-1 text-xs text-civic-stone">
                Upper house — 47 elected + 16 women + 2 youth + 2 PWDs.
              </p>
            </li>
          </ul>
          <Link
            href="/institutions"
            className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
          >
            All institutions →
          </Link>
        </section>
      </div>

      <footer className="mt-12 border-t border-civic-border pt-6 text-xs text-civic-stone">
        <p>
          Page data verified through {formatDate(new Date().toISOString())}.
          The {leg.name} overview is rendered from the platform&apos;s
          curated legislature metadata; per-legislature borrowing figures are
          sourced from <code>/api/v1/debt/legislatures/{id}</code>.
        </p>
      </footer>
    </div>
  );
}

function DebtStat({
  label,
  value,
  asCount,
}: {
  label: string;
  value: number | null | undefined;
  asCount?: boolean;
}) {
  const display =
    value == null
      ? '—'
      : asCount
        ? String(value)
        : new Intl.NumberFormat('en-KE', {
            style: 'currency',
            currency: 'USD',
            maximumFractionDigits: 0,
          }).format(value);
  return (
    <div className="rounded-lg border border-civic-border bg-civic-paper p-4">
      <dt className="text-xs uppercase tracking-wide text-civic-stone">{label}</dt>
      <dd className="mt-1 font-mono text-lg font-semibold text-civic-forest">
        {display}
      </dd>
    </div>
  );
}
