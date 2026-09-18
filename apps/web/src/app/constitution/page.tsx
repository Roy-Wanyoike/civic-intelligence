import { Scale, ExternalLink, BookOpen } from 'lucide-react';
import type { Metadata } from 'next';
import { getConstitution, listConstitutionArticles, type ConstitutionChapter } from '@/lib/government-api';
import { constitutionArticles } from '@/data/constitution-articles';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Constitution of Kenya',
  description:
    'Explore the Constitution of Kenya, Article by Article. The Constitution is authoritative source material — the platform never reinterprets it.',
};

function formatDate(d?: string): string {
  if (!d) return '';
  return new Date(d).toLocaleDateString('en-KE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

// chaptersFromFallback transforms the hardcoded constitution-articles.ts data
// file into the same chapter+article tree shape the Go BFF returns from
// GET /api/v1/constitution/articles. Used only when the API is unreachable.
//
// The data file's chapter field is a display string like
// "Chapter 4 — The Bill of Rights"; this helper splits that into the
// chapter number (4) and title ("The Bill of Rights") so the fallback
// renders with the same structure as the API-driven path.
function chaptersFromFallback(): ConstitutionChapter[] {
  const byChapter = new Map<string, ConstitutionChapter>();
  let order = 0;
  for (const a of constitutionArticles) {
    // Split "Chapter N — Title" into number + title. The em-dash is U+2014.
    const match = a.chapter.match(/^Chapter\s+(\d+)\s+\u2014\s+(.+)$/);
    const number = match ? parseInt(match[1], 10) : ++order;
    const title = match ? match[2] : a.chapter;
    const id = `chapter-${number}`;
    if (!byChapter.has(id)) {
      byChapter.set(id, {
        id,
        constitution_id: 'constitution-ke-2010',
        number,
        title,
        articles: [],
      });
    }
    byChapter.get(id)!.articles.push({
      id: `article-${a.number.replace(/\s+/g, '-').toLowerCase()}`,
      chapter_id: id,
      number: a.number,
      title: a.title,
      text: a.text,
      source_url: a.source,
    });
  }
  return Array.from(byChapter.values()).sort((a, b) => a.number - b.number);
}

export default async function ConstitutionPage() {
  let constitution: Awaited<ReturnType<typeof getConstitution>>['constitution'] | null = null;
  try {
    const resp = await getConstitution();
    constitution = resp.constitution;
  } catch { /* ignore */ }

  // Fetch the curated chapter + article tree from the Go BFF (issue #212).
  // Fall back to the hardcoded data file when the API is unreachable so the
  // page degrades gracefully instead of rendering an empty list.
  let chapters: ConstitutionChapter[] = [];
  let usedFallback = false;
  try {
    const resp = await listConstitutionArticles();
    chapters = Array.isArray(resp.chapters) ? resp.chapters : [];
  } catch {
    usedFallback = true;
    chapters = chaptersFromFallback();
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Scale className="h-4 w-4" aria-hidden="true" />
          <span>Authoritative source material · Kenya</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          The Constitution of Kenya
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Explore the Constitution, Article by Article. Connected to legislation,
          institutions, and court decisions. The Constitution itself remains
          authoritative source material — a scenario must never modify or
          reinterpret constitutional text as though the modification were real.
        </p>
      </header>

      <div className="mb-8">
        <RealityDisclaimer kind="FACT" />
      </div>

      {constitution && (
        <section className="mb-8 rounded-lg border border-stone-200 bg-white p-5">
          <div className="flex items-center gap-2">
            <BookOpen className="h-5 w-5 text-civic-forest" aria-hidden="true" />
            <h2 className="font-serif text-lg font-semibold text-civic-forest">
              Version {constitution.version}
            </h2>
            <RealityBadge kind="FACT" />
          </div>
          <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Assented</dt>
              <dd className="text-stone-800">{formatDate(constitution.assented_at)}</dd>
            </div>
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Promulgated</dt>
              <dd className="text-stone-800">{formatDate(constitution.promulgated_at)}</dd>
            </div>
          </dl>
          <a
            href={constitution.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
          >
            Read on Kenya Law <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        </section>
      )}

      <section className="rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Articles
        </h2>
        <p className="mt-1 text-xs text-stone-600">
          A curated selection of constitutional articles, with the original
          text from Kenya Law. The full text of the Constitution is available
          on Kenya Law.
        </p>
        {usedFallback && (
          <p className="mt-2 rounded bg-amber-50 px-2 py-1 text-xs text-amber-900">
            The Constitution API is unreachable — showing the bundled
            fallback article set.
          </p>
        )}
        <div className="mt-4 space-y-6">
          {chapters.map((c) => (
            <div key={c.id}>
              <h3 className="border-b border-stone-200 pb-1 font-serif text-sm font-semibold text-civic-forest">
                Chapter {c.number} — {c.title}
              </h3>
              <ul className="mt-3 space-y-2">
                {c.articles.map((a) => (
                  <li
                    key={a.id}
                    className="rounded-lg border border-stone-200 bg-stone-50 p-3 transition hover:border-civic-forest/40"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <span className="text-xs font-semibold text-civic-stone">
                          Chapter {c.number} — {c.title}
                        </span>
                        <h4 className="mt-1 font-serif text-base font-semibold text-civic-forest">
                          {a.number}: {a.title}
                        </h4>
                      </div>
                      <RealityBadge kind="FACT" />
                    </div>
                    <p className="mt-2 line-clamp-3 text-sm text-stone-700">
                      {a.text}
                    </p>
                    <div className="mt-2 flex items-center gap-2 text-xs">
                      <a
                        href={a.source_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-sky-700 hover:underline"
                      >
                        Read full article <ExternalLink className="h-3 w-3" aria-hidden="true" />
                      </a>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5">
        <div className="flex items-center gap-2">
          <h2 className="font-serif text-lg font-semibold text-amber-900">
            Constitutional Context Engine
          </h2>
          <RealityBadge kind="LIMITATION" />
        </div>
        <p className="mt-1 text-sm text-amber-900">
          The platform distinguishes four kinds of constitutional references:
        </p>
        <ul className="mt-2 list-disc pl-5 text-sm text-amber-900">
          <li><strong>EXPLICIT</strong> — the Bill cites the Article directly.</li>
          <li><strong>INFERRED</strong> — the system identifies a possible relationship.</li>
          <li><strong>JUDICIAL</strong> — a court decision interpreted the relationship.</li>
          <li><strong>UNKNOWN</strong> — no sufficiently reliable relationship established.</li>
        </ul>
        <p className="mt-3 text-xs text-amber-800">
          The platform NEVER independently declares a Bill unconstitutional
          merely because an AI model detects a possible conflict. For legal
          conclusions, rely on authoritative legal sources.
        </p>
      </section>
    </div>
  );
}
