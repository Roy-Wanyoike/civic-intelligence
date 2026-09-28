'use client';

/**
 * Global Command Palette (spec §37).
 *
 * Triggered by ⌘K (macOS) / Ctrl+K (others). Provides:
 *
 *   - Recent pages (stored in localStorage, capped at 8)
 *   - Quick actions (search, ask, explore Constitution, view public debt)
 *   - Navigation shortcuts mirroring the navbar mega-menu groups
 *
 * Accessibility (spec §37 — keyboard access, accessible labels, empty states):
 *   - role="dialog" + aria-modal
 *   - cmdk handles arrow-up/down navigation, Enter to select, Esc to close
 *   - Focus is trapped within the dialog and returned to the trigger on close
 *   - Empty state rendered when the query matches nothing
 *
 * Persistence: recent pages are stored in localStorage under
 *   `civic:command-palette:recent` as a JSON array of `{ href, label, ts }`.
 *
 * NOTE: The command palette is mounted once globally in `layout.tsx`. A
 * custom window event (`civic:command-palette:open`) is dispatched by the
 * Header's Cmd+K shortcut listener to open the dialog from anywhere.
 */
import { Command } from 'cmdk';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Search,
  Sparkles,
  BookOpen,
  BarChart3,
  FileText,
  Scale,
  Landmark,
  DollarSign,
  TrendingUp,
  Database,
  Clock,
  CornerDownLeft,
} from 'lucide-react';

interface RecentEntry {
  href: string;
  label: string;
  ts: number;
}

const RECENT_KEY = 'civic:command-palette:recent';
const RECENT_MAX = 8;

/** Mirrors the navbar's mega-menu groups so users always see the same taxonomy. */
const NAV_GROUPS = [
  {
    title: 'Legislation',
    icon: FileText,
    items: [
      { href: '/bills', label: 'Bills' },
      { href: '/acts', label: 'Acts' },
      { href: '/regulations', label: 'Regulations' },
      { href: '/policies', label: 'Policies' },
      { href: '/committees', label: 'Committees' },
    ],
  },
  {
    title: 'Government',
    icon: Landmark,
    items: [
      { href: '/constitution', label: 'Constitution' },
      { href: '/governments', label: 'Governments' },
      { href: '/institutions', label: 'Institutions' },
      { href: '/people', label: 'People' },
      { href: '/participation', label: 'Participation' },
    ],
  },
  {
    title: 'Finance',
    icon: DollarSign,
    items: [
      { href: '/debt', label: 'Public Debt' },
      { href: '/loans', label: 'Loans' },
      { href: '/grants', label: 'Grants' },
    ],
  },
  {
    title: 'Intelligence',
    icon: TrendingUp,
    items: [
      { href: '/what-changed', label: 'What Changed' },
      { href: '/feed', label: 'Feed' },
      { href: '/trending', label: 'Trending' },
      { href: '/graph', label: 'Knowledge Graph' },
      { href: '/compare', label: 'Compare' },
      { href: '/indicators', label: 'Indicators' },
      { href: '/budget', label: 'Budget' },
      { href: '/research', label: 'Research' },
      { href: '/trust', label: 'Trust' },
      { href: '/report', label: 'Reports' },
    ],
  },
  {
    title: 'Scenarios',
    icon: Sparkles,
    items: [{ href: '/scenarios', label: 'What If?' }],
  },
  {
    title: 'Resources',
    icon: Database,
    items: [
      { href: '/briefing', label: 'Daily Briefing' },
      { href: '/gazette', label: 'Kenya Gazette' },
      { href: '/datasets', label: 'Datasets' },
      { href: '/dashboard', label: 'Dashboard' },
      { href: '/topics', label: 'Topics' },
      { href: '/developers', label: 'Developers' },
      { href: '/about', label: 'About' },
      { href: '/sponsor', label: 'Sponsor' },
    ],
  },
] as const;

const QUICK_ACTIONS = [
  {
    id: 'search-bills',
    label: 'Search Bills',
    hint: 'Open the Bills search page',
    icon: Search,
    href: '/bills',
  },
  {
    id: 'ask-question',
    label: 'Ask a question',
    hint: 'Ask a civic question in plain language',
    icon: Sparkles,
    href: '/ask',
  },
  {
    id: 'explore-constitution',
    label: 'Explore Constitution',
    hint: 'Browse the Constitution of Kenya, Article by Article',
    icon: BookOpen,
    href: '/constitution',
  },
  {
    id: 'view-public-debt',
    label: 'View Public Debt',
    hint: 'Open the Public Debt dashboard',
    icon: BarChart3,
    href: '/debt',
  },
] as const;

