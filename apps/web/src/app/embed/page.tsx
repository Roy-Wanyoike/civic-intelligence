import Link from 'next/link';
import type { Metadata } from 'next';

// The /embed index is NOT chrome-less — it's a small landing page
// inside the normal app shell so a developer who navigates to the
// embed root (rather than a specific widget URL) sees the list of
// available widgets + the documentation link, not a 404. This is
// distinct from the /embed/<widget> routes, which are the actual
// chrome-less iframe content.
export const metadata: Metadata = {
  title: 'Embeddable Widgets',
  description: 'Civic Intelligence embeddable widgets for third-party sites.',
};

const WIDGETS = [
  {
    name: 'MP Finder',
    path: '/embed/mp-finder',
    description:
      'Constituency name → MP name + party + scorecard link. Simple GET form, no client-side JS.',
    width: 400,
    height: 320,
  },
  {
    name: 'Bill Tracker',
    path: '/embed/bill-tracker',
    description:
      "Compact horizontal timeline of a Bill's 8 legislative stages. Pass ?bill_id=…",
    width: 480,
    height: 240,
  },
  {
    name: 'Today in Parliament',
    path: '/embed/today',
    description: "Today's parliament calendar events in a compact list.",
    width: 400,
    height: 360,
  },
] as const;

export default function EmbedIndexPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">
          Embeddable widgets
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Drop these chrome-less iframe widgets into any third-party site — a
          newsroom article, a civil-society voter guide, or a school resources
          page. Each widget renders with no navbar/footer and minimal inline
          CSS so the host page&rsquo;s styles can&rsquo;t leak in or out.
        </p>
        <p className="mt-2 text-sm">
          <Link href="/developers" className="text-civic-leaf hover:underline">
            See the developer docs for copy-pasteable iframe snippets →
          </Link>
        </p>
      </header>

      <ul className="mt-8 space-y-4">
        {WIDGETS.map((w) => (
          <li
            key={w.path}
            className="rounded-lg border border-civic-border bg-civic-paper p-4"
          >
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <h2 className="font-serif text-lg font-semibold text-civic-forest">
                {w.name}
              </h2>
              <code className="font-mono text-xs text-civic-stone">{w.path}</code>
            </div>
            <p className="mt-1 text-sm text-civic-ink">{w.description}</p>
            <p className="mt-2 text-xs text-civic-stone">
              Suggested size: {w.width} × {w.height}px
            </p>
          </li>
        ))}
      </ul>

      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-6 text-sm text-civic-stone">
        These widgets are designed to be loaded via{' '}
        <code className="rounded bg-civic-paper px-1.5 py-0.5 font-mono text-xs text-civic-forest">
          &lt;iframe&gt;
        </code>
        . They render with no app chrome and use{' '}
        <code className="rounded bg-civic-paper px-1.5 py-0.5 font-mono text-xs text-civic-forest">
          frame-ancestors *
        </code>{' '}
        so any origin may embed them. See{' '}
        <Link href="/developers" className="text-civic-leaf hover:underline">
          /developers
        </Link>{' '}
        for live previews + snippet copy-paste.
      </div>
    </div>
  );
}
