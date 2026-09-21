// Tiny typed fetch client for the BFF + AI service.
// We don't pull in heavier HTTP libs to keep the bundle small.
//
// Shared API types — mirror the Go BFF + Python AI service contracts.
// In production these are generated from openapi.yaml via openapi-typescript.

import type {
  AIResponse,
  Bill,
  BillEvent,
  BillSummary,
  BillVersion,
  BriefArchiveEntry,
  CivicBrief,
  DailyBriefing,
  SearchResult,
} from './types';
import { GOVERNMENT_COOKIE, DEFAULT_SELECTION } from './government-defaults';

// X-Civic-Country is the HTTP header the BFF's CountryMiddleware
// (services/api/internal/middleware/country.go) reads to scope every list
// endpoint to the user's selected country. The header value is the ISO
// 3166-1 alpha-2 code (KE, UG, TZ, GH, NG, ZA) or "ALL" for the global
// dashboard view. Exported so other client modules (e.g. graph-api,
// scenarios-api) can reference the constant by name.
export const COUNTRY_HEADER = 'X-Civic-Country';

/**
 * Reads the user's selected country code from the `civic_gov_selection`
 * cookie. This is the SAME cookie that `app/layout.tsx` reads server-side
 * to seed the GovernmentProvider — so the value the API client sends on
 * the very first request from a freshly loaded page matches the country
 * the navbar Government Selector will render.
 *
 * Returns "KE" (the platform default) when:
 *   - running on the server (no `document`)
 *   - the cookie is absent (first-time visitor)
 *   - the cookie is malformed (corrupt or older schema)
 *
 * This helper is safe to call from any client-side code; it never throws.
 * It is the single source of truth for "what country does the API client
 * think the user is in?" — kept here in `lib/api.ts` rather than in
 * `lib/government-context.tsx` so non-React modules (e.g. the service
 * worker, server actions) can import it without dragging in the React
 * client boundary.
 */
export function getCountryFromCookie(): string {
  if (typeof document === 'undefined') return DEFAULT_SELECTION.countryCode;
  // document.cookie returns a single "k=v; k2=v2; ..." string. We do a
  // manual scan rather than `new URLSearchParams` because the cookie
  // value is URI-encoded JSON — URLSearchParams would double-decode it.
  const prefix = `${GOVERNMENT_COOKIE}=`;
  const raw = document.cookie
    .split(';')
    .map((c) => c.trim())
    .find((c) => c.startsWith(prefix));
  if (!raw) return DEFAULT_SELECTION.countryCode;
  try {
    const json = decodeURIComponent(raw.slice(prefix.length));
    const parsed = JSON.parse(json) as { countryCode?: string };
    const code = (parsed.countryCode ?? '').toUpperCase();
    // Validate against the supported set so a stale cookie with an old
    // country code (e.g. a removed country) does not produce a 400 from
    // the BFF — fall back to the default instead.
    if (SUPPORTED_COUNTRY_CODES.has(code) || code === 'ALL') {
      return code;
    }
  } catch {
    // Malformed cookie — fall through to the default.
  }
  return DEFAULT_SELECTION.countryCode;
}

// SUPPORTED_COUNTRY_CODES mirrors middleware.SupportedCountries in the Go
// BFF (services/api/internal/middleware/country.go). Kept in sync manually
// — if a new country is added there, it must be added here too. The
// contract test in tests/contract/adapter_contract_test.go enforces the
// adapter side; this constant enforces the client side.
//
// Wave 12 (2026) added Rwanda, Zambia, Senegal, Egypt.
// Wave 13 (2026) added Morocco, DR Congo, Ethiopia, Malawi.
export const SUPPORTED_COUNTRY_CODES = new Set<string>([
  'KE',
  'UG',
  'TZ',
  'GH',
  'NG',
  'ZA',
  'RW',
  'ZM',
  'SN',
  'EG',
  'MA',
  'CD',
  'ET',
  'MW',
]);

export class ApiError extends Error {
  constructor(public status: number, message: string, public detail?: unknown) {
    super(message);
    this.name = 'ApiError';
  }
}

// countryHeaders returns the country-scoping HTTP header bag for the
// current session. Used by both getJSON and postJSON so EVERY API call
// automatically carries the country context — no caller has to remember
// to set it.
function countryHeaders(): Record<string, string> {
  return { [COUNTRY_HEADER]: getCountryFromCookie() };
}

async function getJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(url, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...countryHeaders(),
      ...(init?.headers ?? {}),
    },
  });
  if (!resp.ok) {
    let detail: unknown;
    try { detail = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, `HTTP ${resp.status} ${url}`, detail);
  }
  // Some endpoints may return 204 No Content.
  if (resp.status === 204) return undefined as T;
  return (await resp.json()) as T;
}

