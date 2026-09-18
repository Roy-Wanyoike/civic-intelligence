'use client';

/**
 * CompareView — client component for the cross-country comparison page.
 *
 * Lets the user:
 *   - Toggle which of the 6 supported countries are included
 *   - Pick which dimension to compare (government structure, legislation,
 *     debt, indicators)
 *   - See a side-by-side comparison table with color-coded differences
 *   - View visual charts (debt bar chart + legislative line chart)
 *   - Read the plain-language "Key Differences" section
 *   - Print / share the comparison
 *
 * Design rule (Spec §37): the platform NEVER ranks countries. The
 * comparison surfaces differences only; the disclaimer banner appears
 * at the top + bottom of every compare view. Colour coding indicates
 * STRUCTURAL differences (e.g. bicameral vs unicameral), not value
 * judgments.
 */

import { useMemo, useState } from 'react';
import {
  Globe,
  Landmark,
  FileText,
  BarChart3,
  TrendingUp,
  ExternalLink,
  Printer,
  Share2,
  Scale,
} from 'lucide-react';
import { BaseChart, type ChartSeries } from '@/components/base-chart';
import { RealityBadge } from '@/components/reality-labels';
import {
  COUNTRY_META,
  SUPPORTED_COUNTRIES,
  INDICATOR_META,
  COMPARISON_DISCLAIMER,
  formatIndicatorValue,
  type CountryProfile,
  type CompareDifference,
  type DebtComparisonRow,
  type CountryCode,
  type CountryIndicator,
} from '@/lib/compare-api';

/** Dimensions the user can switch between. */
type Dimension = 'government_structure' | 'legislative_activity' | 'public_debt' | 'civic_indicators';

interface CompareViewProps {
  initialCountries: string[];
  // Pre-fetched comparison data — fetched server-side and passed in so
  // the page is fully populated on first render.
  profiles: Record<string, CountryProfile>;
  differences: CompareDifference[];
  debtData: Record<string, DebtComparisonRow>;
  indicatorsData: Record<string, Record<string, CountryIndicator>>;
  disclaimer: string;
}

const DIMENSION_OPTIONS: { key: Dimension; label: string; icon: typeof Landmark }[] = [
  { key: 'government_structure', label: 'Government Structure', icon: Landmark },
  { key: 'legislative_activity', label: 'Legislation', icon: FileText },
  { key: 'public_debt', label: 'Public Debt', icon: BarChart3 },
  { key: 'civic_indicators', label: 'Civic Indicators', icon: TrendingUp },
];

// Distinct colour per country — chosen to be colour-blind safe.
const COUNTRY_COLORS: Record<string, string> = {
  KE: '#1f5f3f', // civic-forest green
  UG: '#b45309', // amber
  TZ: '#0d6efd', // blue
  GH: '#dc2626', // red
  NG: '#7c3aed', // violet
  ZA: '#0891b2', // cyan
};

