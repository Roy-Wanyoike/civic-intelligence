// Client for the Phase 18 simulation API. All responses carry an explicit
// reality_layer tag so the UI can never confuse simulated output with
// observed civic fact.

import { ApiError, getJSON_ as getJSON, postJSON } from './api';

export interface ScenarioAssumption {
  id: string;
  scenario_id: string;
  statement: string;
  value?: unknown;
  unit?: string;
  basis?: string;
  source_evidence?: ScenarioEvidenceRef[];
  assumption_type:
    | 'OBSERVED_INPUT'
    | 'USER_DEFINED'
    | 'MODEL_ASSUMPTION'
    | 'HISTORICAL_REFERENCE'
    | 'ESTIMATE'
    | 'UNKNOWN';
  confidence: 'HIGH' | 'MEDIUM' | 'LOW' | 'UNKNOWN';
  created_at?: string;
}

export interface ScenarioEvidenceRef {
  kind: string;
  id: string;
  source_url?: string;
  retrieved_at?: string;
  verification_status?: string;
}

export interface ScenarioVariable {
  id: string;
  name: string;
  type:
    | 'INTEGER'
    | 'DECIMAL'
    | 'PERCENTAGE'
    | 'CURRENCY'
    | 'DATE'
    | 'DURATION'
    | 'BOOLEAN'
    | 'CATEGORICAL'
    | 'GEOGRAPHIC'
    | 'ENTITY_REFERENCE';
  unit?: string;
  value?: unknown;
  minimum?: number;
  maximum?: number;
  default?: unknown;
  source?: string;
  assumption_status:
    | 'OBSERVED_INPUT'
    | 'USER_DEFINED'
    | 'MODEL_ASSUMPTION'
    | 'HISTORICAL_REFERENCE'
    | 'ESTIMATE'
    | 'UNKNOWN';
}

export interface ScenarioBaseline {
  description: string;
  as_of?: string;
  source_refs?: ScenarioEvidenceRef[];
  reality_layer: 'OBSERVED';
}

export interface Scenario {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type:
    | 'POLICY'
    | 'LEGISLATIVE'
    | 'IMPLEMENTATION'
    | 'COMPARATIVE'
    | 'HISTORICAL_COUNTERFACTUAL'
    | 'INSTITUTIONAL'
    | 'ECONOMIC_SOCIAL'
    | 'INFRASTRUCTURE';
  jurisdiction: string;
  reality_layer: 'HYPOTHETICAL';
  created_by?: string;
  created_at?: string;
  updated_at?: string;
  baseline: ScenarioBaseline;
  assumptions: ScenarioAssumption[];
  variables: ScenarioVariable[];
  time_horizon: { start: string; end: string; duration?: string };
  source_evidence?: ScenarioEvidenceRef[];
  methodology?: {
    description?: string;
    inputs?: string[];
    outputs?: string[];
    limitations?: string[];
    engine_name?: string;
    engine_version?: string;
  };
  model_id: string;
  model_version: string;
  status:
    | 'DRAFT'
    | 'CONFIGURED'
    | 'VALIDATING'
    | 'READY'
    | 'RUNNING'
    | 'COMPLETED'
    | 'REVIEW_REQUIRED'
    | 'ARCHIVED'
    | 'FAILED';
  scenario_version: number;
}

export interface ScenarioListResponse {
  scenarios: Scenario[];
  count: number;
  disclaimer: string;
}

export interface ScenarioRunResultSummary {
  status: string;
  headline_metric?: string;
  headline_value?: unknown;
  uncertainty?: {
    has_uncertainty: boolean;
    median?: number;
    p10?: number;
    p90?: number;
    min?: number;
    max?: number;
    notes?: string;
  };
  limitations?: string[];
  generated_at?: string;
}

export interface ScenarioRunResult {
  run_id: string;
  completed_at?: string;
  duration?: string;
  summary: ScenarioRunResultSummary;
}

export interface ScenarioResultsResponse {
  results: ScenarioRunResult[];
  count: number;
  reality_layer: 'MODELED';
  disclaimer: string;
}

export interface ScenarioTimelineEvent {
  timestamp: string;
  label: 'OBSERVED' | 'ASSUMED' | 'MODELED' | 'UNKNOWN';
  title: string;
  description?: string;
}

export interface ScenarioTimelineResponse {
  timeline: ScenarioTimelineEvent[];
  disclaimer: string;
}

export interface ScenarioMethodologyResponse {
  scenario_id: string;
  methodology: NonNullable<Scenario['methodology']>;
  model_id: string;
  model_version: string;
  limitations: string[];
  reality_layer: 'HYPOTHETICAL';
  disclaimer: string;
}

export interface ScenarioComparisonResponse {
  scenarios: Scenario[];
  dimensions: { name: string; description: string }[];
  disclaimer: string;
}

const BASE = '/api/v1/scenarios';

export async function listScenarios(): Promise<ScenarioListResponse> {
  return getJSON<ScenarioListResponse>(BASE);
}

export async function getScenario(id: string): Promise<{ scenario: Scenario; reality_layer: string; disclaimer: string }> {
  return getJSON(`${BASE}/${id}`);
}

export async function listAssumptions(id: string): Promise<{ assumptions: ScenarioAssumption[]; count: number }> {
  return getJSON(`${BASE}/${id}/assumptions`);
}

export async function listEvidence(id: string): Promise<{ source_evidence: ScenarioEvidenceRef[]; baseline_evidence: ScenarioEvidenceRef[]; assumption_evidence_count: number }> {
  return getJSON(`${BASE}/${id}/evidence`);
}

export async function listResults(id: string): Promise<ScenarioResultsResponse> {
  return getJSON(`${BASE}/${id}/results`);
}

export async function getTimeline(id: string): Promise<ScenarioTimelineResponse> {
  return getJSON(`${BASE}/${id}/timeline`);
}

export async function getMethodology(id: string): Promise<ScenarioMethodologyResponse> {
  return getJSON(`${BASE}/${id}/methodology`);
}

export async function runScenario(id: string, seed = 42): Promise<{ run: unknown; reality_layer: string; disclaimer: string }> {
  return postJSON(`${BASE}/${id}?action=run`, { seed });
}

export async function compareScenarios(scenarioIds: string[]): Promise<ScenarioComparisonResponse> {
  return postJSON(`${BASE}/compare`, { scenario_ids: scenarioIds });
}
