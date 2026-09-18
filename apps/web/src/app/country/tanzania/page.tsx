import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Tanzania',
  description:
    'Civic Intelligence for Tanzania — Bunge la Tanzania (unicameral Parliament), 393 members, and the official parliament.go.tz source adapter.',
};

export default function TanzaniaCountryPage() {
  // The Bunge is unicameral: 266 elected constituency members + 117 special
  // seats (women's reserved seats plus Zanzibar and presidential appointees),
  // for 393 members total. Source: https://www.parliament.go.tz
  const bungeMembers = 393;
  const electedMembers = 266;
  const specialSeats = 117;

  const stages = [
    { name: 'First Reading', explanation: 'The Bill is read for the first time in the Bunge. No debate yet.' },
    { name: 'Second Reading', explanation: 'Members debate the principles and policy of the Bill.' },
    { name: 'Committee', explanation: 'A standing or sectoral committee examines the Bill clause-by-clause.' },
    { name: 'Report', explanation: 'The committee reports back to the Bunge; further amendments may be proposed.' },
    { name: 'Third Reading', explanation: 'Final debate and vote on whether to pass the Bill.' },
    { name: 'Assent', explanation: 'The President signs the Bill into law. May refer it back once.' },
    { name: 'Commencement', explanation: 'The Act comes into force, usually upon gazettement.' },
  ];

  return (
    <div>
      <section className="bg-gradient-to-b from-civic-forest to-civic-ink text-white">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">Country · 🇹🇿 Tanzania</p>
          <h1 className="mt-2 font-serif text-4xl font-semibold sm:text-5xl">Tanzania Civic Intelligence</h1>
          <p className="mt-3 max-w-2xl text-white/90">
            The Parliament of Tanzania — <em>Bunge la Tanzania</em> — is
            unicameral. It comprises {bungeMembers} members: {electedMembers}{' '}
            elected from constituencies and {specialSeats} special seats
            (including women&apos;s reserved seats and Zanzibar representatives).
            The country adapter below sources from{' '}
            <a
              href="https://www.parliament.go.tz"
              className="underline decoration-civic-acacia/60 underline-offset-4 hover:text-civic-acacia"
              target="_blank"
              rel="noopener noreferrer"
            >
              parliament.go.tz
            </a>
            .
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
        <h2 className="font-serif text-2xl font-semibold text-civic-forest">Institutions</h2>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[
            { name: 'Bunge la Tanzania', desc: `Unicameral Parliament — ${bungeMembers} members (${electedMembers} elected + ${specialSeats} special seats)` },
            { name: 'Office of the Attorney General', desc: 'Chief legal advisor to the United Republic; may attend and speak in the Bunge ex officio.' },
            { name: 'Office of the President', desc: 'Assents to Bills passed by the Bunge; may refer a Bill back once.' },
            { name: 'Government Gazette', desc: 'Official publication of Acts, statutory instruments, and notices of the United Republic.' },
            { name: 'Zanzibar House of Representatives', desc: 'Semi-autonomous legislature for Zanzibar; five of its members also sit in the Union Bunge.' },
            { name: 'Judiciary of Tanzania', desc: 'Court of Appeal, High Court, and Resident Magistrate Courts.' },
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
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Bill lifecycle in the Bunge</h2>
          <p className="mt-2 max-w-2xl text-sm text-civic-stone">
            Tanzania&apos;s Bill stages run from First Reading through to
            Commencement, in line with the Standing Orders of the National
            Assembly. A Bill may also be Rejected at a vote.
          </p>
          <ol className="mt-4 grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {stages.map((s, idx) => (
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
          <h2 className="font-serif text-2xl font-semibold text-civic-forest">Bills before the Bunge</h2>
          <p className="mt-2 max-w-2xl text-sm text-civic-stone">
            The Tanzania adapter is live but the parliament.go.tz parser is
            still being developed. Once ingestion completes, this panel will
            show the Bills currently before the Bunge. For now, browse the
            source directly.
          </p>
          <div className="mt-4 flex flex-wrap gap-3">
            <a
              href="https://www.parliament.go.tz"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 rounded-md border border-civic-leaf bg-civic-leaf/10 px-3 py-2 text-sm text-civic-leaf hover:bg-civic-leaf/20"
            >
              Open parliament.go.tz →
            </a>
            <Link
              href="/bills"
              className="inline-flex items-center gap-1 rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm text-civic-ink hover:border-civic-leaf"
            >
              All Bills →
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
