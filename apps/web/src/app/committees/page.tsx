import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Committees',
  description: 'Parliamentary committees of Kenya — recent activity, Bills under consideration, and committee reports.',
};

export default function CommitteesPage() {
  const committees = [
    { name: 'Departmental Committee on Lands', house: 'National Assembly', bills: 2 },
    { name: 'Standing Committee on Information, Communication and Technology', house: 'Senate', bills: 1 },
    { name: 'Departmental Committee on Finance and National Planning', house: 'National Assembly', bills: 1 },
    { name: 'Departmental Committee on Transport, Public Works and Housing', house: 'National Assembly', bills: 1 },
    { name: 'Standing Committee on Finance and Budget', house: 'Senate', bills: 1 },
    { name: 'Departmental Committee on Education and Research', house: 'National Assembly', bills: 1 },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Committees</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Parliamentary committees examine Bills in detail, conduct public
          participation, and produce reports. Each committee page shows
          recent activity, Bills under consideration, and published reports.
        </p>
      </header>

      <ul className="mt-6 grid gap-3 md:grid-cols-2">
        {committees.map((c) => (
          <li key={c.name} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
            <div className="text-xs text-civic-stone">{c.house}</div>
            <h2 className="mt-1 font-serif text-base font-semibold text-civic-ink">{c.name}</h2>
            <p className="mt-2 text-xs text-civic-stone">{c.bills} Bill(s) under consideration</p>
          </li>
        ))}
      </ul>

      <p className="mt-8 text-xs text-civic-stone">
        Full committee pages with Hansard, reports, and public participation
        records are tracked in <a href="https://github.com/Roy-Wanyoike/civic-intelligence/issues" className="text-civic-leaf hover:underline" target="_blank" rel="noopener noreferrer">issue #CI-WEB-008</a>.
      </p>
    </div>
  );
}
