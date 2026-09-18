import Link from 'next/link';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Countries',
  description: 'Civic Intelligence across Africa.',
};

const countries = [
  { code: 'KE', name: 'Kenya', active: true, flag: '🇰🇪' },
  { code: 'UG', name: 'Uganda', active: false, flag: '🇺🇬' },
  { code: 'TZ', name: 'Tanzania', active: false, flag: '🇹🇿' },
  { code: 'GH', name: 'Ghana', active: false, flag: '🇬🇭' },
  { code: 'NG', name: 'Nigeria', active: false, flag: '🇳🇬' },
  { code: 'ZA', name: 'South Africa', active: false, flag: '🇿🇦' },
];

export default function CountriesPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Countries</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Civic Intelligence is built country-by-country. Kenya is first — more African countries coming.
      </p>
      <ul className="mt-6 grid gap-3 md:grid-cols-2 lg:grid-cols-3">
        {countries.map(c => (
          <li key={c.code} className={`rounded-lg border p-4 ${c.active ? 'border-civic-leaf bg-civic-paper' : 'border-civic-border bg-civic-mist/30 opacity-60'}`}>
            {c.active ? (
              <Link href={`/country/${c.code.toLowerCase()}`} className="block">
                <div className="text-2xl">{c.flag}</div>
                <h2 className="mt-2 font-serif text-base font-semibold text-civic-ink">{c.name}</h2>
                <p className="text-xs text-civic-leaf">Active — adapter available</p>
              </Link>
            ) : (
              <div>
                <div className="text-2xl">{c.flag}</div>
                <h2 className="mt-2 font-serif text-base font-semibold text-civic-ink">{c.name}</h2>
                <p className="text-xs text-civic-stone">Planned — adapter not yet built</p>
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
