'use client';

/**
 * IndicatorsFilters — small client component that drives the country +
 * year filter on the Civic Indicators Dashboard. Uses the Next.js router
 * to update the URL so the page stays a server component.
 */

import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { COUNTRY_META } from '@/lib/compare-api';

interface IndicatorsFiltersProps {
  selectedCountry: string;
  year: number;
}

const YEARS = [2024, 2023, 2022, 2021, 2020];

export function IndicatorsFilters({ selectedCountry, year }: IndicatorsFiltersProps) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();

  const update = (key: string, value: string) => {
    const next = new URLSearchParams(params?.toString() ?? '');
    if (!value || value === 'ALL') next.delete(key);
    else next.set(key, value);
    const qs = next.toString();
    router.push(qs ? `${pathname}?${qs}` : pathname);
  };

  return (
    <div className="flex flex-wrap items-center gap-3">
      <label htmlFor="indicator-country" className="text-xs font-semibold uppercase tracking-wide text-civic-stone">
        Country
      </label>
      <select
        id="indicator-country"
        value={selectedCountry}
        onChange={(e) => update('country', e.target.value)}
        className="rounded-md border border-civic-border bg-civic-paper px-2 py-1 text-sm text-civic-ink"
      >
        <option value="ALL">All countries</option>
        {Object.entries(COUNTRY_META).map(([code, meta]) => (
          <option key={code} value={code}>
            {meta.flag} {meta.name}
          </option>
        ))}
      </select>
      <label htmlFor="indicator-year" className="ml-3 text-xs font-semibold uppercase tracking-wide text-civic-stone">
        Year
      </label>
      <select
        id="indicator-year"
        value={String(year)}
        onChange={(e) => update('year', e.target.value)}
        className="rounded-md border border-civic-border bg-civic-paper px-2 py-1 text-sm text-civic-ink"
      >
        {YEARS.map((y) => (
          <option key={y} value={String(y)}>
            {y}
          </option>
        ))}
      </select>
    </div>
  );
}
