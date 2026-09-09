import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Countries',
  description: 'Civic Intelligence is country-agnostic. Kenya is first. Uganda, Tanzania, Ghana, Nigeria, and South Africa are planned.',
};

export default function CountryPage() {
  const countries = [
    { code: 'KE', name: 'Kenya', active: true },
    { code: 'UG', name: 'Uganda', active: false },
    { code: 'TZ', name: 'Tanzania', active: false },
    { code: 'GH', name: 'Ghana', active: false },
    { code: 'NG', name: 'Nigeria', active: false },
    { code: 'ZA', name: 'South Africa', active: false },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Countries</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Civic Intelligence is built country-agnostic. Each country is a
          plugin — an adapter that knows that country&apos;s institutions,
          legislative stages, terminology, and document sources. The first
          country is Kenya.
        </p>
      </header>

      <ul className="mt-6 grid gap-3 md:grid-cols-2 lg:grid-cols-3">
        {countries.map((c) => (
          <li
            key={c.code}
            className={`rounded-lg border p-4 ${c.active ? 'border-civic-leaf bg-civic-paper' : 'border-civic-border bg-civic-mist/30 opacity-60'}`}
          >
            <div className="flex items-center justify-between">
              <span className="font-serif text-base font-semibold text-civic-ink">{c.name}</span>
              <span className="text-xs text-civic-stone">{c.code}</span>
            </div>
            <p className="mt-2 text-xs text-civic-stone">
              {c.active ? 'Active — adapter available' : 'Planned — adapter not yet built'}
            </p>
            {c.active && (
              <Link href={`/country/${c.code.toLowerCase()}`} className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
                Open {c.name} →
              </Link>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
