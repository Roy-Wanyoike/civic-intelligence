// Client for the Constitution + Government API endpoints (issue #192).

import { getJSON_ as getJSON } from './api';

export interface President {
  id: string;
  country_code: string;
  full_name: string;
  display_name: string;
  biography_url?: string;
  photo_url?: string;
}

export interface Administration {
  id: string;
  country_code: string;
  president_id: string;
  name: string;
  start_date: string;
  end_date?: string;
  government_system: string;
  source_url?: string;
}

export interface PresidentialTerm {
  id: string;
  administration_id: string;
  president_id: string;
  term_number: number;
  start_date: string;
  end_date?: string;
  status: 'CURRENT' | 'COMPLETED' | 'INTERRUPTED';
  source_url?: string;
}

export interface GovernmentTransition {
  id: string;
  transition_date: string;
  outgoing_admin_id?: string;
  incoming_admin_id: string;
  outgoing_president?: string;
  incoming_president: string;
  source_url?: string;
}

export interface Constitution {
  id: string;
  country_code: string;
  title: string;
  promulgated_at: string;
  assented_at?: string;
  version: string;
  source_url: string;
}

// ConstitutionChapter mirrors the JSON shape returned by GET
// /api/v1/constitution/articles (services/api/cmd/governments.go). The
// chapter carries its articles inline — the list endpoint returns the full
// chapter+article tree.
export interface ConstitutionChapter {
  id: string;
  constitution_id: string;
  number: number;
  title: string;
  articles: ConstitutionArticle[];
}

// ConstitutionArticle mirrors the JSON shape of an article in the
// /api/v1/constitution/articles response. The Number field is a display
// string like "Article 1" or "Article 10" because Kenya Law numbers
// articles with names, not sequential integers.
export interface ConstitutionArticle {
  id: string;
  chapter_id: string;
  number: string;
  title: string;
  text: string;
  source_url: string;
  cross_references?: unknown[];
}

// ConstitutionArticlesResponse is the full response from
// GET /api/v1/constitution/articles. RealityLayer is always FACT —
// the Constitution is authoritative source material the platform never
// reinterprets.
export interface ConstitutionArticlesResponse {
  chapters: ConstitutionChapter[];
  count: number;
  reality_layer: 'FACT';
  disclaimer: string;
}

export interface AdministrationListItem extends Administration {
  president_name: string;
  is_current: boolean;
}

export async function listAdministrations(): Promise<{ administrations: AdministrationListItem[]; count: number }> {
  return getJSON('/api/v1/governments');
}

export async function getAdministration(id: string): Promise<{
  administration: Administration;
  president: President;
  terms: PresidentialTerm[];
}> {
  return getJSON(`/api/v1/governments/${id}`);
}

export async function getConstitution(): Promise<{ constitution: Constitution; disclaimer: string; reality_layer: 'FACT' }> {
  return getJSON('/api/v1/constitution');
}

// listConstitutionArticles returns the curated chapter + article tree from
// the Go BFF (issue #212). Callers should fall back to the hardcoded
// constitution-articles.ts data file when this throws — see
// apps/web/src/app/constitution/page.tsx for the fallback pattern.
export async function listConstitutionArticles(): Promise<ConstitutionArticlesResponse> {
  return getJSON('/api/v1/constitution/articles');
}

export async function listTransitions(): Promise<{ transitions: GovernmentTransition[]; count: number }> {
  return getJSON('/api/v1/transitions');
}
