import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'People',
  description: 'Members of Parliament, Senators, and committee members — connected to the Bills they sponsor and the proceedings they participate in.',
};

export default function PeoplePage() {
  const people = [
    { name: 'Cabinet Secretary, Lands', role: 'Sponsor — Housing Bill', bills: 1 },
    { name: 'Senator, Committee on ICT', role: 'Sponsor — Data Protection (Amendment) Bill', bills: 1 },
    { name: 'Cabinet Secretary, National Treasury', role: 'Sponsor — Public Finance Management (Amendment) Bill', bills: 1 },
    { name: 'Member of Parliament', role: 'Sponsor — Traffic (Amendment) Bill', bills: 1 },
    { name: 'Senator', role: 'Sponsor — County Government (Amendment) Bill', bills: 1 },
    { name: 'Cabinet Secretary, Education', role: 'Sponsor — Education (Amendment) Bill', bills: 1 },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">People</h1>
        <p className="mt-2 text-sm text-civic-stone">
          MPs, Senators, and committee members connected to the Bills they
          sponsor, the speeches they make, and the committees they serve on.
          We do not rank politicians — we connect them to verifiable civic activity.
        </p>
      </header>

      <ul className="mt-6 grid gap-3 md:grid-cols-2">
        {people.map((p) => (
          <li key={p.name} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
            <h2 className="font-serif text-base font-semibold text-civic-ink">{p.name}</h2>
            <p className="mt-1 text-xs text-civic-stone">{p.role}</p>
            <p className="mt-2 text-xs text-civic-stone">{p.bills} Bill(s) sponsored</p>
          </li>
        ))}
      </ul>

      <p className="mt-8 text-xs text-civic-stone">
        Full people pages (with speeches, votes, committee membership) are tracked in <a href="https://github.com/Roy-Wanyoike/civic-intelligence/issues" className="text-civic-leaf hover:underline" target="_blank" rel="noopener noreferrer">issue #CI-WEB-009</a>.
      </p>
    </div>
  );
}
