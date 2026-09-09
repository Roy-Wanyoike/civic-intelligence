import Link from 'next/link';
import { ArrowRight, FileText, Search, Sparkles, BookOpen, ShieldCheck } from 'lucide-react';
import { mockBills } from '@/lib/mock-data';

export default function HomePage() {
  const billsToWatch = mockBills.filter(b => b.status !== 'enacted').slice(0, 4);
  const recentlyChanged = mockBills
    .slice()
    .sort((a, b) => +new Date(b.updated_at) - +new Date(a.updated_at))
    .slice(0, 4);

  return (
    <div>
      {/* Hero */}
      <section className="bg-gradient-to-b from-civic-forest to-civic-ink text-white">
        <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 sm:py-24">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">
            Kenya · 🇰🇪
          </p>
          <h1 className="mt-4 max-w-3xl font-serif text-4xl font-semibold leading-tight sm:text-5xl md:text-6xl">
            Understand what your government is doing.
          </h1>
          <p className="mt-4 max-w-2xl text-lg text-civic-mist/90">
            No legal or parliamentary jargon required. We translate Bills,
            regulations, and civic activity into plain language — with every
            claim traceable to an official source.
          </p>

          <form action="/search" className="mt-8 flex max-w-2xl flex-col gap-3 sm:flex-row">
            <label htmlFor="q" className="sr-only">
              Search a Bill, law, topic, or ask a question
            </label>
            <input
              id="q"
              type="search"
              name="q"
              placeholder="Search a Bill, law, topic, or ask a question"
              className="flex-1 rounded-md border border-white/20 bg-white/10 px-4 py-3 text-white placeholder:text-white/60 focus:border-civic-acacia focus:bg-white/15"
              autoComplete="off"
            />
            <button
              type="submit"
              className="inline-flex items-center justify-center gap-1 rounded-md bg-civic-acacia px-5 py-3 font-semibold text-civic-ink hover:bg-civic-acacia/90"
            >
              <Search className="h-4 w-4" aria-hidden="true" />
              Search
            </button>
          </form>

          <div className="mt-6 flex flex-wrap gap-3 text-sm">
            <span className="text-white/70">Try:</span>
            {[
              'Bills about housing',
              'What is happening in Parliament this week?',
              'What changed in data protection?',
            ].map((q) => (
              <Link
                key={q}
                href={`/search?q=${encodeURIComponent(q)}`}
                className="rounded-full border border-white/20 px-3 py-1 text-white/90 hover:border-civic-acacia hover:text-white"
              >
                {q}
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Principles strip */}
      <section className="border-b border-civic-border bg-civic-paper">
        <div className="mx-auto grid max-w-7xl gap-6 px-4 py-10 sm:grid-cols-2 sm:px-6 lg:grid-cols-4">
          {[
            { icon: ShieldCheck, title: 'Evidence before AI', text: 'Every claim links to an official source.' },
            { icon: BookOpen, title: 'Plain language', text: 'No legal or parliamentary jargon required.' },
            { icon: Sparkles, title: 'AI explains, never decides', text: 'We show what happened. You decide what to think.' },
            { icon: FileText, title: 'Versioned forever', text: 'Legislative history is preserved — never overwritten.' },
          ].map(({ icon: Icon, title, text }) => (
            <div key={title} className="flex gap-3">
              <Icon className="h-5 w-5 flex-shrink-0 text-civic-leaf" aria-hidden="true" />
              <div>
                <h3 className="text-sm font-semibold text-civic-ink">{title}</h3>
                <p className="mt-1 text-sm text-civic-stone">{text}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* What's happening now */}
      <section className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
        <div className="flex items-end justify-between">
          <div>
            <h2 className="font-serif text-2xl font-semibold text-civic-forest">
              What&apos;s happening now
            </h2>
            <p className="mt-1 text-sm text-civic-stone">
              Verified developments in Kenyan Parliament and government.
            </p>
          </div>
          <Link
            href="/briefing"
            className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
          >
            See full daily brief
            <ArrowRight className="h-4 w-4" aria-hidden="true" />
          </Link>
        </div>

        <ul className="mt-6 grid gap-4 md:grid-cols-2">
          {recentlyChanged.map((b) => (
            <li
              key={b.id}
              className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
            >
              <Link href={`/bills/${b.id}`} className="block">
                <div className="flex items-center justify-between gap-2 text-xs text-civic-stone">
                  <span>{b.identifier} · {b.year}</span>
                  <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium text-civic-ink">
                    {b.status}
                  </span>
                </div>
                <h3 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                  {b.title}
                </h3>
                {b.current_stage_simple_explanation && (
                  <p className="mt-2 text-sm text-civic-stone">
                    Current stage: {b.current_stage_simple_explanation}
                  </p>
                )}
              </Link>
            </li>
          ))}
        </ul>
      </section>

      {/* Bills to watch */}
      <section className="border-t border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
          <div className="flex items-end justify-between">
            <div>
              <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                Bills to watch
              </h2>
              <p className="mt-1 text-sm text-civic-stone">
                Active Bills currently before Parliament.
              </p>
            </div>
            <Link
              href="/bills"
              className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
            >
              All Bills
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>
          <ul className="mt-6 grid gap-4 md:grid-cols-2">
            {billsToWatch.map((b) => (
              <li
                key={b.id}
                className="rounded-lg border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf"
              >
                <Link href={`/bills/${b.id}`} className="block">
                  <div className="flex items-center justify-between gap-2 text-xs text-civic-stone">
                    <span>{b.house_name ?? 'Parliament'}</span>
                    <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium text-civic-ink">
                      {b.current_stage ?? 'In progress'}
                    </span>
                  </div>
                  <h3 className="mt-2 font-serif text-lg font-semibold text-civic-ink">
                    {b.title}
                  </h3>
                  {b.purpose && (
                    <p className="mt-2 line-clamp-2 text-sm text-civic-stone">
                      {b.purpose}
                    </p>
                  )}
                  {b.topics && b.topics.length > 0 && (
                    <div className="mt-3 flex flex-wrap gap-1.5">
                      {b.topics.slice(0, 3).map((t) => (
                        <span
                          key={t}
                          className="rounded-full bg-civic-leaf/10 px-2 py-0.5 text-xs text-civic-leaf"
                        >
                          {t}
                        </span>
                      ))}
                    </div>
                  )}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </section>

      {/* CTA */}
      <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6">
        <div className="rounded-2xl bg-civic-forest px-8 py-12 text-center text-white">
          <h2 className="font-serif text-3xl font-semibold">
            Don&apos;t have time to read every Bill?
          </h2>
          <p className="mx-auto mt-3 max-w-xl text-white/90">
            Get the Kenya Civic Brief — a daily summary of verified
            developments, every claim traceable to an official source.
          </p>
          <Link
            href="/briefing"
            className="mt-6 inline-flex items-center gap-2 rounded-md bg-civic-acacia px-6 py-3 font-semibold text-civic-ink hover:bg-civic-acacia/90"
          >
            Read today&apos;s brief
            <ArrowRight className="h-4 w-4" aria-hidden="true" />
          </Link>
        </div>
      </section>
    </div>
  );
}
