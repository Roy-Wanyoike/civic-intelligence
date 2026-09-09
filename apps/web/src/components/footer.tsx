import Link from 'next/link';

export function Footer() {
  return (
    <footer className="border-t border-civic-border bg-civic-mist" role="contentinfo">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
        <div className="grid gap-8 md:grid-cols-4">
          <div>
            <h2 className="font-serif text-base font-semibold text-civic-forest">
              Civic Intelligence
            </h2>
            <p className="mt-2 text-sm text-civic-stone">
              Evidence-grounded civic intelligence for Kenya. We show what
              happened, explain what it means, and let you decide.
            </p>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-civic-ink">Explore</h3>
            <ul className="mt-2 space-y-1 text-sm">
              <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
              <li><Link href="/search" className="hover:text-civic-leaf">Search</Link></li>
              <li><Link href="/briefing" className="hover:text-civic-leaf">Daily Brief</Link></li>
              <li><Link href="/committees" className="hover:text-civic-leaf">Committees</Link></li>
            </ul>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-civic-ink">Principles</h3>
            <ul className="mt-2 space-y-1 text-sm text-civic-stone">
              <li>Evidence before AI</li>
              <li>Never fabricate</li>
              <li>Version everything</li>
              <li>Politically neutral</li>
            </ul>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-civic-ink">Developers</h3>
            <ul className="mt-2 space-y-1 text-sm">
              <li><Link href="/about" className="hover:text-civic-leaf">About</Link></li>
              <li>
                <a
                  href="https://github.com/Roy-Wanyoike/civic-intelligence"
                  className="hover:text-civic-leaf"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  GitHub ↗
                </a>
              </li>
            </ul>
          </div>
        </div>
        <div className="mt-8 border-t border-civic-border pt-4 text-xs text-civic-stone">
          <p>
            Civic Intelligence is an independent civic-information project. It is
            not affiliated with the Government of Kenya. All claims are
            traceable to official sources. This platform does not provide legal
            advice.
          </p>
        </div>
      </div>
    </footer>
  );
}