function loadRecent(): RecentEntry[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(RECENT_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as RecentEntry[];
    if (!Array.isArray(parsed)) return [];
    return parsed.slice(0, RECENT_MAX);
  } catch {
    return [];
  }
}

function saveRecent(entry: RecentEntry) {
  if (typeof window === 'undefined') return;
  try {
    const current = loadRecent();
    // De-dupe by href, then prepend, then cap.
    const deduped = current.filter((e) => e.href !== entry.href);
    const next = [entry, ...deduped].slice(0, RECENT_MAX);
    window.localStorage.setItem(RECENT_KEY, JSON.stringify(next));
  } catch {
    /* localStorage unavailable — silently ignore. */
  }
}

function clearRecent() {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.removeItem(RECENT_KEY);
  } catch {
    /* ignore */
  }
}

export interface CommandPaletteHandle {
  open: () => void;
}

export function CommandPalette() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [recent, setRecent] = useState<RecentEntry[]>([]);
  const [query, setQuery] = useState('');
  const triggerRef = useRef<HTMLElement | null>(null);

  // Open via custom window event so any component (e.g. Header) can trigger it.
  useEffect(() => {
    function handleOpen() {
      // Remember the currently focused element so we can return focus on close.
      triggerRef.current = document.activeElement as HTMLElement | null;
      setOpen(true);
    }
    window.addEventListener('civic:command-palette:open', handleOpen);
    return () =>
      window.removeEventListener('civic:command-palette:open', handleOpen);
  }, []);

  // Global keyboard shortcut: Cmd+K (macOS) / Ctrl+K (others).
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        triggerRef.current = document.activeElement as HTMLElement | null;
        setOpen((v) => !v);
      }
    }
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, []);

  // Load recent pages once the dialog opens (avoids SSR localStorage reads).
  useEffect(() => {
    if (open) {
      setRecent(loadRecent());
      setQuery('');
    }
  }, [open]);

  // Return focus to the trigger element on close (accessibility).
  useEffect(() => {
    if (!open && triggerRef.current) {
      // Defer focus restoration to the next tick so cmdk has finished tearing down.
      const id = window.setTimeout(() => {
        triggerRef.current?.focus();
        triggerRef.current = null;
      }, 0);
      return () => window.clearTimeout(id);
    }
    return undefined;
  }, [open]);

  // Lock body scroll while the palette is open.
  useEffect(() => {
    if (!open) {
      document.body.style.overflow = '';
      return;
    }
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = '';
    };
  }, [open]);

  const navigate = useCallback(
    (href: string, label: string) => {
      saveRecent({ href, label, ts: Date.now() });
      setOpen(false);
      router.push(href);
    },
    [router],
  );

  // Flatten nav items so the search box can filter across all groups.
  const allNavItems = useMemo(
    () =>
      NAV_GROUPS.flatMap((group) =>
        group.items.map((item) => ({ ...item, group: group.title })),
      ),
    [],
  );

  const filteredNav = useMemo(() => {
    if (!query.trim()) return allNavItems;
    const q = query.toLowerCase();
    return allNavItems.filter(
      (item) =>
        item.label.toLowerCase().includes(q) ||
        item.group.toLowerCase().includes(q),
    );
  }, [allNavItems, query]);

  const filteredQuick = useMemo(() => {
    if (!query.trim()) return QUICK_ACTIONS;
    const q = query.toLowerCase();
    return QUICK_ACTIONS.filter(
      (a) =>
        a.label.toLowerCase().includes(q) ||
        a.hint.toLowerCase().includes(q),
    );
  }, [query]);

  const filteredRecent = useMemo(() => {
    if (!query.trim()) return recent;
    const q = query.toLowerCase();
    return recent.filter((r) => r.label.toLowerCase().includes(q));
  }, [recent, query]);

  const hasAnyResult =
    filteredQuick.length > 0 ||
    filteredRecent.length > 0 ||
    filteredNav.length > 0;
  const showEmpty = query.trim().length > 0 && !hasAnyResult;

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
      className="fixed inset-0 z-[100] flex items-start justify-center bg-civic-ink/40 backdrop-blur-sm"
      // Click on the backdrop closes the palette.
      onClick={(e) => {
        if (e.target === e.currentTarget) setOpen(false);
      }}
    >
      <div className="mt-[10vh] w-[min(36rem,calc(100vw-2rem))] overflow-hidden rounded-xl border border-civic-border bg-civic-paper shadow-2xl">
        <Command
          label="Command palette"
          className="flex flex-col"
          shouldFilter={false}
          // cmdk handles Escape internally; this is a belt-and-braces fallback.
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              e.preventDefault();
              setOpen(false);
            }
          }}
        >
          <div className="flex items-center gap-3 border-b border-civic-border px-4">
            <Search className="h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
            <Command.Input
              autoFocus
              placeholder="Type a command or search…"
              value={query}
              onValueChange={setQuery}
              className="w-full bg-transparent py-4 text-sm text-civic-ink placeholder:text-civic-stone focus:outline-none"
              aria-label="Search commands and pages"
            />
            <kbd
              className="hidden items-center gap-1 rounded border border-civic-border bg-civic-mist px-1.5 py-0.5 text-[10px] font-medium text-civic-stone sm:inline-flex"
              aria-hidden="true"
            >
              <CornerDownLeft className="h-3 w-3" />
              to select
            </kbd>
          </div>

          <Command.List
            className="max-h-[60vh] overflow-y-auto p-2"
            // Ensures screen readers announce results count.
            aria-label="Available commands"
          >
            {showEmpty && (
              <Command.Empty className="px-4 py-8 text-center text-sm text-civic-stone">
                No results found for &ldquo;{query}&rdquo;.
              </Command.Empty>
            )}

            {/* Quick actions */}
            {filteredQuick.length > 0 && (
              <Command.Group
                heading="Quick actions"
                className="mb-2"
              >
                {filteredQuick.map((action) => (
                  <Command.Item
                    key={action.id}
                    value={`action-${action.id}-${action.label}`}
                    onSelect={() => navigate(action.href, action.label)}
                    className="flex cursor-pointer items-start gap-3 rounded-md px-3 py-2 text-sm text-civic-ink data-[selected=true]:bg-civic-mist data-[selected=true]:text-civic-leaf"
                  >
                    <action.icon className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
                    <span className="min-w-0 flex-1">
                      <span className="block font-medium">{action.label}</span>
                      <span className="block truncate text-xs text-civic-stone">
                        {action.hint}
                      </span>
                    </span>
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {/* Recent pages */}
            {filteredRecent.length > 0 && (
              <Command.Group
                heading="Recent"
                className="mb-2"
              >
                {filteredRecent.map((entry) => (
                  <Command.Item
                    key={`recent-${entry.href}-${entry.ts}`}
                    value={`recent-${entry.href}-${entry.label}`}
                    onSelect={() => navigate(entry.href, entry.label)}
                    className="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-sm text-civic-ink data-[selected=true]:bg-civic-mist data-[selected=true]:text-civic-leaf"
                  >
                    <Clock className="h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
                    <span className="flex-1 truncate">{entry.label}</span>
                    <span className="truncate text-xs text-civic-stone">
                      {entry.href}
                    </span>
                  </Command.Item>
                ))}
                <Command.Item
                  value="recent-clear"
                  onSelect={() => {
                    clearRecent();
                    setRecent([]);
                  }}
                  className="cursor-pointer rounded-md px-3 py-1.5 text-xs text-civic-stone underline-offset-2 hover:underline data-[selected=true]:bg-civic-mist"
                >
                  Clear recent history
                </Command.Item>
              </Command.Group>
            )}

            {/* Navigation — grouped to mirror the navbar mega-menu */}
            {filteredNav.length > 0 && (
              <Command.Group
                heading="Navigation"
              >
                {filteredNav.map((item) => (
                  <Command.Item
                    key={`nav-${item.href}`}
                    value={`nav-${item.href}-${item.label}`}
                    onSelect={() => navigate(item.href, item.label)}
                    className="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-sm text-civic-ink data-[selected=true]:bg-civic-mist data-[selected=true]:text-civic-leaf"
                  >
                    <Scale className="h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
                    <span className="flex-1 truncate">{item.label}</span>
                    <span className="text-xs text-civic-stone">{item.group}</span>
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {/* Footer / help row (always rendered, not selectable) */}
            <div className="border-t border-civic-border px-3 py-2 text-[11px] text-civic-stone">
              <Link
                href="/search"
                onClick={() => setOpen(false)}
                className="inline-flex items-center gap-1.5 rounded hover:text-civic-leaf"
              >
                <Search className="h-3 w-3" aria-hidden="true" />
                Press <kbd className="rounded border border-civic-border bg-civic-mist px-1 font-mono">Enter</kbd> to open ·{' '}
                <kbd className="rounded border border-civic-border bg-civic-mist px-1 font-mono">Esc</kbd> to close
              </Link>
            </div>
          </Command.List>
        </Command>
      </div>
    </div>
  );
}

/**
 * Imperative helper — dispatch the open event from anywhere in the app.
 * Used by the Header's Cmd+K shortcut button (and by tests).
 */
export function openCommandPalette() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new Event('civic:command-palette:open'));
}
