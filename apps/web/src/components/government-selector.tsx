'use client';

/**
 * Government Selector (spec §11).
 *
 * Renders a persistent breadcrumb + dropdown that lets the user switch
 * the active country / administration / term. The selection is persisted
 * via the `useGovernment()` context provider (cookie-backed) so every
 * downstream page (Bills, Acts, Constitution, Debt, Research, Search…)
 * reads from the same source of truth.
 *
 * Mobile:  `Kenya · Uhuru Kenyatta · Term 2`
 * Desktop: `Kenya / Uhuru Kenyatta Administration / Term 2`
 *
 * The dropdown has TWO sections:
 *   1. Country picker — lists all SUPPORTED_COUNTRIES (Kenya, Uganda,
 *      Tanzania, Ghana, Nigeria, South Africa) with flag emojis. Selecting
 *      a country updates `selection.countryCode` + `countryName` so EVERY
 *      page switches to that country's data on the next render. The
 *      country list is sourced from `lib/government-defaults.ts` so the
 *      first server-render is instant.
 *   2. Administration picker — lazily fetches `/api/v1/governments` on
 *      first open and lists administrations for the currently selected
 *      country. Selecting one updates the administration context.
 *
 * Data isolation contract (see CONTRIBUTING.md "Adding a New Country"):
 *   - Switching country updates the cookie so EVERY downstream page reads
 *     the same country context on next render.
 *   - The administration list is filtered by the active country — the user
 *     cannot accidentally pick a Kenyan administration while viewing
 *     Uganda.
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import Link from 'next/link';
import { ChevronDown, Landmark, MapPin } from 'lucide-react';
import { useGovernment } from '@/lib/government-context';
import {
  SUPPORTED_COUNTRIES,
  findCountry,
  type SupportedCountry,
} from '@/lib/government-defaults';
import {
  listAdministrations,
  type AdministrationListItem,
} from '@/lib/government-api';

export function GovernmentSelector() {
  const { selection, setSelection } = useGovernment();
  const [open, setOpen] = useState(false);
  const [admins, setAdmins] = useState<AdministrationListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  // Lazy-load administrations on first open; keep the cached list so
  // subsequent opens are instant. The list is filtered client-side by the
  // active country code so the user sees ONLY their country's
  // administrations — never a foreign country's.
  useEffect(() => {
    if (!open || admins.length > 0 || loading) return;
    setLoading(true);
    setError(null);
    listAdministrations()
      .then((r) => setAdmins(r.administrations ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
      .finally(() => setLoading(false));
  }, [open, admins.length, loading]);

  // Close on outside click + Escape.
  useEffect(() => {
    if (!open) return;
    function onDoc(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    function onEsc(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('mousedown', onDoc);
    document.addEventListener('keydown', onEsc);
    return () => {
      document.removeEventListener('mousedown', onDoc);
      document.removeEventListener('keydown', onEsc);
    };
  }, [open]);

  // Active country metadata — resolved from SUPPORTED_COUNTRIES by code.
  // Falls back to a minimal stub if the selection's countryCode is unknown
  // (shouldn't happen for end users, but keeps the UI safe during migrations).
  const activeCountry: SupportedCountry = useMemo(
    () =>
      findCountry(selection.countryCode) ?? {
        code: selection.countryCode,
        name: selection.countryName || selection.countryCode,
        flagEmoji: '🏳️',
        parliamentName: selection.countryName || 'Parliament',
        legislatureType: 'unicameral',
      },
    [selection.countryCode, selection.countryName],
  );

  // Administrations filtered to the active country. This enforces the data
  // isolation invariant: the user cannot pick a Kenyan administration while
  // viewing Uganda — the dropdown simply shows no entries for foreign
  // countries. When the user switches countries, the previous admin list
  // is filtered out automatically.
  const countryAdmins = useMemo(
    () => admins.filter((a) => a.country_code === selection.countryCode),
    [admins, selection.countryCode],
  );

  function selectCountry(country: SupportedCountry) {
    // Switching country resets the administration context: the user has not
    // yet picked an administration for the new country, so we clear the
    // administrationId/administrationName/presidentName. The country page
    // for the new country will show its current administration by default.
    setSelection({
      countryCode: country.code,
      countryName: country.name,
      administrationId: '',
      administrationName: '',
      presidentName: '',
      termNumber: null,
    });
    setOpen(false);
  }

  function selectAdmin(a: AdministrationListItem) {
    setSelection({
      administrationId: a.id,
      administrationName: a.name,
      presidentName: a.president_name,
    });
    setOpen(false);
  }

  const termLabel =
    selection.termNumber != null ? `Term ${selection.termNumber}` : null;

  return (
    <div
      className="border-b border-civic-forest/40 bg-civic-forest text-civic-mist"
      data-testid="government-selector"
      data-country-code={selection.countryCode}
    >
      <div className="mx-auto flex max-w-7xl items-center gap-2 px-4 py-2 sm:px-6 lg:px-8">
        <MapPin className="h-3.5 w-3.5 flex-shrink-0 text-civic-acacia" aria-hidden="true" />

        {/* Mobile breadcrumb — compact `·` separators */}
        <nav
          aria-label="Active government context"
          className="flex min-w-0 flex-1 items-center gap-1.5 text-xs sm:hidden"
        >
          <span className="truncate font-medium text-white">
            {activeCountry.flagEmoji} {selection.countryName}
          </span>
          {selection.presidentName ? (
            <>
              <span aria-hidden="true" className="text-white/40">·</span>
              <span className="truncate text-white/90">{selection.presidentName}</span>
            </>
          ) : null}
          {termLabel && (
            <>
              <span aria-hidden="true" className="text-white/40">·</span>
              <span className="text-white/80">{termLabel}</span>
            </>
          )}
        </nav>

        {/* Desktop breadcrumb — `/` separators + administration name */}
        <nav
          aria-label="Active government context"
          className="hidden min-w-0 flex-1 items-center gap-2 text-sm sm:flex"
        >
          <Link
            href={`/country/${selection.countryCode.toLowerCase()}`}
            className="font-medium text-white hover:text-civic-acacia"
          >
            {activeCountry.flagEmoji} {selection.countryName}
          </Link>
          {selection.administrationName ? (
            <>
              <span aria-hidden="true" className="text-white/40">/</span>
              <span className="truncate text-white/90">{selection.administrationName}</span>
              {termLabel && (
                <>
                  <span aria-hidden="true" className="text-white/40">/</span>
                  <span className="text-white/80">{termLabel}</span>
                </>
              )}
            </>
          ) : (
            <>
              <span aria-hidden="true" className="text-white/40">/</span>
              <span className="truncate text-white/60 italic">
                Pick an administration ↓
              </span>
            </>
          )}
        </nav>

        {/* Dropdown trigger */}
        <div ref={ref} className="relative flex-shrink-0">
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            aria-haspopup="listbox"
            aria-label="Switch administration"
            className="inline-flex items-center gap-1.5 rounded-full border border-white/20 px-3 py-1 text-xs font-medium text-white transition hover:border-civic-acacia hover:bg-white/5"
          >
            <Landmark className="h-3.5 w-3.5" aria-hidden="true" />
            <span className="hidden sm:inline">Switch government</span>
            <span className="sm:hidden">Switch</span>
            <ChevronDown
              className={`h-3 w-3 transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
              aria-hidden="true"
            />
          </button>

          {open && (
            <div
              role="listbox"
              aria-label="Available countries and administrations"
              className="absolute right-0 top-full z-50 mt-1 w-80 max-w-[calc(100vw-2rem)] rounded-lg border border-civic-border bg-civic-paper text-civic-ink shadow-xl"
            >
              {/* Section 1 — Country picker */}
              <div className="border-b border-civic-border px-3 py-2">
                <p className="text-[11px] font-semibold uppercase tracking-wider text-civic-stone">
                  Country
                </p>
              </div>
              <ul className="max-h-56 overflow-y-auto py-1 text-sm">
                {SUPPORTED_COUNTRIES.map((c) => {
                  const isSelected = c.code === selection.countryCode;
                  return (
                    <li key={c.code}>
                      <button
                        type="button"
                        role="option"
                        aria-selected={isSelected}
                        onClick={() => selectCountry(c)}
                        className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left transition hover:bg-civic-mist"
                      >
                        <span className="flex items-center gap-2">
                          <span aria-hidden="true" className="text-base leading-none">
                            {c.flagEmoji}
                          </span>
                          <span className="font-medium">{c.name}</span>
                        </span>
                        <span className="flex items-center gap-1.5">
                          <span className="text-[10px] text-civic-stone">
                            {c.legislatureType === 'bicameral' ? 'Bicameral' : 'Unicameral'}
                          </span>
                          {isSelected && (
                            <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 text-[10px] font-semibold text-civic-leaf">
                              Selected
                            </span>
                          )}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>

              {/* Section 2 — Administration picker (scoped to active country) */}
              <div className="border-t border-civic-border px-3 py-2">
                <p className="text-[11px] font-semibold uppercase tracking-wider text-civic-stone">
                  {activeCountry.flagEmoji} {activeCountry.parliamentName} — Administrations
                </p>
              </div>
              <ul className="max-h-72 overflow-y-auto py-1 text-sm">
                {loading && (
                  <li className="px-3 py-2 text-civic-stone">Loading…</li>
                )}
                {error && !loading && (
                  <li className="px-3 py-2 text-rose-600">
                    Couldn&apos;t load administrations: {error}
                  </li>
                )}
                {!loading && !error && countryAdmins.length === 0 && (
                  <li className="px-3 py-2 text-civic-stone">
                    No administrations seeded for {activeCountry.name} yet.
                  </li>
                )}
                {!loading &&
                  !error &&
                  countryAdmins.map((a) => {
                    const isSelected = selection.administrationId
                      ? selection.administrationId === a.id
                      : a.is_current;
                    return (
                      <li key={a.id}>
                        <button
                          type="button"
                          role="option"
                          aria-selected={isSelected}
                          onClick={() => selectAdmin(a)}
                          className="block w-full px-3 py-2 text-left transition hover:bg-civic-mist"
                        >
                          <div className="flex items-center justify-between gap-2">
                            <span className="font-medium">{a.name}</span>
                            {isSelected && (
                              <span className="rounded-full bg-civic-leaf/10 px-2 py-0.5 text-[10px] font-semibold text-civic-leaf">
                                Selected
                              </span>
                            )}
                          </div>
                          <p className="mt-0.5 text-xs text-civic-stone">
                            {a.president_name} · {new Date(a.start_date).getFullYear()}
                            {a.end_date
                              ? `–${new Date(a.end_date).getFullYear()}`
                              : '–Present'}
                            {a.is_current ? ' · Current' : ''}
                          </p>
                        </button>
                      </li>
                    );
                  })}
              </ul>
              <div className="border-t border-civic-border px-3 py-2 text-xs">
                <Link
                  href="/governments"
                  className="text-civic-leaf hover:underline"
                >
                  Browse all governments →
                </Link>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
