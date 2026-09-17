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

export async function listTransitions(): Promise<{ transitions: GovernmentTransition[]; count: number }> {
  return getJSON('/api/v1/transitions');
}
