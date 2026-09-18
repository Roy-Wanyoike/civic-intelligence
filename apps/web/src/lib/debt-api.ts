// Client for the Public Debt & Borrowing API endpoints (issue #195).

import { getJSON_ as getJSON } from './api';

export interface DebtDashboard {
  country_code: string;
  total_debt_stock?: number;
  domestic_debt?: number;
  external_debt?: number;
  debt_service?: number;
  debt_to_gdp?: number;
  currency: string;
  as_of: string;
  source_url: string;
  disclaimer: string;
}

export interface DebtTrendPoint {
  date: string;
  total_debt_stock?: number;
  domestic_debt?: number;
  external_debt?: number;
  source_url: string;
}

export interface DebtTimeline {
  country_code: string;
  points: DebtTrendPoint[];
  disclaimer: string;
}

export interface GovernmentDebtSummary {
  administration_id: string;
  period: string;
  debt_at_start?: number;
  debt_at_end?: number;
  new_borrowing?: number;
  debt_service?: number;
  external_debt?: number;
  domestic_debt?: number;
  currency: string;
  source_urls: string[];
  disclaimer: string;
}

export async function getDebtDashboard(): Promise<DebtDashboard> {
  return getJSON<DebtDashboard>('/api/v1/debt');
}

export async function getDebtTimeline(): Promise<DebtTimeline> {
  return getJSON<DebtTimeline>('/api/v1/debt/timeline');
}

export async function getGovernmentDebtSummary(adminId: string): Promise<GovernmentDebtSummary> {
  return getJSON<GovernmentDebtSummary>(`/api/v1/debt/governments/${adminId}`);
}
