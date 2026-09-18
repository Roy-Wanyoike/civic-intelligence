import type { Metadata } from 'next';
import Link from 'next/link';
import { RealityBadge } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Uganda',
  description:
    'Civic Intelligence for Uganda — Parliament of Uganda (unicameral, 556 members), Bills, government structure, and the parliament.go.ug source adapter.',
};

export const dynamic = 'force-dynamic';

/**
 * Curated seed Bills for Uganda — mirrored from
 * adapters/uganda/internal/uganda_data.go (UgandaSampleBills). Kept in
 * page-local constants (not imported from the adapter) because the
 * adapter is Go; the page renders the seed data verbatim until the live
 * /api/v1/bills?country=UG endpoint is wired (issue #19).
 */
const UGANDA_BILLS = [
  {
    id: 'ug-national-coffee-bill-2024',
    title: 'The National Coffee Bill, 2024',
    number: 'The Bill No. 12 of 2024',
    sponsor: 'Hon. Minister of Agriculture, Animal Industry and Fisheries',
    stage: 'Second Reading',
    source_url: 'https://www.parliament.go.ug/business/bills/national-coffee-bill-2024',
  },
  {
    id: 'ug-traffic-road-safety-amendment-bill-2023',
    title: 'The Traffic and Road Safety (Amendment) Bill, 2023',
    number: 'The Bill No. 8 of 2023',
    sponsor: 'Hon. Minister of Works and Transport',
    stage: 'Committee Stage',
    source_url: 'https://www.parliament.go.ug/business/bills/traffic-road-safety-amendment-bill-2023',
  },
  {
    id: 'ug-anti-corruption-amendment-bill-2024',
    title: 'The Anti-Corruption (Amendment) Bill, 2024',
    number: 'The Bill No. 5 of 2024',
    sponsor: 'Hon. Attorney General',
    stage: 'First Reading',
    source_url: 'https://www.parliament.go.ug/business/bills/anti-corruption-amendment-bill-2024',
  },
  {
    id: 'ug-public-finance-management-amendment-bill-2024',
    title: 'The Public Finance Management (Amendment) Bill, 2024',
    number: 'The Bill No. 15 of 2024',
    sponsor: 'Hon. Minister of Finance, Planning and Economic Development',
    stage: 'Third Reading',
    source_url: 'https://www.parliament.go.ug/business/bills/public-finance-management-amendment-bill-2024',
  },
  {
    id: 'ug-data-protection-privacy-amendment-bill-2023',
    title: 'The Data Protection and Privacy (Amendment) Bill, 2023',
    number: 'The Bill No. 3 of 2023',
    sponsor: 'Hon. Minister of ICT and National Guidance',
    stage: 'Presidential Assent',
    source_url: 'https://www.parliament.go.ug/business/bills/data-protection-privacy-amendment-bill-2023',
  },
] as const;

const STAGES = [
  { name: 'First Reading', explanation: 'The Bill is read for the first time in Parliament. No debate yet.' },
  { name: 'Second Reading', explanation: 'MPs debate the principles of the Bill. A vote decides whether it proceeds.' },
  { name: 'Committee Stage', explanation: 'A sectoral committee examines the Bill clause-by-clause.' },
  { name: 'Report Stage', explanation: 'The committee reports back to Parliament. Further amendments may be proposed.' },
  { name: 'Third Reading', explanation: 'Final debate and vote on whether to pass the Bill.' },
  { name: 'Presidential Assent', explanation: 'The President signs the Bill into law. May refer it back once.' },
  { name: 'Commencement', explanation: 'The Act comes into force.' },
];

