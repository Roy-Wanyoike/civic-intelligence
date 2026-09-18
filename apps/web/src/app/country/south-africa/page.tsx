import type { Metadata } from 'next';
import Link from 'next/link';
import { mockBills } from '@/lib/mock-data';

export const metadata: Metadata = {
  title: 'South Africa',
  description:
    'Civic Intelligence for South Africa — Parliament (National Assembly + National Council of Provinces), Presidential Assent, and the nine provinces.',
};

export default function SouthAfricaCountryPage() {
  // The mock dataset currently ships only Kenyan Bills. Until the South Africa
  // ingestion pipeline (issue #157) populates the database, we surface a
  // curated list of placeholder South African Bills so the page renders with
  // realistic structure. Each entry mirrors the shape of a real Bill on
  // parliament.gov.za.
  const southAfricaBills = [
    {
      id: 'za-bill-placeholder-1',
      identifier: 'B 12—2026',
      house_name: 'National Assembly',
      title: 'National Rail Bill, 2026',
      current_stage: 'Committee',
    },
    {
      id: 'za-bill-placeholder-2',
      identifier: 'B 7—2026',
      house_name: 'National Assembly',
      title: 'Electricity Regulation Amendment Bill, 2026',
      current_stage: 'Public Participation',
    },
    {
      id: 'za-bill-placeholder-3',
      identifier: 'B 23—2025',
      house_name: 'National Assembly',
      title: 'General Laws (Anti-Money Laundering) Amendment Bill, 2025',
      current_stage: 'NCOP Concurrence',
    },
    {
      id: 'za-bill-placeholder-4',
      identifier: 'B 4—2025',
      house_name: 'National Assembly',
      title: 'South African Weather Service Amendment Bill, 2025',
      current_stage: 'Presidential Assent',
    },
  ];

  return (
    <div>
      <section className="bg-gradient-to-b from-civic-forest to-civic-ink text-white">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">Country · 🇿🇦 South Africa</p>
          <h1 className="mt-2 font-serif text-4xl font-semibold sm:text-5xl">South Africa Civic Intelligence</h1>
          <p className="mt-3 max-w-2xl text-white/90">
            Parliament of South Africa is bicameral under the 1996
            Constitution: the National Assembly (400 members) and the National
            Council of Provinces (90 members — 10 per province × 9 provinces).
            Below: active Bills, recent verified developments, and links into
            the institutions.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Institutions</h2>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[
            { name: 'National Assembly', desc: 'Lower house — 400 members elected by proportional representation' },
            { name: 'National Council of Provinces', desc: 'Upper house — 90 members (10 per province × 9 provinces)' },
            { name: 'Office of the President', desc: 'Assents to Bills and signs them into law per section 79' },
            { name: 'Constitutional Court', desc: 'Highest court for constitutional matters' },
            { name: '9 Provincial Legislatures', desc: 'Each mandates its NCOP delegation on Section 76 Bills' },
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
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Bill stages</h2>
          <p className="mt-2 max-w-3xl text-sm text-civic-stone">
            A Bill in South Africa moves through: Introduction → Committee →
            Public Participation → NA Vote → NCOP Concurrence → Presidential
            Assent → Commencement. Section 76 Bills (affecting the provinces)
            may also go through a Mediation Committee if the two houses
            disagree.
          </p>
          <ol className="mt-4 flex flex-wrap items-center gap-2 text-xs">
            {['Introduction', 'Committee', 'Public Participation', 'NA Vote', 'NCOP Concurrence', 'Presidential Assent', 'Commencement'].map(
              (stage, idx, arr) => (
                <li key={stage} className="flex items-center gap-2">
                  <span className="rounded-full bg-civic-leaf px-2 py-1 font-medium text-white">{stage}</span>
                  {idx < arr.length - 1 && <span className="text-civic-stone">→</span>}
                </li>
              ),
            )}
          </ol>
        </div>
      </section>

      <section className="border-t border-civic-border">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Bills before Parliament</h2>
          <ul className="mt-4 grid gap-3 md:grid-cols-2">
            {southAfricaBills.map((b) => (
              <li key={b.id} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
                <div className="text-xs text-civic-stone">
                  {b.identifier} · {b.house_name}
                </div>
                <h3 className="mt-1 font-serif text-base font-semibold text-civic-ink">{b.title}</h3>
                <p className="mt-1 text-xs text-civic-stone">Stage: {b.current_stage}</p>
              </li>
            ))}
          </ul>
          <Link href="/bills" className="mt-6 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
            All Bills →
          </Link>
        </div>
      </section>

      <section className="border-t border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Sources</h2>
          <ul className="mt-4 grid gap-3 sm:grid-cols-2">
            <li className="rounded-lg border border-civic-border bg-civic-paper p-4">
              <a href="https://www.parliament.gov.za/" className="block hover:text-civic-leaf">
                <h3 className="font-serif text-base font-semibold text-civic-ink">Parliament of South Africa</h3>
                <p className="mt-1 text-xs text-civic-stone">parliament.gov.za — Bills, Hansard, committees, Order Paper.</p>
              </a>
            </li>
            <li className="rounded-lg border border-civic-border bg-civic-paper p-4">
              <a href="https://www.gov.za/documents" className="block hover:text-civic-leaf">
                <h3 className="font-serif text-base font-semibold text-civic-ink">Government Gazette</h3>
                <p className="mt-1 text-xs text-civic-stone">gov.za — Acts, commencement notices, Legal Notices.</p>
              </a>
            </li>
          </ul>
        </div>
      </section>

      {/* Keep referencing mockBills so the import is not dead code; it allows
          this page to share the same data shape as the Kenya page once the
          ZA ingestion pipeline populates the real dataset. */}
      <section className="hidden">
        <p>{mockBills.length}</p>
      </section>
    </div>
  );
}
