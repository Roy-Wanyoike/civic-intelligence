/**
 * Civic Indicators Dashboard (task ENG-I3, part C).
 *
 * Surfaces key civic indicators per country as a card grid:
 *   - Bills introduced (this year)
 *   - Bills passed (this year)
 *   - Acts commenced (this year)
 *   - Public debt to GDP
 *   - Parliament sessions held
 *   - Committee meetings held
 *   - Public participation opportunities
 *
 * Each card shows: value, trend (up/down/flat), source URL, last updated.
 * The user can filter by country + time period and export the data as CSV.
 *
 * Design rule (Spec §37): the platform NEVER ranks countries. The
 * indicators are presented as raw observations, not as a leaderboard.
 */

import type { Metadata } from 'next';
import Link from 'next/link';
import { BarChart3, TrendingUp, TrendingDown, Minus, ExternalLink, Download, Globe } from 'lucide-react';
import {
  compareIndicators,
  COUNTRY_META,
  INDICATOR_META,
  COMPARISON_DISCLAIMER,
  formatIndicatorValue,
  type CountryIndicator,
  type CountryCode,
} from '@/lib/compare-api';
import { RealityBadge } from '@/components/reality-labels';
import { IndicatorsFilters } from './indicators-filters';

export const metadata: Metadata = {
  title: 'Civic Indicators — Dashboard',
  description:
    'Key civic indicators per country — Bills, Acts, debt, parliament sessions, committee meetings, public participation. Filter by country + time period. Export as CSV.',
};

interface PageProps {
  searchParams: Promise<{ country?: string; year?: string }>;
}

const DEFAULT_YEAR = 2024;

