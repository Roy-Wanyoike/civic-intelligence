import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Nigeria',
  description:
    'Civic Intelligence for Nigeria — National Assembly, House of Representatives, Senate, and the 36 states of the Federation.',
};

export default function NigeriaCountryPage() {
  const nigeriaBills = [
    {
      id: 'ng-bill-001',
      identifier: 'HB 2024/012',
      title: 'The Electoral Act (Amendment) Bill, 2024',
      house_name: 'House of Representatives',
      current_stage: 'Second Reading',
    },
    {
      id: 'ng-bill-002',
      identifier: 'SB 2024/008',
      title: 'The Data Protection Bill, 2024',
      house_name: 'Senate',
      current_stage: 'Public Hearing',
    },
    {
      id: 'ng-bill-003',
      identifier: 'HB 2024/023',
      title: 'The Nigerian Mineral Resources (Amendment) Bill, 2024',
      house_name: 'House of Representatives',
      current_stage: 'Committee',
    },
    {
      id: 'ng-bill-004',
      identifier: 'SB 2024/015',
      title: 'The Federal Road Safety Commission (Amendment) Bill, 2024',
      house_name: 'Senate',
      current_stage: 'Third Reading',
    },
    {
      id: 'ng-bill-005',
      identifier: 'HB 2024/041',
      title: 'The Appropriation Bill, 2024',
      house_name: 'House of Representatives',
      current_stage: 'Concurrence',
    },
    {
      id: 'ng-bill-006',
      identifier: 'SB 2024/027',
      title: 'The Cybercrimes (Prohibition, Prevention, etc.) (Amendment) Bill, 2024',
      house_name: 'Senate',
      current_stage: 'Assent',
    },
  ];

  const billStages = [
    'First Reading',
    'Second Reading',
    'Public Hearing',
    'Committee',
    'Report',
    'Third Reading',
    'Concurrence',
    'Assent',
    'Commencement',
  ];

  return (
    <div>
      <section className="bg-gradient-to-b from-civic-forest to-civic-ink text-white">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">Country · 🇳🇬 Nigeria</p>
          <h1 className="mt-2 font-serif text-4xl font-semibold sm:text-5xl">Nigeria Civic Intelligence</h1>
          <p className="mt-3 max-w-2xl text-white/90">
            The National Assembly of Nigeria is bicameral under the 1999 Constitution: the
            House of Representatives (360 members) and the Senate (109 members). Below: active
            Bills and the Nigerian legislative process.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Institutions</h2>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[
            {
              name: 'House of Representatives',
              desc: 'Lower house — 360 members, each elected from one Federal Constituency',
            },
            {
              name: 'Senate',
              desc: 'Upper house — 109 members, three per state plus one for the FCT',
            },
            {
              name: 'National Assembly of Nigeria',
              desc: 'The bicameral legislature established under Section 47 of the 1999 Constitution',
            },
            {
              name: 'Office of the Attorney General of the Federation',
              desc: 'Principal legal advisor to the Federal Government (Section 150)',
            },
            {
              name: '36 State Houses of Assembly',
              desc: 'Devolved state legislatures established under Section 90 of the Constitution',
            },
          ].map((i) => (
            <li key={i.name} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
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
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Bills before the National Assembly</h2>
          <ul className="mt-4 grid gap-3 md:grid-cols-2">
            {nigeriaBills.map((b) => (
              <li key={b.id} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
                <Link href={`/bills/${b.id}`} className="block">
                  <div className="text-xs text-civic-stone">{b.identifier} · {b.house_name}</div>
                  <h3 className="mt-1 font-serif text-base font-semibold text-civic-ink">{b.title}</h3>
                  <p className="mt-1 text-xs text-civic-stone">Stage: {b.current_stage}</p>
                </Link>
              </li>
            ))}
          </ul>
          <Link href="/bills" className="mt-6 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
            All Bills →
          </Link>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">How a Bill becomes an Act in Nigeria</h2>
        <p className="mt-2 text-sm text-civic-stone">
          The Nigerian legislative process is set out in the 1999 Constitution and the Standing
          Orders of the Senate and House of Representatives. A Bill follows the stages below.
        </p>
        <ol className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {billStages.map((stage, idx) => (
            <li
              key={stage}
              className="flex items-start gap-3 rounded-lg border border-civic-border bg-civic-paper p-4"
            >
              <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-civic-leaf text-xs font-semibold text-white">
                {idx + 1}
              </span>
              <div>
                <h3 className="font-serif text-sm font-semibold text-civic-ink">{stage}</h3>
                <p className="mt-1 text-xs text-civic-stone">
                  {stageDescriptions[stage] ?? 'A stage in the Nigerian legislative process.'}
                </p>
              </div>
            </li>
          ))}
        </ol>
      </section>
    </div>
  );
}

const stageDescriptions: Record<string, string> = {
  'First Reading': 'The Bill is formally introduced and entered on the Order Paper.',
  'Second Reading': 'The chamber debates the principles and policy of the Bill.',
  'Public Hearing': 'The committee receives memoranda from the public and stakeholders.',
  Committee: 'The committee scrutinises the Bill clause-by-clause and proposes amendments.',
  Report: 'The committee reports back to the full chamber, which may further amend.',
  'Third Reading': 'Final reading and vote on the Bill in the originating chamber.',
  Concurrence: 'The other chamber considers and votes on the Bill as passed.',
  Assent: 'The President signs the Bill into law per Section 58 of the Constitution.',
  Commencement: 'The Act comes into force on a date fixed by the Act or by Gazette notice.',
};
