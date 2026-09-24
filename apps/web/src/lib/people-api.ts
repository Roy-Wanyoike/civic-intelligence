// Client for the People + Scorecard + Constituencies API endpoints (task ENG-K2).
//
// The MP Scorecard endpoint returns a factual, evidence-backed record of
// each MP's parliamentary activity. The platform NEVER calculates a
// "performance score" or "MP rating" — only raw counts + rates. The
// Constituency Dashboard endpoint returns per-area civic data: which MP
// represents the area, which bills affect it, what budget has been
// allocated, local projects, public participation opportunities, recent
// civic events and government context.
//
// Both endpoints return a `disclaimer` string and a `reality_layer: "FACT"`
// tag so the frontend can surface the platform's editorial posture verbatim.

import { getJSON_ as getJSON } from './api';

// === MP Scorecard ===

export interface ScorecardMetric {
  value: number;
  source_url: string;
  source: string;
  period?: string;
}

export interface ScorecardBillRef {
  title: string;
  url: string;
  source_url: string;
  house?: string;
}

export interface ScorecardCommitteeRef {
  name: string;
  role: string;
  source_url: string;
}

export interface ScorecardActivityItem {
  kind: 'question' | 'statement' | 'vote' | 'bill_sponsored' | 'committee_meeting';
  date: string;
  title: string;
  detail?: string;
  source_url: string;
  source: string;
}

export interface MPScorecard {
  person_id: string;
  name: string;
  role: string;
  constituency: string;
  party: string;
  parliamentary_period: string;
  photo_url?: string;

  // Contact info — issue #281. Every field is optional because the API
  // uses omitempty (empty values are omitted from the JSON rather than
  // rendered as empty rows). The _note field marks the contact info as
  // seed data pending live scraping from parliament.go.ke.
  email?: string;
  phone?: string;
  office_address?: string;
  twitter?: string;
  facebook?: string;
  _note?: string;

  // 0.0 – 1.0 — the platform does NOT compute a composite performance score.
  attendance_rate: number;
  attendance_source_url: string;

  // Raw integer counts — never weighted aggregates.
  bills_sponsored: number;
  bills_sponsored_list: ScorecardBillRef[];
  committee_memberships: ScorecardCommitteeRef[];
  questions_asked: number;
  votes_recorded: number;
  statements_made: number;

  // Per-metric source URL — lets the citizen verify each datum.
  metrics: Record<string, ScorecardMetric>;
  recent_activity: ScorecardActivityItem[];

  // Editorial posture — must be rendered verbatim by the frontend.
  disclaimer: string;
  reality_layer: 'FACT';
  source_agencies: string[];
}

export interface PeopleListItem {
  person_id: string;
  name: string;
  role: string;
  constituency: string;
  party: string;
  scorecard_url: string;
}

export async function listPeople(): Promise<{
  items: PeopleListItem[];
  total: number;
  disclaimer: string;
}> {
  return getJSON('/api/v1/people');
}

export async function getMPScorecard(personId: string): Promise<MPScorecard> {
  return getJSON(`/api/v1/people/${encodeURIComponent(personId)}/scorecard`);
}

<<<<<<< HEAD
<<<<<<< HEAD
// === Written Questions (issue #286) ===

// WrittenQuestionStatus mirrors the Go WrittenQuestionStatus type
// (services/api/cmd/written_questions.go) 1:1 so the JSON contract
// stays in lock-step. The 3 values are the wire forms emitted by
// the API (lowercase, no spaces).
export type WrittenQuestionStatus = 'pending' | 'answered' | 'overdue';

// WrittenQuestion mirrors the Go WrittenQuestion struct 1:1. Both the
// per-MP endpoint (/people/{id}/questions), the list endpoint
// (/questions/written), and the detail endpoint
// (/questions/written/{id}) return the same item shape so the
// frontend can render a question row identically on any page.
export interface WrittenQuestion {
  id: string;
  mp_id: string;
  mp_name: string;
  minister: string;
  ministry: string;
  question_text: string;
  asked_at: string;
  // response_text + responded_at are omitted (NOT empty string) for
  // pending + overdue questions. The omitempty invariant mirrors the
  // real-world lifecycle: a question is tabled BEFORE a response
  // exists, and the API surface should not invent empty-string
  // responses for unanswered questions.
  response_text?: string;
  responded_at?: string;
  deadline: string;
  status: WrittenQuestionStatus;
  source_url: string;
}