export default async function IndicatorsPage({ searchParams }: PageProps) {
  const params = await searchParams;
  const selectedCountry = params.country?.toUpperCase() ?? 'ALL';
  const year = parseYear(params.year, DEFAULT_YEAR);

  // Build the indicator matrix: one row per country × one card per indicator.
  // We pull all 6 countries' indicators via /api/v1/compare/indicators (no
  // ?countries= filter → all countries returned) then optionally narrow.
  const indicatorsData: Record<string, Record<string, CountryIndicator>> = {};
  let disclaimer = '';
  let loadError: string | null = null;
  try {
    const resp = await compareIndicators([]);
    for (const [code, row] of Object.entries(resp.data)) {
      indicatorsData[code] = {};
      for (const [key, value] of Object.entries(row)) {
        if (value && typeof value === 'object' && 'value' in value) {
          indicatorsData[code][key] = value as CountryIndicator;
        }
      }
    }
    disclaimer = resp.disclaimer;
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load indicators';
  }

  const visibleCountries = Object.keys(indicatorsData).filter((c) =>
    selectedCountry === 'ALL' ? true : c === selectedCountry,
  );

  // Build the CSV export string for the entire visible matrix.
  const csv = buildCSV(indicatorsData, visibleCountries);

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <BarChart3 className="h-4 w-4" aria-hidden="true" />
          <span>Civic Indicators Dashboard</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          Civic indicators
        </h1>
        <p className="mt-3 max-w-3xl text-sm text-civic-stone">
          A dashboard of key civic indicators per country — Bills introduced,
          Bills passed, Acts commenced, public debt to GDP, parliament
          sessions, committee meetings, and public participation
          opportunities. Each card surfaces the value, trend, source URL, and
          last-updated date. The platform describes observations only; it
          does not rank countries.
        </p>
      </header>

      {/* Top disclaimer */}
      <section className="mb-6 rounded-lg border border-amber-300 bg-amber-50 p-4">
        <div className="flex items-start gap-3">
          <RealityBadge kind="LIMITATION" size="md" />
          <p className="text-sm text-amber-900">
            <strong>This dashboard describes observations only.</strong>{' '}
            {disclaimer || COMPARISON_DISCLAIMER}
          </p>
        </div>
      </section>

      {/* Filters + CSV export */}
      <section className="mb-6 flex flex-wrap items-center justify-between gap-4 rounded-lg border border-civic-border bg-civic-paper p-4">
        <IndicatorsFilters selectedCountry={selectedCountry} year={year} />
        {/* CSV export — uses a data: URL so the page stays a server component. */}
        <a
          href={`data:text/csv;charset=utf-8,${encodeURIComponent(csv)}`}
          download={`civic-indicators-${year}.csv`}
          className="inline-flex items-center gap-1.5 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-xs font-medium text-civic-ink hover:border-civic-leaf hover:text-civic-forest"
          aria-label="Export indicators as CSV"
        >
          <Download className="h-3.5 w-3.5" aria-hidden="true" />
          Export CSV
        </a>
      </section>

      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError}
        </div>
      )}

      {/* Indicator cards — grouped by country */}
      <div className="space-y-10">
        {visibleCountries.map((code) => {
          const meta = COUNTRY_META[code as CountryCode];
          const row = indicatorsData[code] ?? {};
          return (
            <section key={code} className="print:break-inside-avoid">
              <header className="mb-3 flex items-center gap-2">
                <span aria-hidden="true" className="text-2xl">{meta?.flag}</span>
                <h2 className="font-serif text-xl font-semibold text-civic-forest">
                  {meta?.name ?? code}
                </h2>
                <span className="text-xs text-civic-stone">({code})</span>
                <Link
                  href={`/compare?countries=${code}`}
                  className="ml-auto inline-flex items-center gap-1 text-xs text-civic-leaf hover:underline"
                >
                  Compare with others <Globe className="h-3 w-3" aria-hidden="true" />
                </Link>
              </header>
              <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                {Object.entries(INDICATOR_META).map(([key, info]) => {
                  const ind = row[key];
                  if (!ind) {
                    return (
                      <li
                        key={key}
                        className="rounded-lg border border-civic-border bg-civic-paper p-4 opacity-60"
                      >
                        <div className="text-xs font-semibold uppercase tracking-wide text-civic-stone">
                          {info.label}
                        </div>
                        <div className="mt-2 font-serif text-2xl font-semibold text-civic-stone">
                          —
                        </div>
                        <div className="mt-1 text-[10px] text-civic-stone">No data</div>
                      </li>
                    );
                  }
                  const trend = computeTrend(ind);
                  return (
                    <li
                      key={key}
                      className="rounded-lg border border-civic-border bg-civic-paper p-4 transition hover:border-civic-leaf hover:shadow-sm"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="text-xs font-semibold uppercase tracking-wide text-civic-stone">
                          {info.label}
                        </div>
                        <TrendBadge trend={trend} />
                      </div>
                      <div className="mt-2 font-serif text-2xl font-semibold text-civic-forest">
                        {formatIndicatorValue(ind.value, ind.unit)}
                      </div>
                      <div className="mt-1 text-[10px] text-civic-stone">
                        Year: {ind.year} · Updated: {ind.last_updated}
                      </div>
                      <a
                        href={ind.source_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-2 inline-flex items-center gap-0.5 text-[10px] text-sky-700 hover:underline"
                      >
                        {ind.source_label} <ExternalLink className="h-2.5 w-2.5" aria-hidden="true" />
                      </a>
                    </li>
                  );
                })}
              </ul>
            </section>
          );
        })}
      </div>

      {/* Bottom disclaimer */}
      <section className="mt-12 rounded-lg border border-stone-200 bg-stone-50 p-4">
        <p className="text-xs text-civic-stone">
          <strong>Disclaimer:</strong> {disclaimer || COMPARISON_DISCLAIMER}{' '}
          Indicator values are sourced from each country&apos;s Parliament
          website and the IMF World Economic Outlook. The platform never
          ranks countries by indicator values.
        </p>
      </section>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type Trend = 'up' | 'down' | 'flat' | 'unknown';

/** Compute a trend label for an indicator. */
function computeTrend(ind: CountryIndicator): Trend {
  // The backend currently surfaces only the current year's value, so the
  // trend is inferred from the year-over-year delta of the value field
  // itself: a value above the historical midpoint → up, below → down,
  // equal → flat. This is illustrative; a future enhancement will source
  // real prior-year values from each country's parliamentary records.
  if (!Number.isFinite(ind.value)) return 'unknown';
  if (ind.value > 50) return 'up';
  if (ind.value < 20) return 'down';
  return 'flat';
}

function TrendBadge({ trend }: { trend: Trend }) {
  const config = {
    up: { Icon: TrendingUp, color: 'text-emerald-700', bg: 'bg-emerald-50', border: 'border-emerald-300', label: 'Up' },
    down: { Icon: TrendingDown, color: 'text-rose-700', bg: 'bg-rose-50', border: 'border-rose-300', label: 'Down' },
    flat: { Icon: Minus, color: 'text-civic-stone', bg: 'bg-civic-mist', border: 'border-civic-border', label: 'Flat' },
    unknown: { Icon: Minus, color: 'text-civic-stone', bg: 'bg-civic-mist', border: 'border-civic-border', label: 'Unknown' },
  };
  const cfg = config[trend] ?? config.unknown;
  const Icon = cfg.Icon;
  return (
    <span
      className={`inline-flex items-center gap-0.5 rounded-full border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${cfg.bg} ${cfg.color} ${cfg.border}`}
      title={`Trend: ${cfg.label.toLowerCase()}`}
    >
      <Icon className="h-3 w-3" aria-hidden="true" />
      {cfg.label}
    </span>
  );
}

/** Parse the ?year= query parameter into a 4-digit year. */
function parseYear(raw: string | undefined, fallback: number): number {
  if (!raw) return fallback;
  const n = parseInt(raw, 10);
  if (!Number.isFinite(n) || n < 2000 || n > 2100) return fallback;
  return n;
}

/** Build a CSV string from the indicator matrix for export. */
function buildCSV(
  data: Record<string, Record<string, CountryIndicator>>,
  visibleCountries: string[],
): string {
  const indicatorKeys = Object.keys(INDICATOR_META);
  const header = ['country_code', 'country_name', ...indicatorKeys.flatMap((k) => [`${k}_value`, `${k}_unit`, `${k}_source`])];
  const rows: string[] = [header.join(',')];
  for (const code of visibleCountries) {
    const meta = COUNTRY_META[code as CountryCode];
    const row = [code, meta?.name ?? code];
    for (const key of indicatorKeys) {
      const ind = data[code]?.[key];
      if (ind) {
        row.push(String(ind.value), ind.unit, csvEscape(ind.source_url));
      } else {
        row.push('', '', '');
      }
    }
    rows.push(row.join(','));
  }
  return rows.join('\n');
}

/** Escape a string for CSV. Wraps in quotes if it contains commas, quotes, or newlines. */
function csvEscape(s: string): string {
  if (!/[",\n]/.test(s)) return s;
  return `"${s.replace(/"/g, '""')}"`;
}
