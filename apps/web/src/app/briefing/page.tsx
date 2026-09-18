import Link from 'next/link';
import { mockBriefing } from '@/lib/mock-data';
import { formatDate, confidenceClass } from '@/lib/utils';
import { ExternalLink } from 'lucide-react';
import { PrintButton } from '@/components/print-button';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Daily Civic Brief',
  description: 'A daily, evidence-grounded summary of verified civic developments in Kenya.',
};

export default function BriefingPage() {
  const briefing = mockBriefing;

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header className="no-print">
        <div className="flex items-center justify-between">
          <p className="text-sm uppercase tracking-widest text-civic-acacia">Kenya Civic Brief</p>
          <PrintButton />
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          {briefing.headline}
        </h1>
        <p className="mt-2 text-sm text-civic-stone">
          {formatDate(briefing.date)} · {briefing.items.length} verified item(s) · Generated {formatDate(briefing.generated_at)}
        </p>
      </header>

      <ol className="mt-8 space-y-6">
        {briefing.items.map((item, i) => (
          <li key={i} className="rounded-lg border border-civic-border bg-civic-paper p-5">
            <div className="flex items-center justify-between text-xs text-civic-stone">
              <span className="rounded-full bg-civic-mist px-2 py-0.5 font-medium uppercase">
                {item.kind.replace('_', ' ')}
              </span>
              <span className={`rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${confidenceClass(item.confidence)}`}>
                {item.confidence} confidence
              </span>
            </div>
            <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">{item.title}</h2>
            <p className="mt-2 text-sm text-civic-ink">{item.description}</p>
            <p className="mt-2 text-sm text-civic-stone">
              <span className="font-medium text-civic-ink">Why it matters:</span> {item.significance}
            </p>
            {item.citations.length > 0 && (
              <div className="mt-3 border-t border-civic-border pt-2">
                <p className="text-xs font-semibold text-civic-ink">Sources:</p>
                <ul className="mt-1 space-y-1">
                  {item.citations.map((c, j) => (
                    <li key={j} className="text-xs text-civic-stone">
                      <a href={c.source_url} target="_blank" rel="noopener noreferrer" className="text-civic-leaf hover:underline">
                        [{j + 1}] {c.source_type}
                        {c.page_number ? ` p.${c.page_number}` : ''}
                        {c.section ? ` §${c.section}` : ''}
                      </a>
                      <span className="italic"> &mdash; &ldquo;{c.snippet}&rdquo;</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {item.bill_id && (
              <Link
                href={`/bills/${item.bill_id}`}
                className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
              >
                View Bill <ExternalLink className="h-3 w-3" aria-hidden="true" />
              </Link>
            )}
          </li>
        ))}
      </ol>

      <p className="mt-8 text-xs text-civic-stone">
        The Civic Brief explains significance without political persuasion. Every
        item links to evidence. We do not rank politicians, parties, or
        positions. Information could not be verified from available
        authoritative sources is excluded.
      </p>
    </div>
  );
}
