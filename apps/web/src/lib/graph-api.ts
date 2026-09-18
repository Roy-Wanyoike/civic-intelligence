// Client for the Civic Knowledge Graph API endpoints (ENG-I1, Wave 9).
//
// The graph is the platform's signature differentiator — a visual
// relationship explorer that traces how Bills, Acts, Institutions, People,
// Constitution Articles, Administrations, and Government borrowing are
// connected. The shape below mirrors the Go structs in
// services/api/cmd/graph.go so a change to the API contract surfaces as a
// TypeScript compile error here.

import { getJSON_ as getJSON } from './api';

export type GraphNodeType =
  | 'bill'
  | 'act'
  | 'institution'
  | 'person'
  | 'constitution_article'
  | 'administration'
  | 'presidential_term'
  | 'legislature'
  | 'committee'
  | 'creditor'
  | 'borrowing_agreement';

export type GraphEdgeType =
  | 'ORIGINATES_FROM'
  | 'SPONSORED_BY'
  | 'BELONGS_TO'
  | 'ASSESSED_BY'
  | 'CITES'
  | 'ESTABLISHES'
  | 'GOVERNED_BY'
  | 'ASSESNTED_BY'
  | 'BORROWED_BY'
  | 'CONTRACTED_DURING'
  | 'MEMBER_OF'
  | 'LEADS';

export interface GraphNode {
  id: string;
  type: GraphNodeType;
  label: string;
  properties?: Record<string, unknown>;
}

export interface GraphEdge {
  source: string;
  target: string;
  type: GraphEdgeType;
  properties?: Record<string, unknown>;
}

export interface GraphMetadata {
  total_nodes: number;
  total_edges: number;
  depth: number;
}

export interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
  metadata: GraphMetadata;
}

export interface GraphSummary {
  total_nodes: number;
  total_edges: number;
  node_types: Record<string, number>;
  edge_types: Record<string, number>;
  reality_layer: string;
  disclaimer: string;
}

// NODE_TYPE_META carries display metadata for each node type: the human
// label, the Tailwind colour classes, and a Tailwind text colour for the
// side panel / legend. Colours follow the design spec:
//   bills=blue, acts=green, institutions=orange, people=purple,
//   articles=gold, plus civic-themed colours for the remaining types.
export const NODE_TYPE_META: Record<GraphNodeType, {
  label: string;
  fill: string;
  stroke: string;
  text: string;
  badge: string;
}> = {
  bill: {
    label: 'Bill',
    fill: '#dbeafe',
    stroke: '#2563eb',
    text: 'text-blue-800',
    badge: 'bg-blue-100 text-blue-800',
  },
  act: {
    label: 'Act',
    fill: '#dcfce7',
    stroke: '#16a34a',
    text: 'text-green-800',
    badge: 'bg-green-100 text-green-800',
  },
  institution: {
    label: 'Institution',
    fill: '#ffedd5',
    stroke: '#ea580c',
    text: 'text-orange-800',
    badge: 'bg-orange-100 text-orange-800',
  },
  person: {
    label: 'Person',
    fill: '#f3e8ff',
    stroke: '#9333ea',
    text: 'text-purple-800',
    badge: 'bg-purple-100 text-purple-800',
  },
  constitution_article: {
    label: 'Article',
    fill: '#fef9c3',
    stroke: '#ca8a04',
    text: 'text-yellow-800',
    badge: 'bg-yellow-100 text-yellow-800',
  },
  administration: {
    label: 'Administration',
    fill: '#fee2e2',
    stroke: '#b91c1c',
    text: 'text-red-800',
    badge: 'bg-red-100 text-red-800',
  },
  presidential_term: {
    label: 'Term',
    fill: '#ffe4e6',
    stroke: '#be123c',
    text: 'text-rose-800',
    badge: 'bg-rose-100 text-rose-800',
  },
  legislature: {
    label: 'Legislature',
    fill: '#e0e7ff',
    stroke: '#4338ca',
    text: 'text-indigo-800',
    badge: 'bg-indigo-100 text-indigo-800',
  },
  committee: {
    label: 'Committee',
    fill: '#cffafe',
    stroke: '#0e7490',
    text: 'text-cyan-800',
    badge: 'bg-cyan-100 text-cyan-800',
  },
  creditor: {
    label: 'Creditor',
    fill: '#f1f5f9',
    stroke: '#475569',
    text: 'text-slate-700',
    badge: 'bg-slate-100 text-slate-700',
  },
  borrowing_agreement: {
    label: 'Loan',
    fill: '#fce7f3',
    stroke: '#be185d',
    text: 'text-pink-800',
    badge: 'bg-pink-100 text-pink-800',
  },
};

// EDGE_TYPE_META carries a human label + colour for each edge type, used by
// the legend + path display.
export const EDGE_TYPE_META: Record<GraphEdgeType, { label: string; colour: string }> = {
  ORIGINATES_FROM: { label: 'Bill → Act', colour: '#16a34a' },
  SPONSORED_BY: { label: 'Bill → Person (sponsor)', colour: '#9333ea' },
  BELONGS_TO: { label: 'Bill → Legislature', colour: '#4338ca' },
  ASSESSED_BY: { label: 'Bill → Committee', colour: '#0e7490' },
  CITES: { label: 'Bill → Article', colour: '#ca8a04' },
  ESTABLISHES: { label: 'Article → Institution', colour: '#ea580c' },
  GOVERNED_BY: { label: 'Bill → Administration', colour: '#b91c1c' },
  ASSESNTED_BY: { label: 'Act → President', colour: '#be123c' },
  BORROWED_BY: { label: 'Agreement → Creditor', colour: '#475569' },
  CONTRACTED_DURING: { label: 'Agreement → Administration', colour: '#b91c1c' },
  MEMBER_OF: { label: 'Person → Committee', colour: '#0e7490' },
  LEADS: { label: 'Person → Institution', colour: '#ea580c' },
};

// --- API helpers ---

export async function getGraphSummary(): Promise<GraphSummary> {
  return getJSON<GraphSummary>('/api/v1/graph');
}

export async function listGraphNodes(params: {
  type?: GraphNodeType;
  limit?: number;
  q?: string;
} = {}): Promise<GraphResponse> {
  const qs = new URLSearchParams();
  if (params.type) qs.set('type', params.type);
  if (params.limit) qs.set('limit', String(params.limit));
  if (params.q) qs.set('q', params.q);
  return getJSON<GraphResponse>(`/api/v1/graph/nodes?${qs.toString()}`);
}

export async function getGraphNode(id: string): Promise<GraphResponse> {
  return getJSON<GraphResponse>(`/api/v1/graph/node/${encodeURIComponent(id)}`);
}

export async function getGraphRelationships(
  id: string,
  depth: 1 | 2 | 3 = 1,
): Promise<GraphResponse> {
  return getJSON<GraphResponse>(
    `/api/v1/graph/relationships?id=${encodeURIComponent(id)}&depth=${depth}`,
  );
}

export async function searchGraphNodes(q: string, limit = 20): Promise<GraphResponse> {
  const qs = new URLSearchParams({ q });
  if (limit) qs.set('limit', String(limit));
  return getJSON<GraphResponse>(`/api/v1/graph/search?${qs.toString()}`);
}

export async function findGraphPath(from: string, to: string): Promise<GraphResponse> {
  const qs = new URLSearchParams({ from, to });
  return getJSON<GraphResponse>(`/api/v1/graph/paths?${qs.toString()}`);
}
