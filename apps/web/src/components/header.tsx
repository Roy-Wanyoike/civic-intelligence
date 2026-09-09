import Link from 'next/link';
import { Search, Scale, Menu } from 'lucide-react';

export function Header() {
  return (
    <header
      className="sticky top-0 z-50 border-b border-civic-border bg-civic-paper/95 backdrop-blur"
      role="banner"
    >
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6">
        <Link
          href="/"
          className="flex items-center gap-2 text-civic-forest hover:text-civic-leaf"
          aria-label="Civic Intelligence — home"
        >
          <Scale className="h-6 w-6" aria-hidden="true" />
          <span className="font-serif text-lg font-semibold tracking-tight">
            Civic Intelligence
          </span>
          <span className="hidden text-xs text-civic-stone sm:inline">
            · Kenya
          </span>
        </Link>

        <nav aria-label="Primary" className="hidden md:block">
          <ul className="flex items-center gap-6 text-sm">
            <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
            <li><Link href="/acts" className="hover:text-civic-leaf">Acts</Link></li>
            <li><Link href="/trending" className="hover:text-civic-leaf">Trending</Link></li>
            <li><Link href="/following" className="hover:text-civic-leaf">Following</Link></li>
            <li><Link href="/loans" className="hover:text-civic-leaf">Loans</Link></li>
            <li><Link href="/grants" className="hover:text-civic-leaf">Grants</Link></li>
            <li><Link href="/search" className="hover:text-civic-leaf">Search</Link></li>
            <li><Link href="/briefing" className="hover:text-civic-leaf">Daily Brief</Link></li>
            <li><Link href="/committees" className="hover:text-civic-leaf">Committees</Link></li>
            <li><Link href="/people" className="hover:text-civic-leaf">People</Link></li>
            <li><Link href="/about" className="hover:text-civic-leaf">About</Link></li>
          </ul>
        </nav>

        <div className="flex items-center gap-2">
          <Link
            href="/search"
            className="inline-flex items-center gap-1 rounded-md border border-civic-border px-3 py-1.5 text-sm hover:bg-civic-mist"
            aria-label="Search"
          >
            <Search className="h-4 w-4" aria-hidden="true" />
            <span className="hidden sm:inline">Search a Bill, law, or ask a question</span>
            <span className="sm:hidden">Search</span>
          </Link>
          <button
            type="button"
            className="md:hidden"
            aria-label="Open menu"
          >
            <Menu className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
      </div>
    </header>
  );
}
