import type { operations } from '@/api/generated/schema';

export type DashboardKind = 'executive' | 'regional';
export type PlanType = 'operational' | 'strategic' | 'crossMarket';

export interface DashboardFilters {
  year: number | null;
  planType: PlanType | '';
  currency: string;
  team: string;
  plan: string;
}

type RegionalQuery = NonNullable<operations['getRegionalDashboard']['parameters']['query']>;

export const DEFAULT_FILTERS: DashboardFilters = {
  year: null,
  planType: '',
  currency: '',
  team: '',
  plan: '',
};

export function toQuery(kind: DashboardKind, filters: DashboardFilters): RegionalQuery {
  const query: RegionalQuery = {};
  if (filters.year !== null) {
    query.year = filters.year;
  }
  if (filters.planType) {
    query.planType = filters.planType;
  }
  if (filters.currency) {
    query.currency = filters.currency;
  }
  if (kind === 'regional' && filters.team) {
    query.teamId = filters.team;
  }
  if (kind === 'regional' && filters.plan) {
    query.planId = filters.plan;
  }
  return query;
}

export function writeFilters(filters: DashboardFilters): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.year !== null) {
    params.set('year', String(filters.year));
  }
  for (const key of ['planType', 'currency', 'team', 'plan'] as const) {
    if (filters[key]) {
      params.set(key, filters[key]);
    }
  }
  return params;
}

export function dashboardHref(kind: DashboardKind, filters: DashboardFilters): string {
  const search = writeFilters(filters).toString();
  return `/dashboards/${kind}${search ? `?${search}` : ''}`;
}