// getJSON_ is the exported form of the internal helper, so other client
// modules (e.g. government-api, scenarios-api, debt-api) can reuse the
// same fetch + error-handling logic.
export { getJSON as getJSON_ };

// postJSON is exported for clients that need to POST JSON bodies
// (e.g. scenarios comparison + run endpoints, follow-a-law).
export async function postJSON<T>(url: string, body?: unknown): Promise<T> {
  const init: RequestInit = {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...countryHeaders(),
    },
  };
  if (body !== undefined) init.body = JSON.stringify(body);
  return getJSON<T>(url, init);
}

// ----- Bills -----

export async function listBills(params: {
  page?: number;
  pageSize?: number;
  status?: string;
  house?: string;
  topic?: string;
  q?: string;
} = {}): Promise<{ items: Bill[]; total: number; page: number; page_size: number }> {
  const qs = new URLSearchParams();
  if (params.page) qs.set('page', String(params.page));
  if (params.pageSize) qs.set('page_size', String(params.pageSize));
  if (params.status) qs.set('status', params.status);
  if (params.house) qs.set('house', params.house);
  if (params.topic) qs.set('topic', params.topic);
  if (params.q) qs.set('q', params.q);
  return getJSON(`/api/v1/bills?${qs.toString()}`);
}

export async function getBill(id: string): Promise<Bill> {
  return getJSON(`/api/v1/bills/${id}`);
}

export async function getBillTimeline(id: string): Promise<{ events: BillEvent[] }> {
  return getJSON(`/api/v1/bills/${id}/timeline`);
}

export async function getBillVersions(id: string): Promise<{ versions: BillVersion[] }> {
  return getJSON(`/api/v1/bills/${id}/versions`);
}

export async function getBillSummary(id: string): Promise<BillSummary> {
  return getJSON(`/api/v1/bills/${id}/summary`);
}

// ----- Search -----

export async function search(q: string, opts: {
  kind?: string;
  page?: number;
  pageSize?: number;
} = {}): Promise<{ items: SearchResult[]; total: number }> {
  const qs = new URLSearchParams({ q });
  if (opts.kind) qs.set('kind', opts.kind);
  if (opts.page) qs.set('page', String(opts.page));
  if (opts.pageSize) qs.set('page_size', String(opts.pageSize));
  return getJSON(`/api/v1/search?${qs.toString()}`);
}

// ----- Briefing -----

export async function getBriefing(date?: string): Promise<DailyBriefing> {
  const qs = date ? `?date=${date}` : '';
  return getJSON(`/api/v1/briefing${qs}`);
}

// ----- Civic Daily Brief (task ENG-I2) -----
//
// The personalised Civic Daily Brief lives under /api/v1/brief/* on the Go
// BFF. Every item in every section carries an evidence_url — the brief
// never presents a claim without a primary source. The AI summary carries
// a disclaimer that the frontend renders next to the ASSUMPTION reality
// badge.

export interface BriefGenerateRequest {
  user_id?: string;
  followed_topics?: string[];
  followed_institutions?: string[];
  followed_bills?: string[];
  country?: string;
}

/** POST /api/v1/brief/generate — generate (and persist) a personalised brief. */
export async function generateBrief(req: BriefGenerateRequest = {}): Promise<CivicBrief> {
  return postJSON<CivicBrief>('/api/v1/brief/generate', req);
}

/** GET /api/v1/brief/today — today's brief, generated on-demand if missing. */
export async function getTodaysBrief(): Promise<CivicBrief> {
  return getJSON<CivicBrief>('/api/v1/brief/today');
}

/** GET /api/v1/brief/archive — list previously generated briefs (newest-first). */
export async function listBriefArchive(): Promise<{ items: BriefArchiveEntry[]; total: number }> {
  return getJSON<{ items: BriefArchiveEntry[]; total: number }>('/api/v1/brief/archive');
}

/** GET /api/v1/brief/{id} — fetch a single brief by ID. */
export async function getBrief(id: string): Promise<CivicBrief> {
  return getJSON<CivicBrief>(`/api/v1/brief/${encodeURIComponent(id)}`);
}

// ----- AI Q&A (streams via SSE) -----

export async function askQuestion(
  text: string,
  opts: { billId?: string; impactLens?: string } = {},
): Promise<AIResponse> {
  const body: Record<string, unknown> = { text, country: 'KE' };
  if (opts.billId) body.bill_id = opts.billId;
  if (opts.impactLens) body.impact_lens = opts.impactLens;
  const resp = await fetch('/api/v1/ai/questions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!resp.ok) {
    throw new ApiError(resp.status, `AI question failed: ${resp.status}`);
  }
  return (await resp.json()) as AIResponse;
}

