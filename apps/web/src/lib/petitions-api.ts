// Client for the Public Participation Portal API (issue #287 / FEAT-9).
//
// All 4 endpoints under /api/v1/petitions expose the e-petition
// lifecycle: list (filterable by status + country), detail (full
// description + signatures), create (POST a new petition), and sign
// (POST a signature on an existing petition).
//
// The platform NEVER derives a "popular support score" or "approval
// rating" from these records (rule: NO_POLITICAL_PERFORMANCE_SCORE) —
// the UI mirrors that invariant: it renders raw signature counts +
// progress bars (count / target) without any qualitative verdict.

import { getJSON_ as getJSON, postJSON } from './api';

// PetitionStatus is the 3-valued enum for a petition's lifecycle
// state. The string values are the JSON wire forms (lowercase, no
// spaces) — see services/api/cmd/petitions.go for the canonical
// definitions + the defensive init-time validation.
export type PetitionStatus = 'open' | 'closed' | 'answered';

// Petition mirrors the JSON shape returned by GET /api/v1/petitions
// (list) + GET /api/v1/petitions/{id} (detail). The list endpoint
// re-derives signatures_count from the signatures slice on every
// read so a stale denormalised value can never drift away from the
// actual signature list.
export interface Petition {
  id: string;
  title: string;
  description: string;
  created_by: string;
  created_at: string;
  status: PetitionStatus;
  signatures_count: number;
  target_signatures: number;
  closes_at: string;
  country: string;
}

// PetitionSignature mirrors the JSON shape of a single signature row.
// The platform NEVER publishes the signer's email on the public list
// (the list endpoint returns Name + SignedAt + Verified only); the
// email is collected solely so the verification pipeline can confirm
// the signer's identity out-of-band. The detail endpoint DOES return
// the email so the petitioner can audit who has signed — but the
// frontend deliberately does NOT render it on the public detail view.
export interface PetitionSignature {
  id: string;
  petition_id: string;
  name: string;
  email: string;
  signed_at: string;
  verified: boolean;
}

// PetitionsListResponse is the JSON envelope returned by
// GET /api/v1/petitions. The Status + Country fields echo the
// caller-supplied filters (empty when omitted) so the frontend can
// render the active filter as a chip / breadcrumb.
export interface PetitionsListResponse {
  items: Petition[];
  total: number;
  source: string; // always 'seed' until ingestion ships
  status?: PetitionStatus;
  country?: string;
}

// PetitionDetailResponse is the JSON envelope returned by
// GET /api/v1/petitions/{id}. Carries the full petition + the
// signatures slice hydrated.
export interface PetitionDetailResponse {
  petition: Petition;
  signatures: PetitionSignature[];
  source: string;
}

// CreatePetitionRequest is the JSON body for POST /api/v1/petitions.
// Mirrors the Go createPetitionRequest struct — every field is
// required except where noted.
export interface CreatePetitionRequest {
  title: string;
  description: string;
  created_by: string;
  target_signatures: number;
  closes_at: string; // RFC3339 timestamp; MUST be in the future
  country: string; // ISO 3166-1 alpha-2 country code (e.g. KE)
}

// SignPetitionRequest is the JSON body for POST /api/v1/petitions/{id}/sign.
export interface SignPetitionRequest {
  name: string;
  email: string;
}

// SignPetitionResponse is the JSON envelope returned by the sign
// endpoint. Carries the created signature + the updated petition so
// the frontend can re-render the progress bar without a second
// round-trip.
export interface SignPetitionResponse {
  signature: PetitionSignature;
  petition: Petition;
}

const BASE = '/api/v1/petitions';

// listPetitions returns the petition list, optionally filtered by
// status + country. Both filters are optional + case-insensitive
// (country). An invalid status filter throws an ApiError with
// status 400 — the Go handler is strict so a typo does not silently
// look like a legitimate "no petitions match" response.
export async function listPetitions(params: {
  status?: PetitionStatus;
  country?: string;
} = {}): Promise<PetitionsListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set('status', params.status);
  if (params.country) qs.set('country', params.country);
  const suffix = qs.toString() ? `?${qs.toString()}` : '';
  return getJSON<PetitionsListResponse>(`${BASE}${suffix}`);
}

// getPetition returns the petition detail + the full signatures
// slice. Throws ApiError with status 404 for an unknown id.
export async function getPetition(id: string): Promise<PetitionDetailResponse> {
  return getJSON<PetitionDetailResponse>(`${BASE}/${id}`);
}

// createPetition POSTs a new petition. Throws ApiError with status
// 400 for a missing / invalid field. On success returns the created
// Petition (status: open, signatures_count: 0).
export async function createPetition(req: CreatePetitionRequest): Promise<Petition> {
  return postJSON<Petition>(BASE, req);
}

// signPetition POSTs a signature on the given petition. Throws
// ApiError with status 409 if the petition is not open for
// signatures (status closed / answered). On success returns the
// created signature + the updated petition.
export async function signPetition(
  id: string,
  req: SignPetitionRequest,
): Promise<SignPetitionResponse> {
  return postJSON<SignPetitionResponse>(`${BASE}/${id}/sign`, req);
}
