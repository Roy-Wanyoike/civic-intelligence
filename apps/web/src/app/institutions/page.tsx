import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Institutions',
  description: 'Kenyan civic institutions — Parliament, Senate, National Assembly, constitutional commissions, and counties.',
};

export default function InstitutionsPage() {
  const institutions = [
    { name: 'Parliament of Kenya', type: 'Legislature', houses: 2 },
    { name: 'National Assembly', type: 'House', houses: 0 },
    { name: 'Senate', type: 'House', houses: 0 },
    { name: 'Office of the Attorney General', type: 'Constitutional Office', houses: 0 },
    { name: 'Judiciary', type: 'Judiciary', houses: 0 },
    { name: 'Independent Electoral and Boundaries Commission', type: 'Constitutional Commission', houses: 0 },
    { name: 'Ethics and Anti-Corruption Commission', type: 'Constitutional Commission', houses: 0 },
    { name: 'County Government of Nairobi', type: 'County', houses: 1 },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Institutions</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Civic institutions — legislatures, executive, judiciary, ministries,
          agencies, regulators, counties, constitutional commissions, and
          independent offices. Each institution page shows recent activity,
          Bills, committees, documents, and events.
        </p>
      </header>

      <ul className="mt-6 grid gap-3 md:grid-cols-2">
        {institutions.map((i) => (
          <li key={i.name} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
            <div className="text-xs text-civic-stone">{i.type}</div>
            <h2 className="mt-1 font-serif text-base font-semibold text-civic-ink">{i.name}</h2>
            {i.houses > 0 && (<p className="mt-2 text-xs text-civic-stone">{i.houses} House(s)</p>)}
          </li>
        ))}
      </ul>

      <div className="mt-8">
        <Link href="/country/kenya" className="text-sm text-civic-leaf hover:underline">
          View Kenya country page →
        </Link>
      </div>
    </div>
  );
}
