// Client for the Cross-country Civic Comparison API (task ENG-I3).
//
// Endpoints (all under /api/v1/compare):
//   GET /countries              -- compare country profiles
//   GET /legislation            -- compare legislation by topic across countries
//   GET /debt                    -- compare public debt across countries
//   GET /government-structure    -- compare government structures
//   GET /indicators              -- compare civic indicators
//
// Every response carries the comparisonDisclaimer ("the platform does not
// rank countries"). The frontend MUST surface this disclaimer visibly on
// every compare page so users understand the platform describes
// differences only; it never ranks or implies political preference.

import { getJSON_ as getJSON } from './api';

/** Country code (ISO 3166-1 alpha-2). One of KE, UG, TZ, GH, NG, ZA. */
export type CountryCode = 'KE' | 'UG' | 'TZ' | 'GH' | 'NG' | 'ZA';

/** Supported comparison dimensions (mirrors the Go compare.go dimensions). */
export type CompareDimension =
  | 'government_structure'
  | 'legislative_activity'
  | 'public_debt'
  | 'civic_indicators';

/** A single chamber of a country's legislature. */
export interface HouseSummary {
  code: string;
  name: string;
  type: 'lower' | 'upper' | 'single';
  members: number;
  term_days: number;
}

/** Static institutional profile for one country. */
export interface CountryProfile {
  code: string;
  name: string;
  flag: string;
  government_system: string;
  house_count: number;
  houses: HouseSummary[];
  term_days: number;
  constitution_url: string;
  parliament_url: string;
  total_members: number;
  source_urls: string[];
  last_updated: string;
}

/** A single civic indicator value for one country. */
export interface CountryIndicator {
  code: string;
  name: string;
  flag: string;
  value: number;
  unit: string;
  year: number;
  source_url: string;
  source_label: string;
  last_updated: string;
}

/** A plain-language structural difference across the selected countries. */
export interface CompareDifference {
  dimension: string;
  values: Record<string, number | string>;
  note: string;
}

/** Inner comparison payload (the `comparison` field of every response). */
export interface ComparisonPayload {
  countries: string[];
  dimensions?: string[];
  topic?: string;
  from_year?: number;
  to_year?: number;
  indicators?: string[];
  data: Record<string, Record<string, unknown>>;
  differences?: CompareDifference[];
  disclaimer: string;
}

/** A single point in a country's debt series (Kenya only — sourced from CBK). */
export interface DebtSeriesPoint {
  date: string;
  year: number;
  total_stock?: number;
  domestic?: number;
  external?: number;
  source_url: string;
}

/** One row of debt comparison data. */
export interface DebtComparisonRow {
  country: string;
  debt_to_gdp: number;
  currency: string;
  source_url: string;
  source_label: string;
  last_updated: string;
  latest_total_stock?: number;
  latest_domestic?: number;
  latest_external?: number;
  latest_date?: string;
  series?: DebtSeriesPoint[];
  note?: string;
}

/** The 6 supported country codes. */
export const SUPPORTED_COUNTRIES: CountryCode[] = ['KE', 'UG', 'TZ', 'GH', 'NG', 'ZA'];

/** Human-readable labels + flag emojis for each supported country. */
export const COUNTRY_META: Record<CountryCode, { name: string; flag: string }> = {
  KE: { name: 'Kenya', flag: '🇰🇪' },
  UG: { name: 'Uganda', flag: '🇺🇬' },
  TZ: { name: 'Tanzania', flag: '🇹🇿' },
  GH: { name: 'Ghana', flag: '🇬🇭' },
  NG: { name: 'Nigeria', flag: '🇳🇬' },
  ZA: { name: 'South Africa', flag: '🇿🇦' },
};

/** Supported indicator keys + display labels. */
export const INDICATOR_META: Record<string, { label: string; unit: string }> = {
  bills_introduced: { label: 'Bills introduced', unit: 'count' },
  bills_passed: { label: 'Bills passed', unit: 'count' },
  acts_commenced: { label: 'Acts commenced', unit: 'count' },
  debt_to_gdp: { label: 'Public debt to GDP', unit: 'percent' },
  parliament_sessions: { label: 'Parliament sessions held', unit: 'count' },
  committee_meetings: { label: 'Committee meetings held', unit: 'count' },
  public_participation_opportunities: { label: 'Public participation opportunities', unit: 'count' },
};

/** Canonical disclaimer text. Mirrors the Go comparisonDisclaimer constant. */
export const COMPARISON_DISCLAIMER =
  'This comparison describes differences only. The platform does not rank countries or imply political preference.';

// --- API call helpers ---

/** GET /api/v1/compare/countries?countries=KE,UG,TZ */
export async function compareCountries(codes: string[]): Promise<ComparisonPayload & { data: Record<string, CountryProfile> }> {
  const qs = new URLSearchParams();
  if (codes.length > 0) qs.set('countries', codes.join(','));
  return getJSON(`/api/v1/compare/countries?${qs.toString()}`);
}

/** GET /api/v1/compare/legislation?countries=KE,UG&topic=health */
export async function compareLegislation(codes: string[], topic?: string): Promise<ComparisonPayload> {
  const qs = new URLSearchParams();
  if (codes.length > 0) qs.set('countries', codes.join(','));
  if (topic) qs.set('topic', topic);
  return getJSON(`/api/v1/compare/legislation?${qs.toString()}`);
}

/** GET /api/v1/compare/debt?countries=KE,UG&from=2020&to=2024 */
export async function compareDebt(codes: string[], fromYear?: number, toYear?: number): Promise<ComparisonPayload & { data: Record<string, DebtComparisonRow> }> {
  const qs = new URLSearchParams();
  if (codes.length > 0) qs.set('countries', codes.join(','));
  if (fromYear) qs.set('from', String(fromYear));
  if (toYear) qs.set('to', String(toYear));
  return getJSON(`/api/v1/compare/debt?${qs.toString()}`);
}

/** GET /api/v1/compare/government-structure?countries=KE,UG,NG,ZA */
export async function compareGovernmentStructure(codes: string[]): Promise<ComparisonPayload> {
  const qs = new URLSearchParams();
  if (codes.length > 0) qs.set('countries', codes.join(','));
  return getJSON(`/api/v1/compare/government-structure?${qs.toString()}`);
}

/** GET /api/v1/compare/indicators?countries=KE,UG,TZ&indicators=debt_to_gdp,bills_introduced */
export async function compareIndicators(codes: string[], indicators?: string[]): Promise<ComparisonPayload> {
  const qs = new URLSearchParams();
  if (codes.length > 0) qs.set('countries', codes.join(','));
  if (indicators && indicators.length > 0) qs.set('indicators', indicators.join(','));
  return getJSON(`/api/v1/compare/indicators?${qs.toString()}`);
}

/** Helper to format a numeric value with the indicator's unit. */
export function formatIndicatorValue(value: number, unit: string): string {
  if (!Number.isFinite(value)) return '—';
  switch (unit) {
    case 'percent':
      return `${value.toFixed(1)}%`;
    case 'count':
      return new Intl.NumberFormat('en-KE', { notation: 'compact', maximumFractionDigits: 1 }).format(value);
    default:
      return new Intl.NumberFormat('en-KE', { maximumFractionDigits: 2 }).format(value);
  }
}
