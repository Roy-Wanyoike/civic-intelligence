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
 * The dropdown fetches `/api/v1/governments` lazily on first open.
 */

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { ChevronDown, Landmark, MapPin } from 'lucide-react';
import { useGovernment } from '@/lib/government-context';
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
  // subsequent opens are instant.
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
    >
      <div className="mx-auto flex max-w-7xl items-center gap-2 px-4 py-2 sm:px-6 lg:px-8">
        <MapPin className="h-3.5 w-3.5 flex-shrink-0 text-civic-acacia" aria-hidden="true" />

        {/* Mobile breadcrumb — compact `·` separators */}
        <nav
          aria-label="Active government context"
          className="flex min-w-0 flex-1 items-center gap-1.5 text-xs sm:hidden"
        >
          <span className="truncate font-medium text-white">{selection.countryName}</span>
          <span aria-hidden="true" className="text-white/40">·</span>
          <span className="truncate text-white/90">{selection.presidentName}</span>
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
            {selection.countryName}
          </Link>
          <span aria-hidden="true" className="text-white/40">/</span>
          <span className="truncate text-white/90">{selection.administrationName}</span>
          {termLabel && (
            <>
              <span aria-hidden="true" className="text-white/40">/</span>
              <span className="text-white/80">{termLabel}</span>
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
            <span className="hidden sm:inline">Switch administration</span>
            <span className="sm:hidden">Switch</span>
            <ChevronDown
              className={`h-3 w-3 transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
              aria-hidden="true"
            />
          </button>

          {open && (
            <div
              role="listbox"
              aria-label="Available administrations"
              className="absolute right-0 top-full z-50 mt-1 w-80 max-w-[calc(100vw-2rem)] rounded-lg border border-civic-border bg-civic-paper text-civic-ink shadow-xl"
            >
              <div className="border-b border-civic-border px-3 py-2">
                <p className="text-[11px] font-semibold uppercase tracking-wider text-civic-stone">
                  Switch administration
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
                {!loading && !error && admins.length === 0 && (
                  <li className="px-3 py-2 text-civic-stone">
                    No administrations found.
                  </li>
                )}
                {!loading &&
                  !error &&
                  admins.map((a) => {
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