export default function UgandaCountryPage() {
  // The Parliament of Uganda is unicameral: 556 members total —
  // constituency members + women district representatives + special
  // interest groups (UPDF, youth, workers, persons with disabilities,
  // older persons, etc.). Source: https://www.parliament.go.ug
  const parliamentMembers = 556;

  return (
    <div>
      <section className="bg-gradient-to-b from-civic-forest to-civic-ink text-white">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">
            Country · 🇺🇬 Uganda
          </p>
          <h1 className="mt-2 font-serif text-4xl font-semibold sm:text-5xl">
            Uganda Civic Intelligence
          </h1>
          <p className="mt-3 max-w-2xl text-white/90">
            The Parliament of Uganda is unicameral, comprising {parliamentMembers}{' '}
            members. The country adapter below sources from{' '}
            <a
              href="https://www.parliament.go.ug"
              className="underline decoration-civic-acacia/60 underline-offset-4 hover:text-civic-acacia"
              target="_blank"
              rel="noopener noreferrer"
            >
              parliament.go.ug
            </a>
            . The 11th Parliament (2021–2026) is the current legislature.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <div className="flex items-center gap-2">
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">
            Institutions
          </h2>
          <RealityBadge kind="FACT" />
        </div>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[
            {
              name: 'Parliament of Uganda',
              desc: `Unicameral legislature — ${parliamentMembers} members serving a 5-year term`,
            },
            {
              name: 'Office of the President',
              desc: 'Head of state and government; assents to Bills passed by Parliament',
            },
            {
              name: 'Office of the Attorney General',
              desc: 'Chief legal advisor to the government; principal draftsman of Bills',
            },
            {
              name: 'Cabinet of Uganda',
              desc: 'Comprises the President, Vice President, Prime Minister, and Cabinet Ministers',
            },
            {
              name: 'Judiciary of Uganda',
              desc: 'Supreme Court, Court of Appeal, High Court, and Magistrates Courts',
            },
            {
              name: 'Electoral Commission of Uganda',
              desc: 'Manages public elections and demarcates constituencies',
            },
          ].map((i) => (
            <li
              key={i.name}
              className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf"
            >
              <Link href="/institutions" className="block">
                <h3 className="font-serif text-base font-semibold text-civic-ink">{i.name}</h3>
                <p className="mt-1 text-xs text-civic-stone">{i.desc}</p>
              </Link>
            </li>
          ))}
        </ul>
      </section>

      <section className="border-t border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">
            Bill lifecycle in Parliament
          </h2>
          <p className="mt-2 max-w-2xl text-sm text-civic-stone">
            Uganda&apos;s Bill stages run from First Reading through to
            Commencement, in line with the Rules of Procedure of Parliament.
            A Bill may be Rejected at a vote.
          </p>
          <ol className="mt-4 grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {STAGES.map((s, idx) => (
              <li
                key={s.name}
                className="rounded-lg border border-civic-border bg-civic-paper p-4"
              >
                <div className="flex items-center gap-2">
                  <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-civic-leaf/15 text-xs font-semibold text-civic-leaf">
                    {idx + 1}
                  </span>
                  <h3 className="font-serif text-base font-semibold text-civic-ink">{s.name}</h3>
                </div>
                <p className="mt-2 text-xs text-civic-stone">{s.explanation}</p>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section className="border-t border-civic-border">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
          <div className="flex items-center gap-2">
            <h2 className="font-serif text-2xl font-semibold text-civic-forest">
              Bills before Parliament
            </h2>
            <RealityBadge kind="EVIDENCE" />
            <span className="text-xs text-civic-stone">
              Seed data sourced from parliament.go.ug
            </span>
          </div>
          <ul className="mt-4 grid gap-3 md:grid-cols-2">
            {UGANDA_BILLS.map((b) => (
              <li
                key={b.id}
                className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf"
              >
                <a
                  href={b.source_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block"
                >
                  <div className="text-xs text-civic-stone">
                    {b.number}
                  </div>
                  <h3 className="mt-1 font-serif text-base font-semibold text-civic-ink">
                    {b.title}
                  </h3>
                  <p className="mt-1 text-xs text-civic-stone">
                    Sponsor: {b.sponsor}
                  </p>
                  <p className="mt-1 text-xs text-civic-stone">Stage: {b.stage}</p>
                  <span className="mt-2 inline-flex items-center gap-1 text-xs text-civic-leaf hover:underline">
                    View on parliament.go.ug →
                  </span>
                </a>
              </li>
            ))}
          </ul>
          <Link
            href="/bills"
            className="mt-6 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
          >
            All Bills →
          </Link>
        </div>
      </section>

      <section className="border-t border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">
            Adapter status
          </h2>
          <div className="mt-4 rounded-lg border border-civic-border bg-civic-paper p-4">
            <div className="flex items-center justify-between">
              <span className="font-serif text-base font-semibold text-civic-ink">
                adapters/uganda
              </span>
              <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 text-xs font-medium text-civic-leaf">
                Seed + parliament parser ready
              </span>
            </div>
            <p className="mt-2 text-xs text-civic-stone">
              The Uganda country adapter implements{' '}
              <code>contracts.LegislativeSourceAdapter</code> with seed Bills
              sourced from public parliament.go.ug records. The
              parliament.go.ug Bills-list parser is implemented and exercised
              against the test fixture in
              <code> adapters/uganda/parliament/testdata/bills.html</code>.
              Live discovery is pending the upstream Bills listing being
              available at the time of fetch.
            </p>
            <Link
              href="/country"
              className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
            >
              ← All countries
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