// WrittenQuestionsByPersonResponse is the JSON envelope returned by
// GET /api/v1/people/{id}/questions. The shape mirrors the
// VotesByPersonResponse + billsByPersonResponse envelopes so the
// scorecard page can reuse its rendering pattern
// (mp_id + name + items + total + source + scorecard_url).
export interface WrittenQuestionsByPersonResponse {
  mp_id: string;
  name: string;
  items: WrittenQuestion[];
=======
// === Registered Interests (issue #288) ===

// RegisteredInterestCategory is the closed enum of declaration categories
// the API accepts on the ?category= filter. Mirrors the Go-side
// RegisteredInterestCategory const block in services/api/cmd/interests.go.
export type RegisteredInterestCategory =
  | 'directorship'
  | 'land_property'
  | 'shares'
  | 'gifts'
  | 'other_income'
  | 'loans';

// RegisteredInterest is a single declared interest sourced from the MP's
// official Declaration of Interests Register entry. value_kes is 0 when
// the value is not disclosed / not applicable (e.g. a non-monetary gift)
// — the description text carries the qualifier in that case.
export interface RegisteredInterest {
  person_id: string;
  person_name: string;
  category: RegisteredInterestCategory;
  description: string;
  value_kes: number;
  declared_at: string;
  source_url: string;
  source: string;
}

// RegisteredInterestsResponse is the JSON envelope returned by
// /api/v1/people/{id}/interests. The category field is the echoed
// ?category= filter value when a filter was applied; omitted otherwise.
export interface RegisteredInterestsResponse {
  person_id: string;
  person_name: string;
  items: RegisteredInterest[];
  total: number;
  category?: RegisteredInterestCategory;
  source: 'seed';
  scorecard_url: string;
  disclaimer: string;
}

// getRegisteredInterests fetches the declared interests for the given MP.
// Pass a category to narrow the response to a single declaration category;
// omit it to fetch every declared interest for the MP.
export async function getRegisteredInterests(
  personId: string,
  params: { category?: RegisteredInterestCategory } = {},
): Promise<RegisteredInterestsResponse> {
  const qs = new URLSearchParams();
  if (params.category) qs.set('category', params.category);
  const tail = qs.size ? `?${qs.toString()}` : '';
  return getJSON(
    `/api/v1/people/${encodeURIComponent(personId)}/interests${tail}`,
  );
>>>>>>> origin/main
=======
// === MP Voting Records (issue #284) ===

// VoteKind mirrors the Go VoteKind type (services/api/cmd/votes.go) 1:1
// so the JSON contract stays in lock-step. The 4 values are the wire
// forms emitted by the API (lowercase, no spaces).
export type VoteKind = 'aye' | 'nay' | 'abstain' | 'absent';

// VoteRecord mirrors the Go VoteRecord struct 1:1. Both the per-MP and
// per-Bill endpoints return the same item shape so the frontend can
// render a vote row identically on either page.
export interface VoteRecord {
  person_id: string;
  person_name: string;
  vote: VoteKind;
  bill_id: string;
  bill_title: string;
  division: string;
  date: string;
  source_url: string;
}

// VoteCounts is the per-division tally returned by /bills/{id}/votes.
// The platform NEVER derives a "pass / fail" verdict from these counts
// — they are surfaced raw.
export interface VoteCounts {
  aye: number;
  nay: number;
  abstain: number;
  absent: number;
  total: number;
}

// VotesByPersonResponse is the JSON envelope returned by
// GET /api/v1/people/{id}/votes.
export interface VotesByPersonResponse {
  person_id: string;
  name: string;
  items: VoteRecord[];
>>>>>>> origin/main
  total: number;
  source: string;
  scorecard_url?: string;
}

<<<<<<< HEAD
// getMPWrittenQuestions fetches an MP's written questions tabled to
// Cabinet Secretaries + Ministers (issue #286). Returns the raw
// questions most-recent-first; the scorecard page renders them with a
// status-coloured badge (green=answered, amber=pending, red=overdue).
// The platform NEVER derives a "responsiveness score" from these
// records — the colour coding is a readability affordance ONLY.
export async function getMPWrittenQuestions(
  personId: string,
): Promise<WrittenQuestionsByPersonResponse> {
  return getJSON(`/api/v1/people/${encodeURIComponent(personId)}/questions`);
=======
// VotesByBillResponse is the JSON envelope returned by
// GET /api/v1/bills/{id}/votes.
export interface VotesByBillResponse {
  bill_id: string;
  bill_title: string;
  division: string;
  date: string;
  source_url: string;
  counts: VoteCounts;
  items: VoteRecord[];
  total: number;
  source: string;
}

// getMPVotes fetches an MP's voting history (issue #284). Returns the
// raw vote records most-recent-first; the scorecard page renders them
// with a color-coded badge (green=aye, red=nay, gray=abstain/absent).
export async function getMPVotes(personId: string): Promise<VotesByPersonResponse> {
  return getJSON(`/api/v1/people/${encodeURIComponent(personId)}/votes`);
}

// getBillVotes fetches the division summary for a single Bill (issue
// #284). Returns the per-division tally (aye / nay / abstain / absent /
// total) + the per-MP roll-call items list.
export async function getBillVotes(billId: string): Promise<VotesByBillResponse> {
  return getJSON(`/api/v1/bills/${encodeURIComponent(billId)}/votes`);
>>>>>>> origin/main
}

// === Constituencies ===

export interface ConstituencyBillRef {
  bill_id: string;
  title: string;
  house: string;
  topics: string[];
  tag: string;
  url: string;
  source_url: string;
}

export interface ConstituencyBudgetLine {
  fiscal_year: string;
  vote: string;
  amount_kes_millions: number;
  source_url: string;
  source: string;
}

export interface ConstituencyProject {
  name: string;
  sector: string;
  status: 'ongoing' | 'completed' | 'stalled';
  allocated_kes_millions: number;
  source_url: string;
  gazette_date: string;
}

export interface ConstituencyParticipationOpportunity {
  title: string;
  organiser: string;
  date: string;
  location: string;
  source_url: string;
  status: 'upcoming' | 'held' | 'cancelled';
}

export interface ConstituencyEvent {
  kind: 'bill_published' | 'gazette_notice' | 'committee_meeting' | 'public_participation' | 'budget_allocation';
  date: string;
  title: string;
  detail?: string;
  source_url: string;
  source: string;
}

export interface ConstituencyGovernmentContext {
  administration_id: string;
  administration_name: string;
  parliamentary_term: string;
  county_government: string;
  governor: string;
  source_url: string;
}

export interface ConstituencyDebtContext {
  region_note: string;
  national_debt_kes_billions: number;
  as_of_date: string;
  source_url: string;
  source: string;
}

export interface ConstituencyMPRef {
  person_id: string;
  name: string;
  role: string;
  party: string;
  scorecard_url: string;
  source_url: string;
}

export interface ConstituencyListItem {
  id: string;
  name: string;
  county: string;
  country: string;
  region: string;
  mp_name: string;
  mp_party: string;
  population: number;
  dashboard_url: string;
}

export interface Constituency {
  id: string;
  name: string;
  county: string;
  country: string;
  region: string;
  population: number;
  area_km2: number;

  mp: ConstituencyMPRef;
  bills_affecting_area: ConstituencyBillRef[];
  budget_allocated: ConstituencyBudgetLine[];
  local_projects: ConstituencyProject[];
  public_participation: ConstituencyParticipationOpportunity[];
  recent_events: ConstituencyEvent[];
  government_context: ConstituencyGovernmentContext;
  debt_context: ConstituencyDebtContext;

  disclaimer: string;
  reality_layer: 'FACT';
}

export async function listConstituencies(params: {
  country?: string;
  q?: string;
} = {}): Promise<{
  items: ConstituencyListItem[];
  total: number;
  country: string;
  disclaimer: string;
}> {
  const qs = new URLSearchParams();
  if (params.country) qs.set('country', params.country);
  if (params.q) qs.set('q', params.q);
  const tail = qs.size ? `?${qs.toString()}` : '';
  return getJSON(`/api/v1/constituencies${tail}`);
}

export async function getConstituency(id: string): Promise<Constituency> {
  return getJSON(`/api/v1/constituencies/${encodeURIComponent(id)}`);
}
