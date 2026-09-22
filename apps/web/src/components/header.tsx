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
  Command as CommandIcon,
  CalendarDays,
  MapPin,
  LogIn,
  Network,
} from 'lucide-react';
import { useTranslations } from 'next-intl';
import { ThemeToggle } from './theme-toggle';
import { LanguageSwitcher } from './language-switcher';
import { openCommandPalette } from './command-palette';

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
 *
 * i18n (spec §66): labels are pulled from the `nav` message namespace via
 * `useTranslations('nav')`. Each nav item carries an `i18nKey` (e.g.
 * `'pages.bills'`) that resolves to the localized label. Descriptions and
 * other secondary strings are still hard-coded pending translation.
 * The brand name "Civic Intelligence" and the "Menu" label are intentionally English across all locales (product decision).
 */

interface NavItem {
  href: string;
  /** Translation key under the `nav` namespace (e.g. `'pages.bills'`). */
  i18nKey: string;
  icon: typeof FileText;
  description?: string;
}

interface NavGroup {
  /** Translation key under `nav.groups.*`. */
  i18nKey: string;
  icon: typeof FileText;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    i18nKey: 'groups.legislation',
    icon: FileText,
    items: [
      { href: '/bills', i18nKey: 'pages.bills', icon: FileText, description: 'Active and historical Bills before Parliament' },
      { href: '/acts', i18nKey: 'pages.acts', icon: ScaleIcon, description: 'Enacted Acts of Parliament with full lifecycle' },
      { href: '/regulations', i18nKey: 'pages.regulations', icon: FileSearch, description: 'Subordinate legislation and Legal Notices' },
      { href: '/policies', i18nKey: 'pages.policies', icon: FileBarChart, description: 'Government policy documents' },
      { href: '/committees', i18nKey: 'pages.committees', icon: Users, description: 'Parliamentary standing and select committees' },
    ],
  },
  {
    i18nKey: 'groups.government',
    icon: Landmark,
    items: [
      { href: '/constitution', i18nKey: 'pages.constitution', icon: BookOpen, description: 'The Constitution of Kenya, Article by Article' },
      { href: '/governments', i18nKey: 'pages.governments', icon: Landmark, description: 'Presidential administrations and terms' },
      { href: '/legislatures', i18nKey: 'pages.legislatures', icon: Landmark, description: 'Parliaments — 11th, 12th, 13th — with Bills, Acts, committees' },
      { href: '/institutions', i18nKey: 'pages.institutions', icon: Landmark, description: 'Government institutions and agencies' },
      { href: '/people', i18nKey: 'pages.people', icon: Users, description: 'MPs, senators, and civic persons' },
      { href: '/constituencies', i18nKey: 'pages.constituencies', icon: MapPin, description: 'Per-constituency civic dashboards (MP, bills, budget, projects)' },
      { href: '/participation', i18nKey: 'pages.participation', icon: Users, description: 'Public participation opportunities' },
    ],
  },
  {
    i18nKey: 'groups.finance',
    icon: DollarSign,
    items: [
      { href: '/debt', i18nKey: 'pages.public_debt', icon: BarChart3, description: 'National debt dashboard with trends' },
      { href: '/loans', i18nKey: 'pages.loans', icon: DollarSign, description: 'Sovereign loans tracker' },
      { href: '/grants', i18nKey: 'pages.grants', icon: HeartHandshake, description: 'Grants received by the government' },
    ],
  },
  {
    i18nKey: 'groups.intelligence',
    icon: TrendingUp,
    items: [
      { href: '/what-changed', i18nKey: 'pages.what_changed', icon: Radio, description: 'Proactive civic change feed' },
      { href: '/feed', i18nKey: 'pages.feed', icon: Newspaper, description: 'Full civic activity feed' },
      { href: '/trending', i18nKey: 'pages.trending', icon: TrendingUp, description: 'Trending Bills and topics' },
      { href: '/calendar', i18nKey: 'pages.calendar', icon: CalendarDays, description: 'Civic calendar — sessions, committees, gazette dates' },
      { href: '/graph', i18nKey: 'pages.graph', icon: Network, description: 'Trace how Bills, Acts, People, and Institutions are connected' },
      { href: '/compare', i18nKey: 'pages.compare', icon: Globe, description: 'Cross-country civic comparison — legislation, debt, indicators' },
      { href: '/indicators', i18nKey: 'pages.indicators', icon: BarChart3, description: 'Civic indicators dashboard — Bills, Acts, debt, sessions' },
      { href: '/research', i18nKey: 'pages.research', icon: FileSearch, description: 'Research missions and reports' },
      { href: '/trust', i18nKey: 'pages.trust', icon: ShieldCheck, description: 'Trust and verification network' },
      { href: '/report', i18nKey: 'pages.reports', icon: FileBarChart, description: 'Generated civic reports' },
    ],
  },
  {
    i18nKey: 'groups.scenarios',
    icon: Sparkles,
    items: [
      { href: '/scenarios', i18nKey: 'pages.what_if', icon: Sparkles, description: 'Explore hypothetical civic scenarios — HYPOTHETICAL, not observed fact' },
    ],
  },
  {
    i18nKey: 'groups.resources',
    icon: Database,
    items: [
      { href: '/briefing', i18nKey: 'pages.daily_briefing', icon: Newspaper, description: 'Today\'s civic brief' },
      { href: '/gazette', i18nKey: 'pages.kenya_gazette', icon: FileText, description: 'Official gazette notices' },
      { href: '/gazette/alerts', i18nKey: 'pages.gazette_alerts', icon: BellRing, description: 'Subscribe to keywords in the Kenya Gazette' },
      { href: '/datasets', i18nKey: 'pages.datasets', icon: Database, description: 'Downloadable civic datasets' },
      { href: '/dashboard', i18nKey: 'pages.dashboard', icon: LayoutDashboard, description: 'Your civic dashboard' },
      { href: '/topics', i18nKey: 'pages.topics', icon: BookOpen, description: 'Browse by topic' },
      { href: '/developers', i18nKey: 'pages.developers', icon: Code2, description: 'API docs and developer platform' },
      { href: '/about', i18nKey: 'pages.about', icon: Info, description: 'About Civic Intelligence' },
      { href: '/sponsor', i18nKey: 'pages.sponsor', icon: Heart, description: 'Support the platform' },
    ],
  },
];

