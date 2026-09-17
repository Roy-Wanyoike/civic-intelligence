import Link from 'next/link';
import { ArrowRight, FileText, Search, Sparkles, BookOpen, ShieldCheck, TrendingUp } from 'lucide-react';
import { mockBills } from '@/lib/mock-data';
import { TrendingCarousel } from '@/components/trending-carousel';
import { ConstitutionSpotlight } from '@/components/constitution-spotlight';

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
        <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6 sm:py-28">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">
            Kenya · 🇰🇪
          </p>
          <h1 className="mt-4 max-w-3xl font-serif text-4xl font-semibold leading-tight sm:text-5xl md:text-6xl">
            Understand what your government is doing.
          </h1>
          <p className="mt-5 max-w-2xl text-lg text-civic-mist/90">
            Evidence-grounded explanations of Kenyan legislation, regulations, and civic activity.
            No legal jargon. No political bias. Just the facts — with sources you can verify.
          </p>

          <form action="/ask" className="mt-10 flex max-w-2xl flex-col gap-3 sm:flex-row">
            <label htmlFor="q" className="sr-only">
              Search a Bill, law, topic, or ask a question
            </label>
            <input
              id="q"
              type="search"
              name="q"
              placeholder="Ask Kenya anything…"
              className="flex-1 rounded-lg border border-white/20 bg-white/10 px-5 py-4 text-base text-white placeholder:text-white/60 focus:border-civic-acacia focus:bg-white/15"
              autoComplete="off"
            />
            <button
              type="submit"
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-civic-acacia px-6 py-4 font-semibold text-civic-ink hover:bg-civic-acacia/90"
            >
              <Search className="h-5 w-5" aria-hidden="true" />
              Ask
            </button>
          </form>

          <div className="mt-8 flex flex-wrap gap-3 text-sm">
            <span className="text-white/70">Try:</span>
            {[
              'What is happening with housing?',
              'What did the President sign today?',
              'What changed in data protection?',
              'Track government loans',
            ].map((q) => (
              <Link
                key={q}
                href={`/ask`}
                className="rounded-full border border-white/20 px-4 py-1.5 text-white/90 transition hover:border-civic-acacia hover:text-white"
              >
                {q}
              </Link>
            ))}
          </div>

          {/* Constitution Spotlight — random constitutional clause */}
          <ConstitutionSpotlight />
        </div>
      </section>

      {/* Principles strip */}
      <section className="border-b border-civic-border bg-civic-paper">
        <div className="mx-auto grid max-w-7xl gap-8 px-4 py-12 sm:grid-cols-2 sm:px-6 lg:grid-cols-4">
          {[
            { icon: ShieldCheck, title: 'Evidence before AI', text: 'Every claim links to an official source.' },
            { icon: BookOpen, title: 'Plain language', text: 'No legal or parliamentary jargon required.' },
            { icon: Sparkles, title: 'AI explains, never decides', text: 'We show what happened. You decide what to think.' },
            { icon: FileText, title: 'Versioned forever', text: 'Legislative history is preserved — never overwritten.' },
          ].map(({ icon: Icon, title, text }) => (
            <div key={title} className="flex gap-4">
              <Icon className="h-6 w-6 flex-shrink-0 text-civic-leaf" aria-hidden="true" />
              <div>
                <h3 className="text-sm font-semibold text-civic-ink">{title}</h3>
                <p className="mt-1 text-sm text-civic-stone">{text}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* What's happening now */}
      <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6">
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
            href="/feed"
            className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
          >
            See full feed
            <ArrowRight className="h-4 w-4" aria-hidden="true" />
          </Link>
        </div>

        <ul className="mt-8 grid gap-4 md:grid-cols-2">
          {recentlyChanged.map((b) => (
            <li
              key={b.id}
              className="rounded-xl border border-civic-border bg-civic-paper p-6 transition hover:border-civic-leaf hover:shadow-sm"
            >
              <Link href={`/bills/${b.id}`} className="block">
                <div className="flex items-center justify-between gap-2 text-xs text-civic-stone">
                  <span>{b.identifier} · {b.year}</span>
                  <span className="rounded-full bg-civic-mist px-2.5 py-0.5 font-medium text-civic-ink">
                    {b.status.replace('_', ' ')}
                  </span>
                </div>
                <h3 className="mt-3 font-serif text-lg font-semibold text-civic-ink">
                  {b.title}
                </h3>
                {b.current_stage_simple_explanation && (
                  <p className="mt-2 text-sm text-civic-stone">
                    {b.current_stage_simple_explanation}
                  </p>
                )}
              </Link>
            </li>
          ))}
        </ul>
      </section>

      {/* Bills to watch */}
      <section className="border-t border-civic-border bg-civic-mist">
        <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6">
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
          <ul className="mt-8 grid gap-4 md:grid-cols-2">
            {billsToWatch.map((b) => (
              <li
                key={b.id}
                className="rounded-xl border border-civic-border bg-civic-paper p-6 transition hover:border-civic-leaf hover:shadow-sm"
              >
                <Link href={`/bills/${b.id}`} className="block">
                  <div className="flex items-center justify-between gap-2 text-xs text-civic-stone">
                    <span>{b.house_name ?? 'Parliament'}</span>
                    <span className="rounded-full bg-civic-acacia/20 px-2.5 py-0.5 font-medium text-civic-ink">
                      {b.current_stage ?? 'In progress'}
                    </span>
                  </div>
                  <h3 className="mt-3 font-serif text-lg font-semibold text-civic-ink">
                    {b.title}
                  </h3>
                  {b.purpose && (
                    <p className="mt-2 line-clamp-2 text-sm text-civic-stone">{b.purpose}</p>
                  )}
                  {b.topics && b.topics.length > 0 && (
                    <div className="mt-4 flex flex-wrap gap-1.5">
                      {b.topics.slice(0, 3).map((t) => (
                        <span
                          key={t}
                          className="rounded-full bg-civic-leaf/10 px-2.5 py-0.5 text-xs text-civic-leaf"
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

      {/* Trending Carousel + CTA */}
      <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6">
        <div className="grid gap-8 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <div className="flex items-center gap-2 mb-4">
              <TrendingUp className="h-6 w-6 text-civic-leaf" aria-hidden="true" />
              <h2 className="font-serif text-2xl font-semibold text-civic-forest">
                Trending This Month
              </h2>
            </div>
            <TrendingCarousel bills={billsToWatch.map(b => ({
              id: b.id,
              title: b.title,
              house: b.house_name ?? 'Parliament',
              year: b.year,
              source_url: b.id,
              current_stage: b.current_stage ?? undefined,
            }))} />
            <Link
              href="/trending"
              className="mt-6 inline-flex items-center gap-2 rounded-lg border border-civic-border bg-civic-paper px-5 py-3 text-sm font-semibold text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf"
            >
              View trending bills
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>

          <div className="rounded-2xl bg-civic-forest px-6 py-8 text-center text-white">
            <h2 className="font-serif text-2xl font-semibold">
              Don&apos;t have time to read every Bill?
            </h2>
            <p className="mx-auto mt-3 max-w-xs text-sm text-white/90">
              Get the Kenya Civic Brief — a daily summary of verified developments.
            </p>
            <Link
              href="/briefing"
              className="mt-6 inline-flex items-center gap-2 rounded-lg bg-civic-acacia px-6 py-3 font-semibold text-civic-ink hover:bg-civic-acacia/90"
            >
              Read today&apos;s brief
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
