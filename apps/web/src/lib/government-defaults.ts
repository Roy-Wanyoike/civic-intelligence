/**
 * Government Selector defaults (spec §11).
 *
 * Extracted from `government-context.tsx` so server components
 * (`app/layout.tsx`) can read the constants without importing a
 * `'use client'` module — the React Server Components bundler treats
 * every property access on a client module export as a separate client
 * reference, which breaks when the export is a plain runtime object.
 *
 * Keep this file free of `'use client'` and free of React imports so it
 * can be safely imported from both server and client modules.
 */

export interface GovernmentSelection {
  /** ISO 3166-1 alpha-2 country code, e.g. `KE`. */
  countryCode: string;
  /** Human-readable country name, e.g. `Kenya`. */
  countryName: string;
  /** Administration UUID (matches services/api governments.id). May be empty
   *  when the user has not explicitly chosen one (defaults are used). */
  administrationId: string;
  /** Display name for the administration, e.g. `Uhuru Kenyatta Administration`. */
  administrationName: string;
  /** Head-of-government display name, e.g. `Uhuru Kenyatta`. */
  presidentName: string;
  /** Term number within the administration, e.g. `2`. `null` when unknown. */
  termNumber: number | null;
}

/**
 * The platform-wide default. Kenya is the first country adapter shipped, so
 * the most recent administration (William Ruto, Term 1) is the sensible
 * initial selection for first-time visitors.
 */
export const DEFAULT_SELECTION: GovernmentSelection = {
  countryCode: 'KE',
  countryName: 'Kenya',
  administrationId: '',
  administrationName: 'William Ruto Administration',
  presidentName: 'William Ruto',
  termNumber: 1,
};

/** Cookie name — kept stable across releases so existing sessions persist. */
export const GOVERNMENT_COOKIE = 'civic_gov_selection';
