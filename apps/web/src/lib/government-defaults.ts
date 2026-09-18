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

/**
 * SupportedCountry is the public-facing metadata for a country the platform
 * supports. Mirrors the Go-side `registry.CountryInfo` struct defined in
 * `adapters/registry/registry.go`. The government selector dropdown renders
 * these entries.
 *
 * Adding a new country:
 *   1. Append a row to `SUPPORTED_COUNTRIES` below.
 *   2. Implement the adapter in `adapters/{country}/` (see CONTRIBUTING.md).
 *   3. Register it in `adapters/registry` (it shows up in
 *      `GET /api/v1/countries` automatically).
 */
export interface SupportedCountry {
  /** ISO 3166-1 alpha-2 code, e.g. `KE`. */
  code: string;
  /** Human-readable name, e.g. `Kenya`. */
  name: string;
  /** Flag emoji for the country dropdown, e.g. `🇰🇪`. */
  flagEmoji: string;
  /** Name of the national legislature, e.g. `Parliament of Kenya`. */
  parliamentName: string;
  /** Either `"unicameral"` or `"bicameral"`. */
  legislatureType: 'unicameral' | 'bicameral';
}

/**
 * SUPPORTED_COUNTRIES is the static, build-time list of countries the
 * platform ships with. The Government Selector component renders this list
 * directly so the first server-side render is instant — no API round-trip
 * required for the dropdown to populate.
 *
 * The same list is also served dynamically from `GET /api/v1/countries`
 * (driven by the Go-side `registry.SupportedCountries()`), so the two
 * sources MUST stay in sync. When the API returns a country not in this
 * list, the selector falls back to the API's value (see
 * `government-selector.tsx` for the merge logic).
 *
 * Flag emojis are Unicode regional indicator symbols — no external assets
 * are loaded.
 */
export const SUPPORTED_COUNTRIES: readonly SupportedCountry[] = [
  {
    code: 'KE',
    name: 'Kenya',
    flagEmoji: '🇰🇪',
    parliamentName: 'Parliament of Kenya',
    legislatureType: 'bicameral',
  },
  {
    code: 'UG',
    name: 'Uganda',
    flagEmoji: '🇺🇬',
    parliamentName: 'Parliament of Uganda',
    legislatureType: 'unicameral',
  },
  {
    code: 'TZ',
    name: 'Tanzania',
    flagEmoji: '🇹🇿',
    parliamentName: 'Bunge la Tanzania',
    legislatureType: 'unicameral',
  },
  {
    code: 'GH',
    name: 'Ghana',
    flagEmoji: '🇬🇭',
    parliamentName: 'Parliament of Ghana',
    legislatureType: 'unicameral',
  },
  {
    code: 'NG',
    name: 'Nigeria',
    flagEmoji: '🇳🇬',
    parliamentName: 'National Assembly of Nigeria',
    legislatureType: 'bicameral',
  },
  {
    code: 'ZA',
    name: 'South Africa',
    flagEmoji: '🇿🇦',
    parliamentName: 'Parliament of South Africa',
    legislatureType: 'bicameral',
  },
];

/**
 * findCountry looks up a country by code in SUPPORTED_COUNTRIES. Returns
 * `undefined` when the code is not recognised — callers should fall back
 * to the API's list (see `government-selector.tsx`).
 */
export function findCountry(code: string): SupportedCountry | undefined {
  return SUPPORTED_COUNTRIES.find((c) => c.code === code);
}

