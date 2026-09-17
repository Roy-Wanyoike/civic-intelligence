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
} from 'lucide-react';
import { ThemeToggle } from './theme-toggle';

/**
 * Explore dropdown — secondary destinations grouped under one primary slot
 * so the top-level nav stays sparse and breathable.
 */
const exploreLinks = [
  { href: '/bills', label: 'Bills' },
  { href: '/trending', label: 'Trending' },
  { href: '/loans', label: 'Loans' },
  { href: '/grants', label: 'Grants' },
  { href: '/acts', label: 'Acts' },
  { href: '/regulations', label: 'Regulations' },
  { href: '/policies', label: 'Policies' },
  { href: '/constitution', label: 'Constitution' },
  { href: '/governments', label: 'Governments' },
  { href: '/what-changed', label: 'What Changed' },
  { href: '/scenarios', label: 'What If?' },
  { href: '/feed', label: 'Feed' },
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/datasets', label: 'Datasets' },
  { href: '/developers', label: 'Developers' },
] as const;

const primaryNav = [
  { href: '/', label: 'Home' },
  { href: '/ask', label: 'Ask' },
  { href: '/topics', label: 'Topics' },
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
          <span className="font-serif text-lg font-semibold tracking-tight">
            Civic Intelligence
          </span>
        </Link>

        {/* Primary nav (center) — capped at 5 top-level items */}
        <nav aria-label="Primary" className="hidden lg:block">
          <ul className="flex items-center gap-8 text-sm font-medium text-civic-ink">
            <li>
              <Link
                href="/"
                className="inline-flex items-center py-3 hover:text-civic-leaf"
              >
                Home
              </Link>
            </li>
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
                  className="absolute left-0 top-full mt-1 w-64 origin-top-left rounded-lg border border-civic-border bg-civic-paper py-2 shadow-lg"
                >
                  <ul className="grid gap-0.5">
                    {exploreLinks.map((link) => (
                      <li key={link.href}>
                        <Link
                          href={link.href}
                          onClick={() => setExploreOpen(false)}
                          role="menuitem"
                          className="block px-4 py-2 text-sm text-civic-ink hover:bg-civic-mist hover:text-civic-leaf"
                        >
                          {link.label}
                        </Link>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </li>
            {primaryNav
              .filter((item) => item.label !== 'Home')
              .map((item) => (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    className="inline-flex items-center py-3 hover:text-civic-leaf"
                  >
                    {item.label}
                  </Link>
                </li>
              ))}
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

          {/* Sponsor */}
          <Link
            href="/sponsor"
            className="hidden items-center gap-1.5 rounded-full bg-civic-forest px-4 py-2.5 text-sm font-medium text-civic-paper hover:bg-civic-leaf sm:inline-flex"
          >
            <Heart className="h-4 w-4" aria-hidden="true" />
            <span className="hidden md:inline">Sponsor</span>
          </Link>

          {/* Notifications */}
          <Link
            href="/notifications"
            aria-label="Notifications"
            className="relative inline-flex h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist"
          >
            <Bell className="h-5 w-5" aria-hidden="true" />
          </Link>

          {/* Dark/Light theme toggle */}
          <ThemeToggle />

          {/* Account */}
          <Link
            href="/following"
            aria-label="Account"
            className="hidden h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist sm:inline-flex"
          >
            <User className="h-5 w-5" aria-hidden="true" />
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
              <p className="mt-6 px-3 text-xs font-semibold uppercase tracking-wider text-civic-stone">
                Explore
              </p>
              <ul className="mt-2 space-y-1">
                {exploreLinks.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      onClick={() => setMobileOpen(false)}
                      className="block rounded-md px-3 py-2.5 text-sm text-civic-ink hover:bg-civic-mist hover:text-civic-leaf"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
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
                  <Bell className="h-4 w-4" aria-hidden="true" />
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
