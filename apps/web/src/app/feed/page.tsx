import Link from 'next/link';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Civic Feed',
  description: 'Chronological feed of verified Kenyan civic activity.',
};

interface FeedItem {
  kind: string;
  title: string;
  house?: string;
  date: string;
  source_url: string;
  description: string;
  significance: string;
}

async function fetchFeed(): Promise<{ items: FeedItem[]; total: number } | null> {
  try {
    const resp = await fetch('http://localhost:9000/api/v1/feed', { next: { revalidate: 300 } });
    if (!resp.ok) return null;
    return resp.json();
  } catch { return null; }
}

export default async function FeedPage() {
  const data = await fetchFeed();
  if (!data) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Civic Feed</h1>
        <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
          Feed temporarily unavailable.
        </div>
      </div>
    );
  }
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Civic Feed</h1>
      <p className="mt-2 text-sm text-civic-stone">Verified civic activity from official sources.</p>
      <ul className="mt-6 space-y-4">
        {data.items.map((item, i) => (
          <li key={i} className="rounded-lg border border-civic-border bg-civic-paper p-4">
            <div className="flex items-center gap-2 text-xs text-civic-stone">
              <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 font-medium text-civic-leaf">{item.kind.replace(/_/g, ' ')}</span>
              <span>{item.date}</span>
              {item.house && <span>· {item.house}</span>}
            </div>
            <h2 className="mt-2 font-serif text-lg font-semibold text-civic-ink">{item.title}</h2>
            <p className="mt-1 text-sm text-civic-stone">{item.description}</p>
            <a href={item.source_url} target="_blank" rel="noopener noreferrer" className="mt-2 inline-block text-xs text-civic-leaf hover:underline">
              View source ↗
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}