// ----- Follow (notifications) -----

export async function followBill(billId: string): Promise<{ followed: boolean }> {
  const resp = await fetch(`/api/v1/bills/${billId}/follow`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  });
  if (!resp.ok) throw new ApiError(resp.status, 'Follow failed');
  return { followed: true };
}

// ----- Subscriptions (issue #110 — Following) -----

export interface Subscription {
  id: string;
  user_id: string;
  entity_type: 'bill' | 'committee' | 'topic' | 'institution' | 'person';
  entity_id: string;
  created_at: string;
}

/**
 * Follow an entity (Bill, committee, topic, institution, person).
 * Requires an authenticated caller; the Bearer token is added by the
 * browser's fetch integration if the user is signed in (issue #20).
 */
export async function createSubscription(params: {
  entityType: string;
  entityId: string;
  token?: string;
}): Promise<Subscription> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (params.token) headers.Authorization = `Bearer ${params.token}`;
  const resp = await fetch('/api/v1/subscriptions', {
    method: 'POST',
    headers,
    body: JSON.stringify({ entity_type: params.entityType, entity_id: params.entityId }),
  });
  if (!resp.ok) {
    let detail: unknown;
    try { detail = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, 'Follow failed', detail);
  }
  return resp.json();
}

/**
 * List the caller's follows. Optional `entityType` filter narrows the result
 * set to a single entity type (e.g. "bill").
 */
export async function listSubscriptions(params: {
  entityType?: string;
  token?: string;
} = {}): Promise<{ items: Subscription[]; total: number }> {
  const qs = new URLSearchParams();
  if (params.entityType) qs.set('entity_type', params.entityType);
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (params.token) headers.Authorization = `Bearer ${params.token}`;
  const resp = await fetch(`/api/v1/subscriptions?${qs.toString()}`, { headers });
  if (!resp.ok) {
    let detail: unknown;
    try { detail = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, 'List subscriptions failed', detail);
  }
  return resp.json();
}

/**
 * Unfollow an entity by subscription ID. Requires an authenticated caller.
 */
export async function deleteSubscription(params: {
  id: string;
  token?: string;
}): Promise<void> {
  const headers: Record<string, string> = {};
  if (params.token) headers.Authorization = `Bearer ${params.token}`;
  const resp = await fetch(`/api/v1/subscriptions/${params.id}`, {
    method: 'DELETE',
    headers,
  });
  if (!resp.ok && resp.status !== 204) {
    let detail: unknown;
    try { detail = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, 'Unfollow failed', detail);
  }
}

// ----- Trust + Provenance (issue #165) -----

export type AuthorityLevel =
  | 'PRIMARY_OFFICIAL'
  | 'OFFICIAL_REPOSITORY'
  | 'SECONDARY_VERIFIED'
  | 'UNVERIFIED';

export type VerificationState =
  | 'UNVERIFIED'
  | 'DISCOVERED'
  | 'EXTRACTED'
  | 'VALIDATING'
  | 'VERIFIED'
  | 'CONFLICTED'
  | 'CORRECTED'
  | 'SUPERSEDED'
  | 'REJECTED';

export interface TrustSource {
  id: string;
  institution_id: string;
  country: string;
  source_type: string;
  authority_level: AuthorityLevel;
  official_url: string;
  domain: string;
  status: 'active' | 'degraded' | 'retired' | 'blocked';
  verification_method?: string;
  last_verified_at?: string;
  health_status: 'healthy' | 'degraded' | 'down' | 'unknown';
  created_at: string;
  last_check?: SourceCheck;
}

export interface SourceCheck {
  id: string;
  source_id: string;
  checked_at: string;
  http_status: number;
  latency_ms: number;
  tls_valid: boolean;
  content_hash?: string;
  changed: boolean;
}

export interface Claim {
  id: string;
  subject: string;
  predicate: string;
  object: string;
  claim_type: 'FACT' | 'EXPLANATION' | 'INFERENCE' | 'UNKNOWN';
  text: string;
  confidence: number;
  verification_state: VerificationState;
  created_at: string;
  valid_from?: string;
  valid_to?: string;
}

export interface Evidence {
  id: string;
  claim_id: string;
  document_id?: string;
  snapshot_id?: string;
  page_number?: number;
  section?: string;
  paragraph?: string;
  text_span?: string;
  source_url: string;
  retrieved_at: string;
  content_hash?: string;
  source?: TrustSource;
}

