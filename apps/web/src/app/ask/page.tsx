import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Ask Kenya',
  description: 'Ask a question about Kenyan civic activity.',
};

export default function AskPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Ask Kenya</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Ask any question about Kenyan legislation, regulations, or government activity.
        Every answer is evidence-grounded.
      </p>
      <div className="mt-8">
        <form action="/search" className="flex gap-2">
          <input
            type="search"
            name="q"
            placeholder="What is Kenya doing about housing?"
            className="flex-1 rounded-md border border-civic-border bg-civic-paper px-4 py-3 text-sm focus:outline-none focus:border-civic-leaf"
            autoFocus
          />
          <button type="submit" className="rounded-md bg-civic-leaf px-5 py-3 font-semibold text-white hover:bg-civic-leaf/90">
            Ask
          </button>
        </form>
      </div>
      <div className="mt-6 flex flex-wrap gap-2">
        {['What is happening with the Housing Bill?', 'What did the President sign today?', 'What laws affect small businesses?', 'What changed in data protection?'].map(q => (
          <a key={q} href={`/search?q=${encodeURIComponent(q)}`} className="rounded-full bg-civic-mist px-3 py-1 text-sm text-civic-stone hover:bg-civic-leaf/10 hover:text-civic-leaf">
            {q}
          </a>
        ))}
      </div>
    </div>
  );
}