interface PrimaryNavItem {
  href: string;
  i18nKey: 'home' | 'ask' | 'briefing' | 'countries';
}

const primaryNav: PrimaryNavItem[] = [
  { href: '/', i18nKey: 'home' },
  { href: '/ask', i18nKey: 'ask' },
  { href: '/briefing', i18nKey: 'briefing' },
  { href: '/countries', i18nKey: 'countries' },
];

export function Header() {
  const t = useTranslations('nav');
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

  // Spec §37 — global Cmd+K shortcut listener. The actual palette is
  // mounted in layout.tsx; this hook only dispatches the open event so
  // the palette can also be triggered from anywhere (even outside the
  // Header). The palette installs its own shortcut listener too, so
  // this is primarily for surfaces where the Header is absent.
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        // The CommandPalette has its own listener that calls preventDefault
        // and toggles state. We don't dispatch the open event here to avoid
        // double-toggling; instead, the palette owns the keyboard shortcut.
        // (No-op — kept for documentation of the architecture.)
      }
    }
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, []);

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
            {/* Brand name stays English across all locales (product decision) */}
            Civic Intelligence
          </span>
        </Link>

        {/* Primary nav (center) — capped at 5 top-level items */}
        <nav aria-label="Primary" className="hidden lg:block">
          <ul className="flex items-center gap-7 text-sm font-medium text-civic-ink">
            {primaryNav.map((item) => (
              <li key={item.href}>
                {item.href === '/countries' ? (
                  <Link
                    href={item.href}
                    className="inline-flex items-center gap-1 py-3 hover:text-civic-leaf"
                  >
                    <Globe className="h-4 w-4" aria-hidden="true" />
                    {t(item.i18nKey)}
                  </Link>
                ) : (
                  <Link
                    href={item.href}
                    className="inline-flex items-center py-3 hover:text-civic-leaf"
                  >
                    {t(item.i18nKey)}
                  </Link>
                )}
              </li>
            ))}
            {/* Mega-menu Explore */}
            <li ref={exploreRef} className="relative">
              <button
                type="button"
                onClick={() => setExploreOpen((v) => !v)}
                aria-expanded={exploreOpen}
                aria-haspopup="true"
                className="inline-flex items-center gap-1 py-3 hover:text-civic-leaf"
              >
                {t('explore')}
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
                  aria-label={t('explore')}
                  className="absolute left-1/2 top-full z-50 mt-1 w-[min(56rem,calc(100vw-2rem))] -translate-x-1/2 origin-top rounded-xl border border-civic-border bg-civic-paper shadow-2xl"
                >
                  <div className="grid grid-cols-3 gap-x-6 gap-y-1 p-5">
                    {navGroups.map((group) => (
                      <div key={group.i18nKey} className="space-y-1">
                        <div className="flex items-center gap-1.5 px-2 pb-1 pt-2">
                          <group.icon className="h-3.5 w-3.5 text-civic-leaf" aria-hidden="true" />
                          <span className="text-[11px] font-semibold uppercase tracking-wider text-civic-stone">
                            {t(group.i18nKey)}
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
                                  <span className="block font-medium">{t(item.i18nKey)}</span>
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
          </ul>
        </nav>

        {/* Right cluster: expandable search + user actions */}
        <div className="flex flex-shrink-0 items-center gap-2 sm:gap-3">
          {/* Command palette trigger (Cmd+K) — spec §37 */}
          <button
            type="button"
            onClick={openCommandPalette}
            aria-label="Open command palette"
            title="Open command palette (⌘K)"
            className="hidden h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist sm:inline-flex"
          >
            <CommandIcon className="h-5 w-5" aria-hidden="true" />
          </button>

          {/* Expandable search bar */}
          <form action="/search" className="flex items-center" role="search">
            <label htmlFor="nav-search" className="sr-only">
              {t('search_label')}
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
                placeholder={t('search_placeholder')}
                aria-label={t('search_label')}
                autoComplete="off"
                onFocus={() => setSearchExpanded(true)}
                onBlur={() => setSearchExpanded(false)}
                className="w-full bg-transparent px-2 py-2 text-sm placeholder:text-civic-stone focus:outline-none"
              />
            </div>
            <Link
              href="/search"
              aria-label={t('search_label')}
              className="inline-flex h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist sm:hidden"
            >
              <Search className="h-5 w-5" aria-hidden="true" />
            </Link>
          </form>

          {/* Following / Bookmarks */}
          <Link
            href="/following"
            aria-label={t('following')}
            title={t('following')}
            className="hidden h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist sm:inline-flex"
          >
            <Bookmark className="h-5 w-5" aria-hidden="true" />
          </Link>

          {/* Notifications */}
          <Link
            href="/notifications"
            aria-label={t('notifications')}
            title={t('notifications')}
            className="relative inline-flex h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist"
          >
            <Bell className="h-5 w-5" aria-hidden="true" />
          </Link>

          {/* Language switcher — spec §66 */}
          <LanguageSwitcher />

          {/* Dark/Light theme toggle */}
          <ThemeToggle />

          {/* Sign In — links to /auth/signin (issue #108). Real auth is
              handled via Keycloak OIDC; the page renders a placeholder form
              that explains the flow. */}
          <Link
            href="/auth/signin"
            aria-label="Sign in"
            title="Sign in"
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-full border border-civic-border px-3 text-sm font-medium text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf"
          >
            <LogIn className="h-4 w-4" aria-hidden="true" />
            <span className="hidden sm:inline">Sign in</span>
          </Link>

          {/* Sponsor */}
          <Link
            href="/sponsor"
            className="hidden items-center gap-1.5 rounded-full bg-civic-forest px-4 py-2.5 text-sm font-medium text-civic-paper hover:bg-civic-leaf sm:inline-flex"
          >
            <Heart className="h-4 w-4" aria-hidden="true" />
            <span className="hidden md:inline">{t('sponsor')}</span>
          </Link>

          {/* Hamburger (mobile) */}
          <button
            type="button"
            onClick={() => setMobileOpen(true)}
            className="inline-flex h-10 w-10 items-center justify-center rounded-md text-civic-forest hover:bg-civic-mist lg:hidden"
            aria-label={t('open_menu')}
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
          aria-label={t('open_menu')}
        >
          <div
            className="absolute inset-0 bg-civic-ink/40 backdrop-blur-sm"
            onClick={() => setMobileOpen(false)}
            aria-hidden="true"
          />
          <div className="absolute right-0 top-0 flex h-full w-80 max-w-[85vw] flex-col bg-civic-paper shadow-xl">
            <div className="flex items-center justify-between border-b border-civic-border px-4 py-4">
              <span className="font-serif text-lg font-semibold text-civic-forest">
                {/* "Menu" label stays English across all locales (product decision) */}
                Menu
              </span>
              <button
                type="button"
                onClick={() => setMobileOpen(false)}
                className="inline-flex h-9 w-9 items-center justify-center rounded-md hover:bg-civic-mist"
                aria-label={t('close_menu')}
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
                      {t(item.i18nKey)}
                    </Link>
                  </li>
                ))}
              </ul>
              {/* Grouped explore links */}
              {navGroups.map((group) => (
                <div key={group.i18nKey} className="mt-5">
                  <div className="flex items-center gap-1.5 px-3 pb-1">
                    <group.icon className="h-3.5 w-3.5 text-civic-leaf" aria-hidden="true" />
                    <span className="text-xs font-semibold uppercase tracking-wider text-civic-stone">
                      {t(group.i18nKey)}
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
                          {t(item.i18nKey)}
                        </Link>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </nav>
            <div className="space-y-2 border-t border-civic-border px-4 py-4">
              <Link
                href="/auth/signin"
                onClick={() => setMobileOpen(false)}
                className="flex items-center justify-center gap-2 rounded-md border border-civic-border bg-civic-paper px-4 py-3 text-sm font-medium text-civic-ink hover:border-civic-leaf hover:text-civic-leaf"
              >
                <LogIn className="h-4 w-4" aria-hidden="true" />
                Sign in
              </Link>
              <Link
                href="/sponsor"
                onClick={() => setMobileOpen(false)}
                className="flex items-center justify-center gap-2 rounded-md bg-civic-forest px-4 py-3 text-sm font-medium text-civic-paper hover:bg-civic-leaf"
              >
                <Heart className="h-4 w-4" aria-hidden="true" />
                {t('sponsor')}
              </Link>
              <div className="grid grid-cols-2 gap-2">
                <Link
                  href="/notifications"
                  onClick={() => setMobileOpen(false)}
                  className="flex items-center justify-center gap-2 rounded-md border border-civic-border px-4 py-3 text-sm text-civic-ink hover:bg-civic-mist"
                >
                  <BellRing className="h-4 w-4" aria-hidden="true" />
                  {t('notifications')}
                </Link>
                <Link
                  href="/following"
                  onClick={() => setMobileOpen(false)}
                  className="flex items-center justify-center gap-2 rounded-md border border-civic-border px-4 py-3 text-sm text-civic-ink hover:bg-civic-mist"
                >
                  <User className="h-4 w-4" aria-hidden="true" />
                  {t('following')}
                </Link>
              </div>
            </div>
          </div>
        </div>
      )}
    </header>
  );
}
