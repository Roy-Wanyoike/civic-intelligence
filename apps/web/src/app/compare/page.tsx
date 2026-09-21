/**
 * Compare page (server component) — Cross-country civic comparison.
 *
 * Loads the comparison data server-side (profiles, debt data, indicators)
 * and passes it to the CompareView client component for interactivity.
 *
 * The page reads `?countries=KE,UG,TZ` from the URL so users can deep-link
 * to a pre-selected comparison.
 */

import type { Metadata } from 'next';
import { Globe, ExternalLink } from 'lucide-react';
import {
  compareCountries,
  compareDebt,
  compareIndicators,
  COUNTRY_META,
  type CountryProfile,
  type DebtComparisonRow,
  type CountryIndicator,
  type CompareDifference,
} from '@/lib/compare-api';
import { CompareView } from './compare-view';

export const metadata: Metadata = {
  title: 'Compare — Cross-country Civic Comparison',
  description:
    'Compare legislation, government structure, public debt, and civic indicators across the 6 supported African countries. The platform describes differences only; it does not rank countries.',
};

// Default country selection when none is supplied via the URL.
const DEFAULT_COUNTRIES = ['KE', 'UG', 'TZ'];

interface PageProps {
  searchParams: Promise<{ countries?: string }>;
}

export default async function ComparePage({ searchParams }: PageProps) {
  const params = await searchParams;
  const codes = parseCountries(params.countries);

  // Fetch all comparison data in parallel. Each call returns the full
  // comparison payload; we slice the relevant fields per dimension.
  let profiles: Record<string, CountryProfile> = {};
  let debtData: Record<string, DebtComparisonRow> = {};
  const indicatorsData: Record<string, Record<string, CountryIndicator>> = {};
  let govDifferences: CompareDifference[] = [];
  let debtDifferences: CompareDifference[] = [];
  let disclaimer = '';
  let loadError: string | null = null;

  try {
    const [countriesResp, debtResp, indicatorsResp] = await Promise.all([
      compareCountries(codes),
      compareDebt(codes, 2020, 2024),
      compareIndicators(codes),
    ]);
    profiles = countriesResp.data as Record<string, CountryProfile>;
    govDifferences = countriesResp.differences ?? [];
    debtData = debtResp.data as Record<string, DebtComparisonRow>;
    debtDifferences = debtResp.differences ?? [];
    // Build a per-country indicator map: { KE: { bills_introduced: {...}, ... }, ... }
    for (const code of Object.keys(indicatorsResp.data)) {
      const row = indicatorsResp.data[code] as Record<string, unknown>;
      indicatorsData[code] = {} as Record<string, CountryIndicator>;
      for (const [key, value] of Object.entries(row)) {
        if (value && typeof value === 'object') {
          indicatorsData[code][key] = value as CountryIndicator;
        }
      }
    }
    disclaimer = indicatorsResp.disclaimer ?? countriesResp.disclaimer ?? debtResp.disclaimer ?? '';
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load comparison data';
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <header className="mb-8 print:mb-4">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Globe className="h-4 w-4" aria-hidden="true" />
          <span>Cross-country Civic Comparison</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          Compare countries
        </h1>
        <p className="mt-3 max-w-3xl text-sm text-civic-stone">
          Compare legislation, government structure, public debt, and civic
          indicators across the 6 supported African countries — Kenya,
          Uganda, Tanzania, Ghana, Nigeria, and South Africa. The platform
          describes <strong>differences only</strong>; it does not rank
          countries or imply political preference.
        </p>
        <p className="mt-2 text-xs text-civic-stone">
          Why this is unique: no platform in Africa offers side-by-side
          comparison of civic data across multiple countries with the same
          structured schema. This enables cross-jurisdictional research that
          was previously impossible.{' '}
          <a
            href="https://github.com/Roy-Wanyoike/civic-intelligence"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-0.5 text-sky-700 hover:underline"
          >
            View the schema <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        </p>
      </header>

      {loadError && (
        <div className="mb-6 rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}

      <CompareView
        initialCountries={codes}
        profiles={profiles}
        differences={govDifferences.length > 0 ? govDifferences : debtDifferences}
        debtData={debtData}
        indicatorsData={indicatorsData}
        disclaimer={disclaimer}
      />

      {/* Footer note: country list */}
      <section className="mt-12 border-t border-civic-border pt-6 text-xs text-civic-stone">
        <h2 className="font-serif text-sm font-semibold text-civic-forest">
          Supported countries
        </h2>
        <ul className="mt-2 flex flex-wrap gap-3">
          {Object.entries(COUNTRY_META).map(([code, meta]) => (
            <li key={code} className="inline-flex items-center gap-1.5">
              <span aria-hidden="true">{meta.flag}</span>
              <span className="font-medium text-civic-ink">{meta.name}</span>
              <span className="text-civic-stone">({code})</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}

/** Parse the ?countries= query parameter into a list of country codes. */
function parseCountries(raw?: string): string[] {
  if (!raw) return DEFAULT_COUNTRIES;
  const parts = raw.split(',').map((p) => p.trim().toUpperCase()).filter(Boolean);
  const valid = parts.filter((p) => COUNTRY_META[p as keyof typeof COUNTRY_META] != null);
  if (valid.length < 2) return DEFAULT_COUNTRIES;
  return valid;
}
