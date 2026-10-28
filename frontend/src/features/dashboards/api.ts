import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { downloadFromApi } from '@/api/download';
import type { components } from '@/api/generated/schema';
import { toQuery } from './dashboard-links';
import type { DashboardFilters, DashboardKind } from './dashboard-links';

export type PlanningDashboard = components['schemas']['PlanningDashboard'];
export type DashboardFigures = components['schemas']['DashboardFigures'];
export type DashboardMonths = components['schemas']['DashboardMonths'];
export type DashboardApprovals = components['schemas']['DashboardApprovals'];
export type DashboardPurchaseRequests = components['schemas']['DashboardPurchaseRequests'];
export type DashboardCompleteness = components['schemas']['DashboardCompleteness'];
export type DashboardGlobalBudget = components['schemas']['DashboardGlobalBudget'];
export type DashboardCampaignRow = components['schemas']['DashboardCampaignRow'];
export type DashboardPlanRow = components['schemas']['DashboardPlanRow'];
export type DashboardMarketRow = components['schemas']['DashboardMarketRow'];
export type Money = components['schemas']['Money'];
export type ExportFormat = 'pptx' | 'pdf';

export const DASHBOARDS_KEY = ['dashboards'] as const;

export function useDashboard(kind: DashboardKind, filters: DashboardFilters) {
  const query = toQuery(kind, filters);
  return useQuery({
    queryKey: [...DASHBOARDS_KEY, kind, query],
    queryFn: () =>
      kind === 'executive'
        ? call(api.GET('/api/v1/dashboards/executive', { params: { query } }))
        : call(api.GET('/api/v1/dashboards/regional', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}

export function downloadDashboard(
  kind: DashboardKind,
  filters: DashboardFilters,
  format: ExportFormat,
): Promise<string> {
  const query = { ...toQuery(kind, filters), format };
  return kind === 'executive'
    ? downloadFromApi(
        api.GET('/api/v1/dashboards/executive/export', { params: { query }, parseAs: 'blob' }),
      )
    : downloadFromApi(
        api.GET('/api/v1/dashboards/regional/export', { params: { query }, parseAs: 'blob' }),
      );
}
