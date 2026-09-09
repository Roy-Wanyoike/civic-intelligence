import type { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Topics',
  description: 'Browse Bills, Acts, and civic activity by topic — housing, health, education, finance, and more.',
};

export default function TopicsPage() {
  const topics = [
    { slug: 'housing', count: 1 },
    { slug: 'data-protection', count: 1 },
    { slug: 'public-finance', count: 2 },
    { slug: 'education', count: 1 },
    { slug: 'transport', count: 1 },
    { slug: 'county-government', count: 1 },
    { slug: 'ict', count: 1 },
    { slug: 'taxation', count: 1 },
    { slug: 'security', count: 1 },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Topics</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Browse civic activity by topic. Each topic page aggregates Bills,
          Acts, regulations, Hansard references, and committee reports touching
          that subject.
        </p>
      </header>

      <ul className="mt-6 flex flex-wrap gap-2">
        {topics.map((t) => (
          <li key={t.slug}>
            <Link
              href={`/search?q=${encodeURIComponent(t.slug.replace('-', ' '))}`}
              className="inline-flex items-center gap-2 rounded-full bg-civic-mist px-4 py-2 text-sm text-civic-ink hover:bg-civic-leaf/10 hover:text-civic-leaf"
            >
              {t.slug.replace('-', ' ')}
              <span className="rounded-full bg-civic-paper px-1.5 text-xs text-civic-stone">{t.count}</span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
