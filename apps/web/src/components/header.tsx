'use client';

import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import {
  Search,
  Scale,
  Menu,
  X,
  ChevronDown,
  Bell,
  User,
  Heart,
  FileText,
  Landmark,
  Users,
  TrendingUp,
  Sparkles,
  BookOpen,
  DollarSign,
  BarChart3,
  Database,
  Globe,
  FileSearch,
  Radio,
  ShieldCheck,
  FileBarChart,
  Newspaper,
  Bookmark,
  BellRing,
  Code2,
  Info,
  HeartHandshake,
  LayoutDashboard,
  Scale as ScaleIcon,
} from 'lucide-react';
import { ThemeToggle } from './theme-toggle';

/**
 * Mega-menu navigation — every platform page is reachable from the navbar.
 *
 * Pages are organized into 6 thematic groups so the menu stays scannable
 * despite surfacing 30+ destinations:
 *
 *   Legislation   — Bills, Acts, Regulations, Policies, Committees
 *   Government    — Constitution, Governments, Institutions, People, Participation
 *   Finance       — Public Debt, Loans, Grants
 *   Intelligence  — What Changed, Feed, Trending, Research, Trust, Reports
 *   Scenarios     — What If?
 *   Resources     — Briefing, Gazette, Datasets, Dashboard, Topics,
 *                   Developers, About, Sponsor
 */

interface NavItem {
  href: string;
  label: string;
  icon: typeof FileText;
  description?: string;
}

interface NavGroup {
  title: string;
  icon: typeof FileText;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    title: 'Legislation',
    icon: FileText,
    items: [
      { href: '/bills', label: 'Bills', icon: FileText, description: 'Active and historical Bills before Parliament' },
      { href: '/acts', label: 'Acts', icon: ScaleIcon, description: 'Enacted Acts of Parliament with full lifecycle' },
      { href: '/regulations', label: 'Regulations', icon: FileSearch, description: 'Subordinate legislation and Legal Notices' },
      { href: '/policies', label: 'Policies', icon: FileBarChart, description: 'Government policy documents' },
      { href: '/committees', label: 'Committees', icon: Users, description: 'Parliamentary standing and select committees' },
    ],
  },
  {
    title: 'Government',
    icon: Landmark,
    items: [
      { href: '/constitution', label: 'Constitution', icon: BookOpen, description: 'The Constitution of Kenya, Article by Article' },
      { href: '/governments', label: 'Governments', icon: Landmark, description: 'Presidential administrations and terms' },
      { href: '/institutions', label: 'Institutions', icon: Landmark, description: 'Government institutions and agencies' },
      { href: '/people', label: 'People', icon: Users, description: 'MPs, senators, and civic persons' },
      { href: '/participation', label: 'Participation', icon: Users, description: 'Public participation opportunities' },
    ],
  },
  {
    title: 'Finance',
    icon: DollarSign,
    items: [
      { href: '/debt', label: 'Public Debt', icon: BarChart3, description: 'National debt dashboard with trends' },
      { href: '/loans', label: 'Loans', icon: DollarSign, description: 'Sovereign loans tracker' },
      { href: '/grants', label: 'Grants', icon: HeartHandshake, description: 'Grants received by the government' },
    ],
  },
  {
    title: 'Intelligence',
    icon: TrendingUp,
    items: [
      { href: '/what-changed', label: 'What Changed', icon: Radio, description: 'Proactive civic change feed' },
      { href: '/feed', label: 'Feed', icon: Newspaper, description: 'Full civic activity feed' },
      { href: '/trending', label: 'Trending', icon: TrendingUp, description: 'Trending Bills and topics' },
      { href: '/research', label: 'Research', icon: FileSearch, description: 'Research missions and reports' },
      { href: '/trust', label: 'Trust', icon: ShieldCheck, description: 'Trust and verification network' },
      { href: '/report', label: 'Reports', icon: FileBarChart, description: 'Generated civic reports' },
    ],
  },
  {
    title: 'Scenarios',
    icon: Sparkles,
    items: [
      { href: '/scenarios', label: 'What If?', icon: Sparkles, description: 'Explore hypothetical civic scenarios — HYPOTHETICAL, not observed fact' },
    ],
  },
  {
    title: 'Resources',
    icon: Database,
    items: [
      { href: '/briefing', label: 'Daily Briefing', icon: Newspaper, description: 'Today\'s civic brief' },
      { href: '/gazette', label: 'Kenya Gazette', icon: FileText, description: 'Official gazette notices' },
      { href: '/datasets', label: 'Datasets', icon: Database, description: 'Downloadable civic datasets' },
      { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard, description: 'Your civic dashboard' },
      { href: '/topics', label: 'Topics', icon: BookOpen, description: 'Browse by topic' },
      { href: '/developers', label: 'Developers', icon: Code2, description: 'API docs and developer platform' },
      { href: '/about', label: 'About', icon: Info, description: 'About Civic Intelligence' },
      { href: '/sponsor', label: 'Sponsor', icon: Heart, description: 'Support the platform' },
    ],
  },
];

