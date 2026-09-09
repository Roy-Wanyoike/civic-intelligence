import type { Metadata } from 'next';
import Link from 'next/link';
import { mockBills } from '@/lib/mock-data';

export const metadata: Metadata = {
  title: 'Kenya',
  description: 'Civic Intelligence for Kenya — Parliament, Senate, National Assembly, constitutional commissions, and 47 counties.',
};

export default function KenyaCountryPage() {
  const kenyaBills = mockBills.filter((b) => b.country === 'KE');

  return (
    <div>
      <section className="bg-gradient-to-b from-civic-forest to-civic-ink text-white">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">Country · 🇰🇪 Kenya</p>
          <h1 className="mt-2 font-serif text-4xl font-semibold sm:text-5xl">Kenya Civic Intelligence</h1>
          <p className="mt-3 max-w-2xl text-white/90">
            Parliament of Kenya is bicameral under the 2010 Constitution: the
            National Assembly and the Senate. Below: active Bills, recent
            verified developments, and links into the institutions.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Institutions</h2>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[
            { name: 'National Assembly', desc: 'Lower house — 290 elected + 50 women + 12 nominated' },
            { name: 'Senate', desc: 'Upper house — 47 elected + 16 women + 2 youth + 2 PWDs' },
            { name: 'Office of the Attorney General', desc: 'Chief legal advisor to the government' },
            { name: 'Judiciary', desc: 'Supreme Court, Court of Appeal, High Court' },
            { name: '47 County Governments', desc: 'Devolved governments under the 2010 Constitution' },
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
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Bills before Parliament</h2>
          <ul className="mt-4 grid gap-3 md:grid-cols-2">
            {kenyaBills.slice(0, 6).map((b) => (
              <li key={b.id} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
                <Link href={`/bills/${b.id}`} className="block">
                  <div className="text-xs text-civic-stone">{b.identifier} · {b.house_name}</div>
                  <h3 className="mt-1 font-serif text-base font-semibold text-civic-ink">{b.title}</h3>
                  <p className="mt-1 text-xs text-civic-stone">Stage: {b.current_stage ?? '—'}</p>
                </Link>
              </li>
            ))}
          </ul>
          <Link href="/bills" className="mt-6 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
            All Bills →
          </Link>
        </div>
      </section>
    </div>
  );
}
