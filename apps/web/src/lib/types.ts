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

export interface Bill {
  id: string;
  identifier: string;
  title: string;
  year: number;
  sponsor_id: string | null;
  sponsor_name: string | null;
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
