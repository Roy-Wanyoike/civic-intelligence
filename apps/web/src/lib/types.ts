// Shared API types — mirror the Go BFF + Python AI service contracts.
// In production these are generated from openapi.yaml via openapi-typescript.

export type ConfidenceLevel = 'high' | 'medium' | 'low' | 'unknown';

export interface Citation {
  document_id: string;
  source_url: string;
  page_number: number | null;
  section: string | null;
  snippet: string;
  source_type:
    | 'parliament'
    | 'gazette'
    | 'kenya_law'
    | 'hansard'
    | 'committee_report'
    | 'order_paper'
    | 'votes_proceedings';
  retrieved_at: string;
}

export interface SponsorRef {
  person_id: string;
  name: string;
  scorecard_url?: string;
}

export interface Bill {
  id: string;
  identifier: string;
  title: string;
  year: number;
  sponsor_id: string | null;
  sponsor_name: string | null;
  // scorecard_url is the platform-internal path to the sponsor's MP
  // scorecard page (/api/v1/people/{id}/scorecard). Populated when
  // sponsor_id is known AND matches a sample MP; omitted otherwise so
  // the frontend can gate the "Sponsored by" link on its presence
  // (issue #282).
  scorecard_url?: string;
  // cosponsors is the list of secondary supporters. Empty/omitted when
  // the Bill has no known cosponsors — the frontend must NOT render an
  // empty "Cosponsors:" row in that case (issue #282).
  cosponsors?: SponsorRef[];
  house_id: string | null;
  house_name: string | null;
  house?: string; // API returns "house" directly (e.g., "National Assembly")
  committee_id: string | null;
  committee_name: string | null;
  status: string;
  current_stage: string | null;
  current_stage_simple_explanation: string | null;
  purpose: string | null;
  description: string | null;
  country: string;
  source_url?: string; // the official source URL (kenyalaw.org, parliament.go.ke)
  publication_date?: string; // ISO date string
  // video_url is the YouTube URL of the most-recent Hansard sitting in
  // which this Bill was debated (issue #283). Empty/omitted when no
  // Hansard video is available yet; the frontend renders an inline
  // YouTube embed below the Bill title when present.
  video_url?: string;
  created_at: string;
  updated_at: string;
  citation_count: number;
  version_count: number;
  event_count: number;
  topics: string[];
}

export interface BillEvent {
  id: string;
  bill_id: string;
  event_type: string;
  date: string | null;
  date_is_approximate: boolean;
  house: string | null;
  description: string;
  source_url: string | null;
  confidence: ConfidenceLevel;
  note: string | null;
}

export interface BillVersion {
  id: string;
  bill_id: string;
  version_no: number;
  content_hash: string;
  retrieved_at: string;
  source_url: string;
  document_id: string;
}

// AmendmentStatus mirrors the Go AmendmentStatus enum in
// services/api/cmd/amendments.go. The three values match the
// parliamentary committee-stage vocabulary:
//   - proposed: tabled, not yet voted on
//   - accepted: carried (majority vote) → folded into the Bill's text
//   - rejected: defeated → Bill's text unchanged
export type AmendmentStatus = 'proposed' | 'accepted' | 'rejected';

// Amendment mirrors the Go Amendment struct returned by
// GET /api/v1/bills/{id}/amendments (issue #293 / FEAT-15). The
// amendments tracker is seed-only for FEAT-15 — the live Hansard
// committee-stage ingestion path is pending issue #19 + ADR-0011.
export interface Amendment {
  id: string;
  bill_id: string;
  title: string;
  proposed_by: string;
  proposed_at: string;
  status: AmendmentStatus;
  summary: string;
  source_url: string;
}

export interface BillSummary {
  bill_id: string;
  short_title: string;
  one_line_summary: string;
  plain_language_explanation: string;
  current_stage_explained: string;
  what_it_would_do: string[];
  who_it_affects: string[];
  what_happens_next: string[];
  citations: Citation[];
  confidence: ConfidenceLevel;
  validated: boolean;
  validation_failures: string[];
}

export interface BriefingItem {
  kind: 'new_bill' | 'stage_change' | 'amendment' | 'new_document' | 'enacted' | 'committee';
  title: string;
  description: string;
  bill_id: string | null;
  citations: Citation[];
  significance: string;
  confidence: ConfidenceLevel;
}

export interface DailyBriefing {
  country: string;
  date: string;
  headline: string;
  items: BriefingItem[];
  generated_at: string;
}

// --- Civic Daily Brief (task ENG-I2) ---
//
// The personalised Civic Daily Brief is a richer shape than the legacy
// DailyBriefing above: it groups items into titled sections, carries an
// AI-generated plain-language summary (with disclaimer + source label),
// and tracks a total evidence count across all sections. The legacy
// DailyBriefing is kept for backward-compat with the old /api/v1/briefing
// endpoint; new code should use CivicBrief.
//
// Mirrors the Go struct services/api/cmd/brief.go::brief.

export interface BriefSectionItem {
  type: string;
  title?: string;
  description?: string;
  bill?: string;
  change?: string;
  house?: string;
  date?: string;
  significance?: string;
  evidence_url: string;
  // Constitutional-context specific.
  article?: string;
  text?: string;
  connection?: string;
}

export interface BriefSection {
  title: string;
  summary: string;
  items: BriefSectionItem[];
}

export interface CivicBrief {
  id: string;
  date: string;
  headline: string;
  sections: BriefSection[];
  ai_summary: string;
  ai_disclaimer: string;
  ai_source: 'ai-service' | 'template-fallback';
  evidence_count: number;
  generated_at: string;
  country: string;
  user_id?: string;
}

export interface BriefArchiveEntry {
  id: string;
  date: string;
  headline: string;
  evidence_count: number;
  generated_at: string;
}

export interface SearchResult {
  kind: 'bill' | 'act' | 'regulation' | 'hansard' | 'committee_report' | 'order_paper' | 'gazette';
  id: string;
  title: string;
  snippet: string;
  url: string;
  score: number;
  published_at: string | null;
  source_type: string;
}

export interface AIResponse {
  id: string;
  answer: string;
  claims: Array<{
    type: string;
    text: string;
    citations: Citation[];
    confidence: ConfidenceLevel;
  }>;
  citations: Citation[];
  validated: boolean;
  validation_status: 'pending' | 'passed' | 'failed' | 'partial';
  validation_failures: string[];
  model: string;
  provider: string;
  latency_ms: number;
}
