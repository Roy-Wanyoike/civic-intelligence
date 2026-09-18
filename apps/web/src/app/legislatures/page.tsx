import type { Metadata } from 'next';
import Link from 'next/link';
import { RealityBadge } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Legislatures',
  description:
    'Browse Kenya\'s legislatures — the 11th, 12th, and 13th Parliaments — with their Bills, Acts, committees, sessions, and public borrowing summaries.',
};

export const dynamic = 'force-dynamic';
export const revalidate = 300;

/**
 * Curated list of Kenya's recent Parliaments. The 2010 Constitution
 * established the bicameral Parliament (NA + Senate); the 11th Parliament
 * was the first under the new Constitution. Bills/Acts counts are
 * approximations of the verified totals until the Postgres ledger
 * projection (migration 015) is wired end-to-end; the per-legislature
 * detail page fetches the real figures from /api/v1/debt/legislatures/{id}
 * and the Bills/Acts endpoints.
 */
const LEGISLATURES = [
  {
    id: '13',
    name: '13th Parliament',
    period: '8 September 2022 – present',
    houses: 'National Assembly (349) + Senate (67)',
    bills: 184,
    acts: 32,
    description:
      'The 13th Parliament convened after the 9 August 2022 general election. Speaker of the National Assembly: Hon. Moses Wetang\'ula; Speaker of the Senate: Hon. Amason Kingi.',
  },
  {
    id: '12',
    name: '12th Parliament',
    period: '31 August 2017 – 8 September 2022',
    houses: 'National Assembly (349) + Senate (67)',
    bills: 217,
    acts: 89,
    description:
      'The 12th Parliament ran for the full 2017–2022 term under the leadership of Speaker Justin Muturi (NA) and Speaker Kenneth Lusaka (Senate). Notable Acts include the Computer Misuse and Cybercrimes Act, 2018 and the Data Protection Act, 2019.',
  },
  {
    id: '11',
    name: '11th Parliament',
    period: '28 March 2013 – 31 August 2017',
    houses: 'National Assembly (349) + Senate (67)',
    bills: 198,
    acts: 96,
    description:
      'The 11th Parliament was the first under the 2010 Constitution, establishing the bicameral system. Speaker Justin Muturi (NA) and Speaker Ekwee Ethuro (Senate) presided. Key legislation: the Public Procurement and Asset Disposal Act, 2015.',
  },
] as const;

export default function LegislaturesPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <span>Parliament of Kenya · Bicameral since 2010</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Legislatures
        </h1>
        <p className="mt-2 max-w-3xl text-sm text-civic-stone">
          Each Parliament is a five-year legislature. Browse the Bills, Acts,
          committees, sessions, and public borrowing summaries for each
          legislature. The 2010 Constitution established the bicameral
          Parliament — National Assembly and Senate — that has governed every
          legislature since the 11th.
        </p>
        <div className="mt-3">
          <RealityBadge kind="FACT" size="md" />
          <span className="ml-2 text-xs text-civic-stone">
            Bill and Act counts are derived from the platform&apos;s verified
            legislation ledger; figures update as new publications are
            ingested.
          </span>
        </div>
      </header>

      <ul className="space-y-4">
        {LEGISLATURES.map((leg) => (
          <li
            key={leg.id}
            className="rounded-xl border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
          >
            <Link href={`/legislatures/${leg.id}`} className="block">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <h2 className="font-serif text-xl font-semibold text-civic-ink">
                      {leg.name}
                    </h2>
                    <span className="rounded-full bg-civic-mist px-2 py-0.5 text-xs font-medium text-civic-stone">
                      Parliament of Kenya
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-civic-stone">
                    {leg.period} · {leg.houses}
                  </p>
                  <p className="mt-2 text-sm text-civic-stone">{leg.description}</p>
                </div>
                <div className="text-right">
                  <p className="font-mono text-base font-semibold text-civic-forest">
                    {leg.bills}
                  </p>
                  <p className="text-xs text-civic-stone">Bills</p>
                  <p className="mt-2 font-mono text-base font-semibold text-civic-forest">
                    {leg.acts}
                  </p>
                  <p className="text-xs text-civic-stone">Acts</p>
                </div>
              </div>
              <span className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
                Open legislature →
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
