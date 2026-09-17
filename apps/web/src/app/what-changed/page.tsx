import Link from 'next/link';
import { TrendingUp, ExternalLink, ShieldCheck } from 'lucide-react';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'What Changed',
  description: 'Verified civic changes — what happened recently in Kenyan government.',
};

interface ChangeItem {
  kind: string;
  title: string;
  description: string;
  house: string;
  date: string;
  source_url: string;
  significance: string;
  verification: string;
  evidence_url: string;
  country: string;
}

interface WhatChangedData {
  items: ChangeItem[];
  total: number;
  country: string;
  source: string;
  generated_at: string;
}

async function fetchWhatChanged(): Promise<WhatChangedData | null> {
  try {
    const resp = await fetch('http://localhost:9000/api/v1/what-changed', {
      next: { revalidate: 300 },
    });
    if (!resp.ok) return null;
    return resp.json();
  } catch {
    return null;
  }
}

const SIGNIFICANCE_COLORS: Record<string, string> = {
  INFORMATIONAL: 'bg-civic-mist text-civic-stone',
  MINOR: 'bg-civic-mist text-civic-stone',
  PROCEDURAL: 'bg-civic-acacia/20 text-civic-ink',
  SUBSTANTIVE: 'bg-civic-acacia/30 text-civic-clay',
  HIGH_IMPACT: 'bg-civic-clay/20 text-civic-clay',
  CRITICAL: 'bg-red-100 text-red-800',
};

export default async function WhatChangedPage() {
  const data = await fetchWhatChanged();

  if (!data) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">What Changed</h1>
        <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
          Change data is temporarily unavailable.
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2">
          <TrendingUp className="h-6 w-6 text-civic-leaf" aria-hidden="true" />
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">What Changed</h1>
        </div>
        <p className="mt-2 text-sm text-civic-stone">
          Verified civic changes from official sources. Every item links to evidence.
          Source: {data.source} · Updated: {data.generated_at}
        </p>
      </header>

      <ul className="space-y-4">
        {data.items.map((item, i) => (
          <li
            key={i}
            className="rounded-xl border border-civic-border bg-civic-paper p-5 hover:border-civic-leaf hover:shadow-sm transition"
          >
            <div className="flex items-center justify-between gap-2 text-xs">
              <span className="rounded-full bg-civic-leaf/10 px-2.5 py-0.5 font-medium text-civic-leaf">
                {item.kind.replace(/_/g, ' ')}
              </span>
              <div className="flex items-center gap-2">
                <span className={`rounded-full px-2.5 py-0.5 font-medium ${SIGNIFICANCE_COLORS[item.significance] || SIGNIFICANCE_COLORS.INFORMATIONAL}`}>
                  {item.significance}
                </span>
                <span className="flex items-center gap-0.5 text-civic-leaf">
                  <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                  {item.verification}
                </span>
              </div>
            </div>

            <h2 className="mt-3 font-serif text-lg font-semibold text-civic-ink">
              <Link href={`/bills/${item.source_url.split('/').pop()?.split('/')[0] || ''}`} className="hover:text-civic-leaf">
                {item.title}
              </Link>
            </h2>

            <p className="mt-1 text-sm text-civic-stone">{item.description}</p>

            <div className="mt-3 flex items-center justify-between text-xs text-civic-stone">
              <span>{item.house} · {item.date}</span>
              <a
                href={item.evidence_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-0.5 text-civic-leaf hover:underline"
              >
                View evidence
                <ExternalLink className="h-3 w-3" aria-hidden="true" />
              </a>
            </div>
          </li>
        ))}
      </ul>

      <div className="mt-8 rounded-lg border border-civic-border bg-civic-mist p-4 text-xs text-civic-stone">
        <p>
          <strong className="text-civic-ink">How this works:</strong> This feed shows verified
          civic changes detected from official sources. Every item links to its evidence.
          The system does not editorialize — it shows what happened, not what to think about it.
        </p>
      </div>
    </div>
  );
}