export interface ClaimWithEvidence {
  claim: Claim;
  evidence: Evidence[];
}

export interface ProvenanceResponse {
  entity_type: string;
  entity_id: string;
  claims: ClaimWithEvidence[];
  total: number;
}

export interface Contradiction {
  id: string;
  claim_a_id: string;
  claim_b_id: string;
  source_a_id?: string;
  source_b_id?: string;
  detected_at: string;
  status: 'detected' | 'under_review' | 'resolved' | 'dismissed';
  resolution?: string;
  reviewer_id?: string;
  resolved_at?: string;
  claim_a?: Claim;
  claim_b?: Claim;
  source_a?: TrustSource;
  source_b?: TrustSource;
}

/**
 * Fetch the full evidence chain for an entity (issue #165).
 * Returns every claim whose subject is `{entity_type}:{entity_id}`, each
 * with its evidence list and (where resolvable) the source row.
 */
export async function getProvenance(
  entityType: string,
  entityId: string,
): Promise<ProvenanceResponse> {
  return getJSON(`/api/v1/provenance/${entityType}/${encodeURIComponent(entityId)}`);
}

/** GET /api/v1/evidence/{id} — evidence detail with hydrated source. */
export async function getEvidence(id: string): Promise<Evidence> {
  return getJSON(`/api/v1/evidence/${encodeURIComponent(id)}`);
}

/** GET /api/v1/claims/{id}/evidence — a single claim plus its evidence list. */
export async function getClaimEvidence(
  id: string,
): Promise<{ claim: Claim; evidence: Evidence[]; total: number }> {
  return getJSON(`/api/v1/claims/${encodeURIComponent(id)}/evidence`);
}

/** GET /api/v1/contradictions — list active source conflicts. */
export async function listContradictions(
  status?: 'detected' | 'under_review' | 'resolved' | 'dismissed',
): Promise<{ items: Contradiction[]; total: number }> {
  const qs = new URLSearchParams();
  if (status) qs.set('status', status);
  return getJSON(`/api/v1/contradictions${qs.size ? `?${qs.toString()}` : ''}`);
}

/** GET /api/v1/sources — list trust sources with optional ?authority= and ?country= filters. */
export async function listTrustSources(opts: {
  authority?: AuthorityLevel;
  country?: string;
} = {}): Promise<{ items: TrustSource[]; total: number }> {
  const qs = new URLSearchParams();
  if (opts.authority) qs.set('authority', opts.authority);
  if (opts.country) qs.set('country', opts.country.toUpperCase());
  return getJSON(`/api/v1/sources${qs.size ? `?${qs.toString()}` : ''}`);
}

/** GET /api/v1/sources/{id} — source detail with health + last check. */
export async function getTrustSource(id: string): Promise<TrustSource> {
  return getJSON(`/api/v1/sources/${encodeURIComponent(id)}`);
}

// ----- Civic Calendar (task ENG-K1 — Feature 1) -----
//
// The calendar surfaces upcoming parliamentary sessions, committee
// meetings, bill readings, public-participation deadlines, gazette
// publications, court hearings, and budget presentations. Events are
// read-only (no auth required); the "Subscribe to event" button posts a
// follow via createSubscription, which DOES require auth.

export type CalendarEventType =
  | 'parliament_session'
  | 'committee_meeting'
  | 'bill_reading'
  | 'public_participation'
  | 'gazette_publication'
  | 'court_hearing'
  | 'budget_presentation';

export const CALENDAR_EVENT_TYPES: CalendarEventType[] = [
  'parliament_session',
  'committee_meeting',
  'bill_reading',
  'public_participation',
  'gazette_publication',
  'court_hearing',
  'budget_presentation',
];

export interface CalendarEvent {
  id: string;
  title: string;
  /** YYYY-MM-DD */
  date: string;
  type: CalendarEventType;
  description?: string;
  source_url?: string;
  country: string;
  institution?: string;
}

/**
 * GET /api/v1/calendar?month=YYYY-MM&country=KE — events for a month.
 * Pass `types` to filter by event type (comma-separated).
 */
export async function listCalendarEvents(params: {
  month?: string; // YYYY-MM (defaults to current month on the server)
  country?: string; // ISO-2 code (defaults to "KE")
  types?: CalendarEventType[];
} = {}): Promise<{ month: string; items: CalendarEvent[]; total: number }> {
  const qs = new URLSearchParams();
  if (params.month) qs.set('month', params.month);
  if (params.country) qs.set('country', params.country);
  if (params.types && params.types.length > 0) qs.set('types', params.types.join(','));
  return getJSON(`/api/v1/calendar?${qs.toString()}`);
}

