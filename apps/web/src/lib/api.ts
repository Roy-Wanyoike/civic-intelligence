// Tiny typed fetch client for the BFF + AI service.
// We don't pull in heavier HTTP libs to keep the bundle small.

import type {
  AIResponse,
  Bill,
  BillEvent,
  BillSummary,
  BillVersion,
  DailyBriefing,
  SearchResult,
} from './types';

export class ApiError extends Error {
  constructor(public status: number, message: string, public detail?: unknown) {
    super(message);
    this.name = 'ApiError';
  }
}

async function getJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(url, {
    ...init,
    headers: {
      'Accept': 'application/json',
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