const primaryNav = [
  { href: '/', label: 'Home' },
  { href: '/ask', label: 'Ask' },
  { href: '/briefing', label: 'Briefing' },
  { href: '/countries', label: 'Countries' },
] as const;

export function Header() {
  const [exploreOpen, setExploreOpen] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [searchExpanded, setSearchExpanded] = useState(false);
  const exploreRef = useRef<HTMLLIElement>(null);

  // Close the Explore dropdown when a click lands outside it.
  useEffect(() => {
    if (!exploreOpen) return;
    function handleClickOutside(e: MouseEvent) {
      if (exploreRef.current && !exploreRef.current.contains(e.target as Node)) {
        setExploreOpen(false);
      }
    }
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') setExploreOpen(false);
    }
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [exploreOpen]);

  // Lock body scroll while the mobile drawer is open.
  useEffect(() => {
    if (!mobileOpen) {
      document.body.style.overflow = '';
      return;
    }
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = '';
    };
  }, [mobileOpen]);

  return (
    <header
      className="sticky top-0 z-50 border-b border-civic-border bg-civic-paper/95 backdrop-blur"
      role="banner"
    >
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-8 px-4 py-3 sm:px-6 lg:px-8">
        {/* Logo + brand (left) */}
        <Link
          href="/"
          className="flex flex-shrink-0 items-center gap-2.5 text-civic-forest hover:text-civic-leaf"
          aria-label="Civic Intelligence — home"
        >
          <Scale className="h-7 w-7" aria-hidden="true" />
          <span className="hidden font-serif text-lg font-semibold tracking-tight sm:inline">
            Civic Intelligence
          </span>
        </Link>

        {/* Primary nav (center) — capped at 5 top-level items */}
        <nav aria-label="Primary" className="hidden lg:block">
          <ul className="flex items-center gap-7 text-sm font-medium text-civic-ink">
            <li>
              <Link
                href="/"
                className="inline-flex items-center py-3 hover:text-civic-leaf"
              >
                Home
              </Link>
            </li>
            <li>
              <Link
                href="/ask"
                className="inline-flex items-center py-3 hover:text-civic-leaf"
              >
                Ask
              </Link>
            </li>
            <li>
              <Link
                href="/briefing"
                className="inline-flex items-center py-3 hover:text-civic-leaf"
              >
                Briefing
              </Link>
            </li>
            {/* Mega-menu Explore */}
            <li ref={exploreRef} className="relative">
              <button
                type="button"
                onClick={() => setExploreOpen((v) => !v)}
                aria-expanded={exploreOpen}
                aria-haspopup="true"
                className="inline-flex items-center gap-1 py-3 hover:text-civic-leaf"
              >
                Explore
                <ChevronDown
                  className={`h-4 w-4 transition-transform duration-200 ${
                    exploreOpen ? 'rotate-180' : ''
                  }`}
                  aria-hidden="true"
                />
              </button>
              {exploreOpen && (
                <div
                  role="menu"
                  aria-label="Explore all sections"
                  className="absolute left-1/2 top-full z-50 mt-1 w-[min(56rem,calc(100vw-2rem))] -translate-x-1/2 origin-top rounded-xl border border-civic-border bg-civic-paper shadow-2xl"
                >
                  <div className="grid grid-cols-3 gap-x-6 gap-y-1 p-5">
                    {navGroups.map((group) => (
                      <div key={group.title} className="space-y-1">
                        <div className="flex items-center gap-1.5 px-2 pb-1 pt-2">
                          <group.icon className="h-3.5 w-3.5 text-civic-leaf" aria-hidden="true" />
                          <span className="text-[11px] font-semibold uppercase tracking-wider text-civic-stone">
                            {group.title}
                          </span>
                        </div>
                        <ul className="space-y-0.5">
                          {group.items.map((item) => (
                            <li key={item.href}>
                              <Link
                                href={item.href}
                                onClick={() => setExploreOpen(false)}
                                role="menuitem"
                                className="group flex items-start gap-2.5 rounded-md px-2 py-1.5 text-sm text-civic-ink transition hover:bg-civic-mist hover:text-civic-leaf"
                              >
                                <item.icon className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-stone group-hover:text-civic-leaf" aria-hidden="true" />
                                <span className="min-w-0">
                                  <span className="block font-medium">{item.label}</span>
                                  {item.description && (
                                    <span className="block truncate text-[11px] text-civic-stone">
                                      {item.description}
                                    </span>
                                  )}
                                </span>
                              </Link>
                            </li>
                          ))}
                        </ul>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </li>
            <li>
              <Link
                href="/countries"
                className="inline-flex items-center gap-1 py-3 hover:text-civic-leaf"
              >
                <Globe className="h-4 w-4" aria-hidden="true" />
                Countries
              </Link>
            </li>
          </ul>
        </nav>

        {/* Right cluster: expandable search + user actions */}
        <div className="flex flex-shrink-0 items-center gap-2 sm:gap-3">
          {/* Expandable search bar */}
          <form action="/search" className="flex items-center" role="search">
            <label htmlFor="nav-search" className="sr-only">
              Search a Bill, law, or ask a question
            </label>
            <div
              className={`hidden items-center overflow-hidden rounded-full border border-civic-border bg-civic-mist transition-all duration-200 sm:flex ${
                searchExpanded ? 'w-64' : 'w-44'
              }`}
            >
              <Search
                className="ml-3 h-4 w-4 flex-shrink-0 text-civic-stone"
                aria-hidden="true"
              />
              <input
                id="nav-search"
                type="search"
                name="q"
                placeholder="Search Bills, laws, ask…"
                aria-label="Search Bills, laws, or ask a question"
                autoComplete="off"
                onFocus={() => setSearchExpanded(true)}
                onBlur={() => setSearchExpanded(false)}
                className="w-full bg-transparent px-2 py-2 text-sm placeholder:text-civic-stone focus:outline-none"
              />
            </div>
            <Link
              href="/search"
              aria-label="Search"
              className="inline-flex h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist sm:hidden"
            >
              <Search className="h-5 w-5" aria-hidden="true" />
            </Link>
          </form>

          {/* Following / Bookmarks */}
          <Link
            href="/following"
            aria-label="Following"
            title="Following"
            className="hidden h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist sm:inline-flex"
          >
            <Bookmark className="h-5 w-5" aria-hidden="true" />
          </Link>

          {/* Notifications */}
          <Link
            href="/notifications"
            aria-label="Notifications"
            title="Notifications"
            className="relative inline-flex h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist"
          >
            <Bell className="h-5 w-5" aria-hidden="true" />
          </Link>

          {/* Dark/Light theme toggle */}
          <ThemeToggle />

          {/* Sponsor */}
          <Link
            href="/sponsor"
            className="hidden items-center gap-1.5 rounded-full bg-civic-forest px-4 py-2.5 text-sm font-medium text-civic-paper hover:bg-civic-leaf sm:inline-flex"
          >
            <Heart className="h-4 w-4" aria-hidden="true" />
            <span className="hidden md:inline">Sponsor</span>
          </Link>

          {/* Hamburger (mobile) */}
          <button
            type="button"
            onClick={() => setMobileOpen(true)}
            className="inline-flex h-10 w-10 items-center justify-center rounded-md text-civic-forest hover:bg-civic-mist lg:hidden"
            aria-label="Open menu"
            aria-expanded={mobileOpen}
          >
            <Menu className="h-6 w-6" aria-hidden="true" />
          </button>
        </div>
      </div>

      {/* Mobile slide-in drawer */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-50 lg:hidden"
          role="dialog"
          aria-modal="true"
          aria-label="Site menu"
        >
          <div
            className="absolute inset-0 bg-civic-ink/40 backdrop-blur-sm"
            onClick={() => setMobileOpen(false)}
            aria-hidden="true"
          />
          <div className="absolute right-0 top-0 flex h-full w-80 max-w-[85vw] flex-col bg-civic-paper shadow-xl">
            <div className="flex items-center justify-between border-b border-civic-border px-4 py-4">
              <span className="font-serif text-lg font-semibold text-civic-forest">
                Menu
              </span>
              <button
                type="button"
                onClick={() => setMobileOpen(false)}
                className="inline-flex h-9 w-9 items-center justify-center rounded-md hover:bg-civic-mist"
                aria-label="Close menu"
              >
                <X className="h-5 w-5" aria-hidden="true" />
              </button>
            </div>
            <nav aria-label="Mobile" className="flex-1 overflow-y-auto px-4 py-4">
              {/* Primary nav */}
              <ul className="space-y-1">
                {primaryNav.map((item) => (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      onClick={() => setMobileOpen(false)}
                      className="block rounded-md px-3 py-3 text-base font-medium text-civic-ink hover:bg-civic-mist"
                    >
                      {item.label}
                    </Link>
                  </li>
                ))}
              </ul>
              {/* Grouped explore links */}
              {navGroups.map((group) => (
                <div key={group.title} className="mt-5">
                  <div className="flex items-center gap-1.5 px-3 pb-1">
                    <group.icon className="h-3.5 w-3.5 text-civic-leaf" aria-hidden="true" />
                    <span className="text-xs font-semibold uppercase tracking-wider text-civic-stone">
                      {group.title}
                    </span>
                  </div>
                  <ul className="mt-1 space-y-0.5">
                    {group.items.map((item) => (
                      <li key={item.href}>
                        <Link
                          href={item.href}
                          onClick={() => setMobileOpen(false)}
                          className="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm text-civic-ink hover:bg-civic-mist hover:text-civic-leaf"
                        >
                          <item.icon className="h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
                          {item.label}
                        </Link>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </nav>
            <div className="space-y-2 border-t border-civic-border px-4 py-4">
              <Link
                href="/sponsor"
                onClick={() => setMobileOpen(false)}
                className="flex items-center justify-center gap-2 rounded-md bg-civic-forest px-4 py-3 text-sm font-medium text-civic-paper hover:bg-civic-leaf"
              >
                <Heart className="h-4 w-4" aria-hidden="true" />
                Sponsor
              </Link>
              <div className="grid grid-cols-2 gap-2">
                <Link
                  href="/notifications"
                  onClick={() => setMobileOpen(false)}
                  className="flex items-center justify-center gap-2 rounded-md border border-civic-border px-4 py-3 text-sm text-civic-ink hover:bg-civic-mist"
                >
                  <BellRing className="h-4 w-4" aria-hidden="true" />
                  Alerts
                </Link>
                <Link
                  href="/following"
                  onClick={() => setMobileOpen(false)}
                  className="flex items-center justify-center gap-2 rounded-md border border-civic-border px-4 py-3 text-sm text-civic-ink hover:bg-civic-mist"
                >
                  <User className="h-4 w-4" aria-hidden="true" />
                  Account
                </Link>
              </div>
            </div>
          </div>
        </div>
      )}
    </header>
  );
}
