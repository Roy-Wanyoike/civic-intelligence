import Link from 'next/link';
import { TrendingUp, FileSignature, Clock } from 'lucide-react';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Trending Bills',
  description: 'Track bills signed into law, approaching final stages, and recently published.',
};

interface BillItem {
  id: string;
  title: string;
  house: string;
  year: number;
  source_url: string;
  publication_date?: string;
  current_stage?: string;
}

interface TrendingData {
  recently_published: BillItem[];
  hot: BillItem[];
  approaching_final: BillItem[];
  total_bills: number;
  source: string;
}

async function fetchTrending(): Promise<TrendingData | null> {
  try {
    const resp = await fetch('http://localhost:9000/api/v1/trending', {
      next: { revalidate: 300 },
    });
    if (!resp.ok) return null;
    return resp.json();
  } catch {
    return null;
  }
}

export default async function TrendingPage() {
  const data = await fetchTrending();

  if (!data) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Trending Bills</h1>
        <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
          Trending data is temporarily unavailable. The Go API service may not be running.
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Trending Bills</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Track bills signed into law, approaching final stages, and recently published.
          Source: {data.source} · {data.total_bills} total bills tracked.
        </p>
      </header>

      {/* Approaching Final Stage */}
      <section className="mb-10">
        <div className="flex items-center gap-2 mb-4">
          <FileSignature className="h-5 w-5 text-civic-clay" aria-hidden="true" />
          <h2 className="font-serif text-xl font-semibold text-civic-ink">
            Approaching Final Stage
          </h2>
        </div>
        <p className="mb-4 text-xs text-civic-stone">
          Bills that may be nearing enactment — finance bills, amendments, and appropriation bills.
        </p>
        {data.approaching_final.length === 0 ? (
          <div className="rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
            No bills currently approaching final stage.
          </div>
        ) : (
          <ul className="grid gap-3 md:grid-cols-2">
            {data.approaching_final.map((bill) => (
              <li key={bill.id} className="rounded-lg border border-civic-clay/30 bg-civic-clay/5 p-4 hover:border-civic-clay">
                <Link href={`/bills/${bill.id}`} className="block">
                  <h3 className="font-serif text-base font-semibold text-civic-ink">{bill.title}</h3>
                  <p className="mt-1 text-xs text-civic-stone">{bill.house} · {bill.year}</p>
                  {bill.source_url && (
                    <p className="mt-1 text-xs text-civic-leaf truncate">{bill.source_url}</p>
                  )}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Hot Bills (most recent) */}
      <section className="mb-10">
        <div className="flex items-center gap-2 mb-4">
          <TrendingUp className="h-5 w-5 text-civic-leaf" aria-hidden="true" />
          <h2 className="font-serif text-xl font-semibold text-civic-ink">
            Hot Bills
          </h2>
        </div>
        <p className="mb-4 text-xs text-civic-stone">
          The most recently published bills on Kenya Law.
        </p>
        <ul className="grid gap-3 md:grid-cols-2">
          {data.hot.map((bill, i) => (
            <li key={bill.id} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
              <Link href={`/bills/${bill.id}`} className="block">
                <div className="flex items-center gap-2 text-xs text-civic-stone">
                  <span className="font-bold text-civic-leaf">#{i + 1}</span>
                  <span>{bill.house} · {bill.year}</span>
                </div>
                <h3 className="mt-1 font-serif text-base font-semibold text-civic-ink">{bill.title}</h3>
                {bill.publication_date && (
                  <p className="mt-1 text-xs text-civic-stone">Published: {bill.publication_date}</p>
                )}
              </Link>
            </li>
          ))}
        </ul>
      </section>

      {/* Recently Published */}
      <section className="mb-10">
        <div className="flex items-center gap-2 mb-4">
          <Clock className="h-5 w-5 text-civic-acacia" aria-hidden="true" />
          <h2 className="font-serif text-xl font-semibold text-civic-ink">
            Recently Published
          </h2>
        </div>
        <p className="mb-4 text-xs text-civic-stone">
          Bills published in the last 30 days.
        </p>
        {data.recently_published.length === 0 ? (
          <div className="rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
            No bills published in the last 30 days.
          </div>
        ) : (
          <ul className="grid gap-3 md:grid-cols-2">
            {data.recently_published.map((bill) => (
              <li key={bill.id} className="rounded-lg border border-civic-border bg-civic-paper p-4 hover:border-civic-leaf">
                <Link href={`/bills/${bill.id}`} className="block">
                  <h3 className="font-serif text-base font-semibold text-civic-ink">{bill.title}</h3>
                  <p className="mt-1 text-xs text-civic-stone">{bill.house} · {bill.publication_date}</p>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
