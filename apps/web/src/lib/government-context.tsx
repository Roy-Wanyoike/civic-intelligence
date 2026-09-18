'use client';

/**
 * Government Selector context (spec §11).
 *
 * Persists the user's selected country / administration / term in a cookie
 * (`civic_gov_selection`) and exposes it through React context so every
 * page — Bills, Acts, Constitution, Institutions, Debt, Research, Search —
 * can read + update the same government context.
 *
 * The cookie is read server-side in `app/layout.tsx` via `next/headers`
 * and passed in as `initialSelection` so the very first server render
 * already reflects the user's previous choice (no client-side flicker).
 * Subsequent updates are written from the client.
 *
 * The shared type, defaults, and cookie name live in
 * `lib/government-defaults.ts` so both this client module and the
 * server-side `app/layout.tsx` can import them without dragging the
 * `'use client'` boundary into the server tree.
 */

import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from 'react';
import {
  DEFAULT_SELECTION,
  GOVERNMENT_COOKIE,
  type GovernmentSelection,
} from './government-defaults';

// Re-export so existing imports (`from '@/lib/government-context'`) keep
// working — the canonical definitions now live in government-defaults.ts.
export { DEFAULT_SELECTION, GOVERNMENT_COOKIE, type GovernmentSelection };

interface GovernmentContextValue {
  selection: GovernmentSelection;
  /** Merge a partial update into the current selection and persist to cookie. */
  setSelection: (next: Partial<GovernmentSelection>) => void;
}

const GovernmentContext = createContext<GovernmentContextValue | null>(null);

function writeCookie(value: GovernmentSelection): void {
  if (typeof document === 'undefined') return;
  const expires = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toUTCString();
  const raw = encodeURIComponent(JSON.stringify(value));
  // SameSite=Lax so the cookie is sent on top-level navigations but not
  // leaked to third-party origins. No `Secure` flag in dev (HTTP); in
  // production the platform terminates TLS at the edge so the cookie is
  // transported over HTTPS regardless.
  document.cookie = `${GOVERNMENT_COOKIE}=${raw}; expires=${expires}; path=/; SameSite=Lax`;
}

export function GovernmentProvider({
  children,
  initialSelection,
}: {
  children: ReactNode;
  initialSelection?: Partial<GovernmentSelection>;
}) {
  const [selection, setSelectionState] = useState<GovernmentSelection>(() => ({
    ...DEFAULT_SELECTION,
    ...initialSelection,
  }));

  const setSelection = useCallback((next: Partial<GovernmentSelection>) => {
    setSelectionState((prev) => {
      const merged: GovernmentSelection = { ...prev, ...next };
      writeCookie(merged);
      return merged;
    });
  }, []);

  return (
    <GovernmentContext.Provider value={{ selection, setSelection }}>
      {children}
    </GovernmentContext.Provider>
  );
}

export function useGovernment(): GovernmentContextValue {
  const ctx = useContext(GovernmentContext);
  if (!ctx) {
    throw new Error('useGovernment must be used inside <GovernmentProvider>');
  }
  return ctx;
}