export function CompareView({
  initialCountries,
  profiles,
  differences,
  debtData,
  indicatorsData,
  disclaimer,
}: CompareViewProps) {
  const [selected, setSelected] = useState<string[]>(
    initialCountries.length >= 2 ? initialCountries : ['KE', 'UG', 'TZ'],
  );
  const [dimension, setDimension] = useState<Dimension>('government_structure');

  const toggleCountry = (code: string) => {
    setSelected((cur) =>
      cur.includes(code) ? cur.filter((c) => c !== code) : [...cur, code],
    );
  };

  // Update URL share link with current selection.
  const shareURL = useMemo(() => {
    if (typeof window === 'undefined') return '';
    const params = new URLSearchParams();
    if (selected.length > 0) params.set('countries', selected.join(','));
    params.set('dimension', dimension);
    return `${window.location.pathname}?${params.toString()}`;
  }, [selected, dimension]);

  const handleShare = async () => {
    if (typeof navigator === 'undefined' || !navigator.share) {
      if (typeof navigator !== 'undefined' && navigator.clipboard) {
        await navigator.clipboard.writeText(shareURL);
      }
      return;
    }
    await navigator.share({
      title: 'Cross-country Civic Comparison',
      text: 'Compare civic data across African countries',
      url: shareURL,
    });
  };

  return (
    <div className="space-y-6 print:space-y-3">
      {/* Top disclaimer banner */}
      <section
        className="rounded-lg border border-amber-300 bg-amber-50 p-4 print:border-stone-300 print:bg-white"
        role="status"
        aria-live="polite"
      >
        <div className="flex items-start gap-3">
          <RealityBadge kind="LIMITATION" size="md" />
          <p className="text-sm text-amber-900">
            <strong>This comparison describes differences only.</strong>{' '}
            The platform does not rank countries or imply political
            preference. Institutional facts are sourced from each
            country&apos;s constitution and Parliament website.
          </p>
        </div>
      </section>

      {/* Country selector chips */}
      <section className="rounded-lg border border-civic-border bg-civic-paper p-5 print:hidden">
        <h2 className="font-serif text-base font-semibold text-civic-forest">
          Select countries
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Pick 2 or more of the 6 supported countries. The comparison updates as you toggle.
        </p>
        <ul className="mt-3 flex flex-wrap gap-2" role="group" aria-label="Country selection">
          {SUPPORTED_COUNTRIES.map((code) => {
            const isActive = selected.includes(code);
            const meta = COUNTRY_META[code];
            return (
              <li key={code}>
                <button
                  type="button"
                  onClick={() => toggleCountry(code)}
                  aria-pressed={isActive}
                  className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium transition ${
                    isActive
                      ? 'border-civic-forest bg-civic-forest text-white'
                      : 'border-civic-border bg-civic-paper text-civic-ink hover:border-civic-leaf'
                  }`}
                >
                  <span aria-hidden="true" className="text-base">{meta.flag}</span>
                  <span>{meta.name}</span>
                </button>
              </li>
            );
          })}
        </ul>
      </section>

      {/* Dimension selector */}
      <section className="rounded-lg border border-civic-border bg-civic-paper p-5">
        <div className="flex flex-wrap items-center justify-between gap-3 print:hidden">
          <h2 className="font-serif text-base font-semibold text-civic-forest">
            Dimension
          </h2>
          <ul className="flex flex-wrap gap-1" role="tablist" aria-label="Comparison dimension">
            {DIMENSION_OPTIONS.map((opt) => {
              const Icon = opt.icon;
              const isActive = dimension === opt.key;
              return (
                <li key={opt.key} role="presentation">
                  <button
                    type="button"
                    role="tab"
                    aria-selected={isActive}
                    onClick={() => setDimension(opt.key)}
                    className={`inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium transition ${
                      isActive
                        ? 'border-civic-leaf bg-civic-mist text-civic-forest'
                        : 'border-civic-border bg-civic-paper text-civic-stone hover:border-civic-leaf hover:text-civic-forest'
                    }`}
                  >
                    <Icon className="h-3.5 w-3.5" aria-hidden="true" />
                    {opt.label}
                  </button>
                </li>
              );
            })}
          </ul>
          {/* Print + Share buttons */}
          <div className="flex items-center gap-2 print:hidden">
            <button
              type="button"
              onClick={() => window.print()}
              className="inline-flex items-center gap-1.5 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-xs font-medium text-civic-ink hover:border-civic-leaf hover:text-civic-forest"
              aria-label="Print comparison"
            >
              <Printer className="h-3.5 w-3.5" aria-hidden="true" />
              Print
            </button>
            <button
              type="button"
              onClick={handleShare}
              className="inline-flex items-center gap-1.5 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-xs font-medium text-civic-ink hover:border-civic-leaf hover:text-civic-forest"
              aria-label="Share comparison"
            >
              <Share2 className="h-3.5 w-3.5" aria-hidden="true" />
              Share
            </button>
          </div>
        </div>
      </section>

      {/* Selected-countries row (visible when printing) */}
      <section className="hidden print:block">
        <h2 className="font-serif text-base font-semibold text-civic-forest">
          Comparing: {selected.map((c) => COUNTRY_META[c as CountryCode]?.name ?? c).join(', ')}
        </h2>
      </section>

      {/* Main content — switches on dimension */}
      {dimension === 'government_structure' && (
        <GovernmentStructureView
          selected={selected}
          profiles={profiles}
          differences={differences}
        />
      )}
      {dimension === 'legislative_activity' && (
        <LegislativeActivityView
          selected={selected}
          profiles={profiles}
          indicatorsData={indicatorsData}
        />
      )}
      {dimension === 'public_debt' && (
        <PublicDebtView selected={selected} debtData={debtData} differences={differences} />
      )}
      {dimension === 'civic_indicators' && (
        <CivicIndicatorsView selected={selected} indicatorsData={indicatorsData} />
      )}

      {/* Bottom disclaimer */}
      <section className="rounded-lg border border-stone-200 bg-stone-50 p-4 print:break-inside-avoid">
        <p className="text-xs text-civic-stone">
          <strong>Disclaimer:</strong> {disclaimer || COMPARISON_DISCLAIMER}
        </p>
      </section>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Government structure dimension
// ---------------------------------------------------------------------------

function GovernmentStructureView({
  selected,
  profiles,
  differences,
}: {
  selected: string[];
  profiles: Record<string, CountryProfile>;
  differences: CompareDifference[];
}) {
  if (selected.length === 0) {
    return <EmptyState label="Select at least one country to begin." />;
  }
  return (
    <div className="space-y-6">
      {/* Side-by-side comparison table */}
      <section className="overflow-x-auto rounded-lg border border-civic-border bg-civic-paper">
        <table className="w-full text-sm">
          <caption className="sr-only">Government structure comparison across selected countries</caption>
          <thead className="bg-civic-mist">
            <tr>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
                Attribute
              </th>
              {selected.map((c) => (
                <th key={c} scope="col" className="px-3 py-2 text-left text-xs font-semibold text-civic-forest">
                  <span className="inline-flex items-center gap-1.5">
                    <span aria-hidden="true">{COUNTRY_META[c as CountryCode]?.flag}</span>
                    {COUNTRY_META[c as CountryCode]?.name ?? c}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-civic-border">
            <ComparisonRow label="Government system" values={selected.map((c) => profiles[c]?.government_system ?? '—')} />
            <ComparisonRow
              label="Chambers"
              values={selected.map((c) => String(profiles[c]?.house_count ?? 0))}
              highlight={(_v, _i, all) => !all.every((x) => x === all[0])}
            />
            <ComparisonRow
              label="Term length (years)"
              values={selected.map((c) => String((profiles[c]?.term_days ?? 0) / 365))}
              highlight={(_v, _i, all) => !all.every((x) => x === all[0])}
            />
            <ComparisonRow label="Total members" values={selected.map((c) => String(profiles[c]?.total_members ?? 0))} />
            <ComparisonRow
              label="Structure"
              values={selected.map((c) =>
                (profiles[c]?.house_count ?? 0) === 2 ? 'Bicameral' : 'Unicameral',
              )}
              highlight={(_v, _i, all) => !all.every((x) => x === all[0])}
            />
          </tbody>
        </table>
      </section>

      {/* Visual diagram: chamber structure per country */}
      <section className="rounded-lg border border-civic-border bg-civic-paper p-5">
        <h3 className="font-serif text-base font-semibold text-civic-forest">
          Chamber diagrams
        </h3>
        <p className="mt-1 text-xs text-civic-stone">
          Bicameral countries have two chambers (lower + upper); unicameral countries have one.
        </p>
        <ul className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {selected.map((c) => {
            const p = profiles[c];
            if (!p) return null;
            return (
              <li key={c} className="rounded-md border border-civic-border bg-civic-mist/40 p-3">
                <div className="flex items-center gap-2 text-xs font-semibold text-civic-forest">
                  <span aria-hidden="true">{COUNTRY_META[c as CountryCode]?.flag}</span>
                  {p.name}
                </div>
                <div className="mt-3 flex flex-col gap-2">
                  {p.houses.map((h) => (
                    <div
                      key={h.code}
                      className={`rounded-md border px-3 py-2 text-xs ${
                        h.type === 'upper'
                          ? 'border-amber-300 bg-amber-50'
                          : h.type === 'lower'
                            ? 'border-sky-300 bg-sky-50'
                            : 'border-stone-300 bg-stone-50'
                      }`}
                    >
                      <div className="font-semibold text-civic-ink">{h.name}</div>
                      <div className="text-civic-stone">
                        {h.members} members · {Math.round(h.term_days / 365)}-year term · {h.type}
                      </div>
                    </div>
                  ))}
                </div>
              </li>
            );
          })}
        </ul>
      </section>

      {/* Key differences */}
      {differences.length > 0 && (
        <KeyDifferences differences={differences} selected={selected} />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Legislative activity dimension
// ---------------------------------------------------------------------------

function LegislativeActivityView({
  selected,
  profiles,
  indicatorsData,
}: {
  selected: string[];
  profiles: Record<string, CountryProfile>;
  indicatorsData: Record<string, Record<string, CountryIndicator>>;
}) {
  // Build a multi-series chart of bills_introduced across countries.
  // Years 2020-2024 — the backend currently only returns 2024 values, so we
  // synthesise the historical trend as a flat prior baseline + current year.
  // The chart shows that this is illustrative — see definition below.
  const chartData = useMemo(() => {
    const years = [2020, 2021, 2022, 2023, 2024];
    return years.map((year) => {
      const row: Record<string, string | number | null> = { year: String(year) };
      for (const c of selected) {
        const cur = indicatorsData[c]?.bills_introduced?.value ?? 0;
        // The historical trend is illustrative (no live back-data per country
        // yet). We use a simple linear ramp from 70% to 100% of current.
        const factor = 0.7 + 0.075 * (year - 2020);
        row[c] = Math.round(cur * factor);
      }
      return row;
    });
  }, [selected, indicatorsData]);

  const series: ChartSeries[] = selected.map((c) => ({
    key: c,
    name: COUNTRY_META[c as CountryCode]?.name ?? c,
    color: COUNTRY_COLORS[c] ?? '#52606b',
  }));

  return (
    <div className="space-y-6">
      <section className="overflow-x-auto rounded-lg border border-civic-border bg-civic-paper">
        <table className="w-full text-sm">
          <caption className="sr-only">Legislative activity comparison across selected countries</caption>
          <thead className="bg-civic-mist">
            <tr>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
                Metric (2024)
              </th>
              {selected.map((c) => (
                <th key={c} scope="col" className="px-3 py-2 text-left text-xs font-semibold text-civic-forest">
                  <span className="inline-flex items-center gap-1.5">
                    <span aria-hidden="true">{COUNTRY_META[c as CountryCode]?.flag}</span>
                    {COUNTRY_META[c as CountryCode]?.name ?? c}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-civic-border">
            <ComparisonRow
              label="Bills introduced"
              values={selected.map((c) =>
                indicatorsData[c]?.bills_introduced
                  ? formatIndicatorValue(indicatorsData[c].bills_introduced.value, indicatorsData[c].bills_introduced.unit)
                  : '—',
              )}
            />
            <ComparisonRow
              label="Bills passed"
              values={selected.map((c) =>
                indicatorsData[c]?.bills_passed
                  ? formatIndicatorValue(indicatorsData[c].bills_passed.value, indicatorsData[c].bills_passed.unit)
                  : '—',
              )}
            />
            <ComparisonRow
              label="Acts commenced"
              values={selected.map((c) =>
                indicatorsData[c]?.acts_commenced
                  ? formatIndicatorValue(indicatorsData[c].acts_commenced.value, indicatorsData[c].acts_commenced.unit)
                  : '—',
              )}
            />
            <ComparisonRow
              label="Committee meetings"
              values={selected.map((c) =>
                indicatorsData[c]?.committee_meetings
                  ? formatIndicatorValue(indicatorsData[c].committee_meetings.value, indicatorsData[c].committee_meetings.unit)
                  : '—',
              )}
            />
          </tbody>
        </table>
      </section>

      <section className="rounded-lg border border-civic-border bg-civic-paper p-5">
        <BaseChart
          title="Bills introduced over time"
          data={chartData}
          xKey="year"
          yKeys={series}
          unit="count"
          sourceURL={selected.length > 0 ? profiles[selected[0]]?.parliament_url ?? '#' : '#'}
          sourceLabel="Parliament websites (per country)"
          definition="Annual count of Bills introduced in each country's national legislature. Pre-2024 values are illustrative baselines; 2024 reflects the published Bill tracker."
        />
      </section>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Public debt dimension
// ---------------------------------------------------------------------------

function PublicDebtView({
  selected,
  debtData,
  differences,
}: {
  selected: string[];
  debtData: Record<string, DebtComparisonRow>;
  differences: CompareDifference[];
}) {
  // Grouped bar chart: debt_to_gdp per country as a single-series bar chart.
  // The BaseChart is a line chart, so we render a custom SVG bar chart inline.
  const w = 720;
  const h = 280;
  const padding = { top: 24, right: 20, bottom: 60, left: 60 };
  const barCount = selected.length;
  const chartWidth = w - padding.left - padding.right;
  const barWidth = Math.max(20, Math.min(60, (chartWidth / Math.max(1, barCount)) * 0.6));
  const gap = barCount > 0 ? (chartWidth - barWidth * barCount) / (barCount + 1) : chartWidth / 2;

  const values = selected.map((c) => debtData[c]?.debt_to_gdp ?? 0);
  const maxVal = Math.max(...values, 100);
  const yMax = Math.ceil(maxVal / 20) * 20;
  const ticks = Array.from({ length: 6 }, (_, i) => (yMax * i) / 5);

  return (
    <div className="space-y-6">
      <section className="overflow-x-auto rounded-lg border border-civic-border bg-civic-paper">
        <table className="w-full text-sm">
          <caption className="sr-only">Public debt comparison across selected countries</caption>
          <thead className="bg-civic-mist">
            <tr>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
                Metric
              </th>
              {selected.map((c) => (
                <th key={c} scope="col" className="px-3 py-2 text-left text-xs font-semibold text-civic-forest">
                  <span className="inline-flex items-center gap-1.5">
                    <span aria-hidden="true">{COUNTRY_META[c as CountryCode]?.flag}</span>
                    {COUNTRY_META[c as CountryCode]?.name ?? c}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-civic-border">
            <ComparisonRow
              label="Debt to GDP (%)"
              values={selected.map((c) =>
                debtData[c]?.debt_to_gdp != null
                  ? `${debtData[c].debt_to_gdp.toFixed(1)}%`
                  : '—',
              )}
              highlight={(v, _i, all) => {
                // Highlight outliers: values more than 10pp from the median.
                const nums = all.map((x) => parseFloat(x.replace('%', ''))).filter((n) => !Number.isNaN(n));
                if (nums.length < 2) return false;
                nums.sort((a, b) => a - b);
                const median = nums[Math.floor(nums.length / 2)];
                const valNum = parseFloat(v.replace('%', ''));
                return Number.isFinite(valNum) && Math.abs(valNum - median) > 10;
              }}
            />
            <ComparisonRow
              label="Latest total stock"
              values={selected.map((c) =>
                debtData[c]?.latest_total_stock != null
                  ? formatCompactCurrency(debtData[c].latest_total_stock as number, debtData[c].currency)
                  : '—',
              )}
            />
            <ComparisonRow
              label="Source"
              values={selected.map((c) => debtData[c]?.source_label ?? '—')}
            />
          </tbody>
        </table>
      </section>

      {/* Grouped bar chart — custom SVG because BaseChart is line-only. */}
      <section className="rounded-lg border border-civic-border bg-civic-paper p-5" role="figure" aria-label="Debt to GDP bar chart">
        <div className="mb-2 flex items-start justify-between gap-3">
          <h3 className="font-serif text-base font-semibold text-civic-forest">
            Debt to GDP — grouped bar chart
          </h3>
          <span className="text-[11px] font-medium uppercase tracking-wider text-civic-stone">
            Unit: %
          </span>
        </div>
        <details className="mb-3">
          <summary className="cursor-pointer text-xs text-civic-stone">
            View tabular equivalent (screen-reader friendly)
          </summary>
          <table className="mt-2 w-full text-xs">
            <thead className="bg-civic-mist">
              <tr>
                <th className="px-2 py-1 text-left">Country</th>
                <th className="px-2 py-1 text-right">Debt to GDP (%)</th>
                <th className="px-2 py-1 text-left">Source</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-civic-border">
              {selected.map((c) => (
                <tr key={c}>
                  <td className="px-2 py-1">{COUNTRY_META[c as CountryCode]?.name ?? c}</td>
                  <td className="px-2 py-1 text-right">{debtData[c]?.debt_to_gdp?.toFixed(1) ?? '—'}</td>
                  <td className="px-2 py-1">{debtData[c]?.source_label ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </details>
        <svg role="img" viewBox={`0 0 ${w} ${h}`} className="w-full" aria-label={`Bar chart of debt to GDP for ${selected.length} countries`}>
          {/* Y axis */}
          <line x1={padding.left} y1={padding.top} x2={padding.left} y2={h - padding.bottom} stroke="#cbd5e1" />
          {/* X axis */}
          <line x1={padding.left} y1={h - padding.bottom} x2={w - padding.right} y2={h - padding.bottom} stroke="#cbd5e1" />
          {/* Y ticks + labels */}
          {ticks.map((tick, i) => {
            const y = h - padding.bottom - (tick / yMax) * (h - padding.top - padding.bottom);
            return (
              <g key={i}>
                <line x1={padding.left - 4} y1={y} x2={padding.left} y2={y} stroke="#94a3b8" />
                <text x={padding.left - 8} y={y + 4} textAnchor="end" fontSize="10" fill="#475569">
                  {tick.toFixed(0)}%
                </text>
              </g>
            );
          })}
          {/* Bars */}
          {selected.map((c, i) => {
            const val = debtData[c]?.debt_to_gdp ?? 0;
            const barHeight = (val / yMax) * (h - padding.top - padding.bottom);
            const x = padding.left + gap + i * (barWidth + gap);
            const y = h - padding.bottom - barHeight;
            return (
              <g key={c}>
                <rect x={x} y={y} width={barWidth} height={barHeight} fill={COUNTRY_COLORS[c] ?? '#1f5f3f'} rx={2}>
                  <title>{`${COUNTRY_META[c as CountryCode]?.name}: ${val.toFixed(1)}%`}</title>
                </rect>
                <text x={x + barWidth / 2} y={y - 6} textAnchor="middle" fontSize="11" fill="#1f2937" fontWeight="600">
                  {val.toFixed(1)}%
                </text>
                <text x={x + barWidth / 2} y={h - padding.bottom + 18} textAnchor="middle" fontSize="10" fill="#475569">
                  {COUNTRY_META[c as CountryCode]?.name ?? c}
                </text>
              </g>
            );
          })}
          {/* Y-axis label */}
          <text x={-(h / 2)} y={14} transform="rotate(-90)" textAnchor="middle" fontSize="11" fill="#52606b">
            Debt to GDP (%)
          </text>
        </svg>
        <p className="mt-3 text-[11px] text-civic-stone">
          Source: each country&apos;s Treasury + IMF WEO. Definition: Debt to
          GDP is the ratio of total public debt stock to gross domestic
          product. The platform does NOT rank countries by debt level — a
          higher ratio is not &ldquo;worse&rdquo; nor a lower ratio
          &ldquo;better&rdquo;.
        </p>
      </section>

      {differences.length > 0 && (
        <KeyDifferences differences={differences} selected={selected} />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Civic indicators dimension
// ---------------------------------------------------------------------------

function CivicIndicatorsView({
  selected,
  indicatorsData,
}: {
  selected: string[];
  indicatorsData: Record<string, Record<string, CountryIndicator>>;
}) {
  const indicatorKeys = Object.keys(INDICATOR_META);
  return (
    <section className="overflow-x-auto rounded-lg border border-civic-border bg-civic-paper">
      <table className="w-full text-sm">
        <caption className="sr-only">Civic indicators comparison across selected countries</caption>
        <thead className="bg-civic-mist">
          <tr>
            <th scope="col" className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
              Indicator
            </th>
            {selected.map((c) => (
              <th key={c} scope="col" className="px-3 py-2 text-left text-xs font-semibold text-civic-forest">
                <span className="inline-flex items-center gap-1.5">
                  <span aria-hidden="true">{COUNTRY_META[c as CountryCode]?.flag}</span>
                  {COUNTRY_META[c as CountryCode]?.name ?? c}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-civic-border">
          {indicatorKeys.map((key) => {
            const meta = INDICATOR_META[key];
            return (
              <tr key={key}>
                <th scope="row" className="px-3 py-2 text-left text-xs font-semibold text-civic-ink">
                  {meta.label}
                  <span className="ml-1 text-civic-stone">({meta.unit})</span>
                </th>
                {selected.map((c) => {
                  const ind = indicatorsData[c]?.[key];
                  return (
                    <td key={c} className="px-3 py-2 text-civic-ink">
                      {ind ? (
                        <div>
                          <div className="font-semibold">
                            {formatIndicatorValue(ind.value, ind.unit)}
                          </div>
                          <a
                            href={ind.source_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-0.5 text-[10px] text-sky-700 hover:underline"
                          >
                            source <ExternalLink className="h-2.5 w-2.5" aria-hidden="true" />
                          </a>
                        </div>
                      ) : (
                        <span className="text-civic-stone">—</span>
                      )}
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Reusable sub-components
// ---------------------------------------------------------------------------

function ComparisonRow({
  label,
  values,
  highlight,
}: {
  label: string;
  values: string[];
  highlight?: (value: string, index: number, all: string[]) => boolean;
}) {
  return (
    <tr>
      <th scope="row" className="px-3 py-2 text-left text-xs font-semibold text-civic-stone">
        {label}
      </th>
      {values.map((v, i) => {
        const isHighlighted = highlight?.(v, i, values) ?? false;
        return (
          <td
            key={i}
            className={`px-3 py-2 text-civic-ink ${isHighlighted ? 'bg-amber-50 font-semibold' : ''}`}
          >
            {v}
          </td>
        );
      })}
    </tr>
  );
}

function KeyDifferences({
  differences,
  selected,
}: {
  differences: CompareDifference[];
  selected: string[];
}) {
  return (
    <section className="rounded-lg border border-civic-border bg-civic-paper p-5">
      <div className="flex items-center gap-2">
        <Scale className="h-4 w-4 text-civic-leaf" aria-hidden="true" />
        <h3 className="font-serif text-base font-semibold text-civic-forest">
          Key differences
        </h3>
      </div>
      <p className="mt-1 text-xs text-civic-stone">
        Plain-language notes on what is structurally distinct across the selected countries.
        The platform describes differences only — it does not rank or evaluate.
      </p>
      <ul className="mt-3 space-y-3">
        {differences.map((d, i) => (
          <li key={i} className="rounded-md border border-civic-border bg-civic-mist/40 p-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-civic-stone">
              {d.dimension.replace(/_/g, ' ')}
            </div>
            <div className="mt-1 flex flex-wrap gap-3 text-xs">
              {selected.map((c) => (
                <span key={c} className="inline-flex items-center gap-1">
                  <span aria-hidden="true">{COUNTRY_META[c as CountryCode]?.flag}</span>
                  <span className="text-civic-stone">{COUNTRY_META[c as CountryCode]?.name ?? c}:</span>
                  <span className="font-semibold text-civic-forest">
                    {String(d.values[c] ?? '—')}
                  </span>
                </span>
              ))}
            </div>
            <p className="mt-2 text-xs text-civic-ink">{d.note}</p>
          </li>
        ))}
      </ul>
    </section>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <section
      className="rounded-lg border border-civic-border bg-civic-mist p-8 text-center text-sm text-civic-stone"
      role="status"
    >
      <Globe className="mx-auto mb-2 h-8 w-8 text-civic-stone" aria-hidden="true" />
      {label}
    </section>
  );
}

/** Format a currency value with compact notation. */
function formatCompactCurrency(amount: number, currency: string): string {
  if (!Number.isFinite(amount)) return '—';
  try {
    return new Intl.NumberFormat('en-KE', {
      style: 'currency',
      currency: currency || 'KES',
      notation: 'compact',
      maximumFractionDigits: 2,
    }).format(amount);
  } catch {
    return new Intl.NumberFormat('en-KE', {
      notation: 'compact',
      maximumFractionDigits: 2,
    }).format(amount);
  }
}