/** GET /api/v1/calendar/today?country=KE — today's events. */
export async function listCalendarToday(params: {
  country?: string;
  types?: CalendarEventType[];
} = {}): Promise<{ date: string; items: CalendarEvent[]; total: number }> {
  const qs = new URLSearchParams();
  if (params.country) qs.set('country', params.country);
  if (params.types && params.types.length > 0) qs.set('types', params.types.join(','));
  return getJSON(`/api/v1/calendar/today?${qs.toString()}`);
}

/** GET /api/v1/calendar/upcoming?country=KE&limit=10 — upcoming events. */
export async function listCalendarUpcoming(params: {
  country?: string;
  limit?: number;
  types?: CalendarEventType[];
} = {}): Promise<{ items: CalendarEvent[]; total: number; limit: number }> {
  const qs = new URLSearchParams();
  if (params.country) qs.set('country', params.country);
  if (params.limit) qs.set('limit', String(params.limit));
  if (params.types && params.types.length > 0) qs.set('types', params.types.join(','));
  return getJSON(`/api/v1/calendar/upcoming?${qs.toString()}`);
}

// ----- Gazette Alerts (task ENG-K1 — Feature 2) -----
//
// Citizens subscribe to keywords in the Kenya Gazette. When a matching
// legal notice is published, they get notified. Alerts are matched ONLY
// against gazette notices already published (gazette_date <= today) —
// drafts / future-dated notices never match.

export interface GazetteAlert {
  id: string;
  user_id: string;
  keywords: string[];
  country: string;
  created_at: string;
  match_count: number;
}

export interface GazetteNotice {
  id: string;
  title: string;
  /** YYYY-MM-DD */
  gazette_date: string;
  notice_number: string;
  volume?: string;
  summary?: string;
  source_url: string;
  authority?: string;
  matched_keywords?: string[];
  country: string;
  body_text?: string;
}

/**
 * POST /api/v1/gazette/alerts — create a keyword alert.
 *
 * Pass `userId` explicitly to use the alert feature before identity (#20)
 * ships. When the caller is authenticated, the principal's user_id takes
 * precedence (the body's user_id is ignored to prevent cross-user alert
 * creation).
 */
export async function createGazetteAlert(params: {
  keywords: string[];
  userId: string;
  country?: string;
  token?: string;
}): Promise<GazetteAlert> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (params.token) headers.Authorization = `Bearer ${params.token}`;
  const resp = await fetch('/api/v1/gazette/alerts', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      keywords: params.keywords,
      user_id: params.userId,
      country: params.country ?? 'KE',
    }),
  });
  if (!resp.ok) {
    let detail: unknown;
    try { detail = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, 'Create alert failed', detail);
  }
  return resp.json();
}

/**
 * GET /api/v1/gazette/alerts?user_id=... — list the caller's alerts.
 * Each alert carries a `match_count` field that's computed on demand
 * against currently-published gazette notices.
 */
export async function listGazetteAlerts(params: {
  userId: string;
  token?: string;
}): Promise<{ items: GazetteAlert[]; total: number }> {
  const qs = new URLSearchParams({ user_id: params.userId });
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (params.token) headers.Authorization = `Bearer ${params.token}`;
  const resp = await fetch(`/api/v1/gazette/alerts?${qs.toString()}`, { headers });
  if (!resp.ok) {
    let detail: unknown;
    try { detail = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, 'List alerts failed', detail);
  }
  return resp.json();
}

/** DELETE /api/v1/gazette/alerts/{id} — delete an alert. */
export async function deleteGazetteAlert(params: {
  id: string;
  userId: string;
  token?: string;
}): Promise<void> {
  const qs = new URLSearchParams({ user_id: params.userId });
  const headers: Record<string, string> = {};
  if (params.token) headers.Authorization = `Bearer ${params.token}`;
  const resp = await fetch(`/api/v1/gazette/alerts/${encodeURIComponent(params.id)}?${qs.toString()}`, {
    method: 'DELETE',
    headers,
  });
  if (!resp.ok && resp.status !== 204) {
    let detail: unknown;
    try { detail = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, 'Delete alert failed', detail);
  }
}

/**
 * GET /api/v1/gazette/alerts/{id}/matches — list gazette notices matching
 * the alert's keywords. The caller does NOT need to be the alert owner;
 * match results are derived from already-published gazette notices (which
 * are public information).
 */
export async function listGazetteAlertMatches(params: {
  id: string;
}): Promise<{ alert_id: string; items: GazetteNotice[]; total: number; note: string }> {
  return getJSON(`/api/v1/gazette/alerts/${encodeURIComponent(params.id)}/matches`);
}
