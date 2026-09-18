import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Ghana',
  description:
    'Civic Intelligence for Ghana — Parliament of Ghana (unicameral, 275 MPs), Bills, hansard, and constitutional institutions.',
};

export default function GhanaCountryPage() {
  const institutions = [
    { name: 'Parliament of Ghana', desc: 'Unicameral legislature — 275 elected MPs serving four-year terms' },
    { name: 'Office of the President', desc: 'Head of state and government; grants assent to Bills' },
    { name: 'Attorney-General’s Department', desc: 'Chief legal advisor to the Government and principal draftsman' },
    { name: 'Judiciary of Ghana', desc: 'Supreme Court, Court of Appeal, High Court, and lower courts' },
    { name: 'Electoral Commission', desc: 'Manages public elections and the boundaries of constituencies' },
  ];

  const stages = [
    { name: 'First Reading', desc: 'The Bill is read for the first time. No debate yet.' },
    { name: 'Second Reading', desc: 'MPs debate the principles and policy of the Bill.' },
    { name: 'Consideration Stage', desc: 'Committee of the Whole examines the Bill clause-by-clause.' },
    { name: 'Third Reading', desc: 'Final debate and vote on whether to pass the Bill.' },
    { name: 'Assent', desc: 'The President assents to the Bill, enacting it into law.' },
    { name: 'Commencement', desc: 'The Act comes into force.' },
  ];

  return (
    <div>
      <section className="bg-gradient-to-b from-civic-forest to-civic-ink text-white">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">Country · 🇬🇭 Ghana</p>
          <h1 className="mt-2 font-serif text-4xl font-semibold sm:text-5xl">Ghana Civic Intelligence</h1>
          <p className="mt-3 max-w-2xl text-white/90">
            The Parliament of Ghana is unicameral under the 1992 Constitution:
            275 Members, each elected from a single-member constituency, serving
            a four-year term. Below: institutions, the Ghana Bill lifecycle, and
            the country adapter status.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Institutions</h2>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {institutions.map((i) => (
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
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Bill Lifecycle</h2>
          <p className="mt-2 text-sm text-civic-stone">
            Ghana’s Bill stages differ from the Commonwealth norm: the
            Consideration Stage replaces the separate Committee Stage, taken in
            Committee of the Whole.
          </p>
          <ol className="mt-4 grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {stages.map((s, i) => (
              <li
                key={s.name}
                className="rounded-lg border border-civic-border bg-civic-paper p-4"
              >
                <div className="flex items-center gap-2">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-civic-leaf text-xs font-semibold text-white">
                    {i + 1}
                  </span>
                  <h3 className="font-serif text-base font-semibold text-civic-ink">{s.name}</h3>
                </div>
                <p className="mt-2 text-xs text-civic-stone">{s.desc}</p>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Adapter Status</h2>
        <div className="mt-4 rounded-lg border border-civic-border bg-civic-paper p-4">
          <div className="flex items-center justify-between">
            <span className="font-serif text-base font-semibold text-civic-ink">
              adapters/ghana
            </span>
            <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 text-xs font-medium text-civic-leaf">
              Ingestion skeleton
            </span>
          </div>
          <p className="mt-2 text-xs text-civic-stone">
            The Ghana country adapter implements <code>contracts.LegislativeSourceAdapter</code>.
            Discovery, fetch, and parse for <code>parliament.gh</code> are stubbed
            pending the upstream Bills listing.
          </p>
          <Link href="/country" className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
            ← All countries
          </Link>
        </div>
      </section>
    </div>
  );
}
