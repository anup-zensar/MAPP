import { screen, within } from '@testing-library/react';
import type { NavigateFunction } from 'react-router';
import { campaignRows, CELL_KEYS } from '@/features/plans/campaigns/campaign-rows';
import type { RowOptions } from '@/features/plans/campaigns/campaign-rows';
import { totalsFooter } from '@/features/plans/campaigns/totals-footer';
import type { PlanCampaigns } from '@/features/plans/campaigns/api';
import { TreeTable } from '@/ui/TreeTable/TreeTable';
import type { TreeNode } from '@/ui/TreeTable/TreeTable';
import {
  DEFAULT_DATE_FORMAT,
  formatDate,
  formatDateRange,
  formatDateTime,
  formatMoney,
  formatMoneyCompact,
  formatMonth,
  formatRelativeTime,
  formatWeekdayDate,
  parseDate,
} from '@/format/intl';
import type { Formatters } from '@/format/useFormatters';
import { closestTo } from '../dom';
import { renderWithProviders } from '../render';
import { budgetAndCost, campaign, initiative, planCampaigns, usd } from './campaigns';

const NOW = new Date('2026-03-12T09:00:00Z');

/** The formatters of a person with the default date format and display currency. */
export const FORMATTERS: Formatters = {
  dateFormat: DEFAULT_DATE_FORMAT,
  displayCurrency: 'USD',
  date: (date) => formatDate(date),
  weekdayDate: (date) => formatWeekdayDate(date),
  month: (month) => formatMonth(month),
  dateTime: (timestamp) => formatDateTime(timestamp),
  dateRange: (start, end) => formatDateRange(start, end),
  relativeTime: (timestamp) => formatRelativeTime(timestamp, NOW),
  parseDate: (text) => parseDate(text),
  money: (value) => (value ? formatMoney(value.amount, value.currency) : '—'),
  moneyCompact: (value) => (value ? formatMoneyCompact(value.amount, value.currency) : '—'),
};

export function rowOptions(
  navigate: NavigateFunction,
  overrides: Partial<RowOptions> = {},
): RowOptions {
  return { planId: 'P-1001', formatters: FORMATTERS, navigate, ...overrides };
}

function allIds(nodes: TreeNode[]): string[] {
  return nodes.flatMap((node) => [node.id, ...allIds(node.children)]);
}

/** The rows as the plan tree shows them, every row expanded. */
export function renderRows(data: PlanCampaigns, options: RowOptions) {
  const nodes = campaignRows(data, options);
  renderWithProviders(
    <TreeTable
      label="Plan tree"
      columns={CELL_KEYS.map((key) => ({ key, header: key }))}
      nodes={nodes}
      expandedIds={new Set(allIds(nodes))}
      onToggle={() => {}}
      footer={totalsFooter(data, FORMATTERS)}
    />,
  );
  return screen.getByRole('treegrid', { name: 'Plan tree' });
}

export function rowOf(tree: HTMLElement, name: string): HTMLElement {
  return closestTo(within(tree).getByRole('link', { name }), 'tr');
}

export const HELIX = campaign({
  budget: usd('120000.00'),
  outsidePlanYear: true,
  endDate: '2027-03-31',
  budgetAndCost: budgetAndCost({
    allocatedToChildren: usd('150000.00'),
    remainingForChildren: usd('-30000.00'),
    overAllocatedBy: usd('30000.00'),
    plannedCost: usd('150000.00'),
    budgetUsedPercent: 125,
    overBudget: true,
  }),
  counts: { initiatives: 2, activities: 5, approvedActivities: 3, lineItems: 0 },
  approvalProgress: { state: 'inProgress', approved: 3, total: 5 },
  initiatives: [
    initiative({
      owner: { id: 'U-1009', name: 'Sam Lee', initials: 'SL', jobTitle: '', active: false },
      counts: { activities: 1, approvedActivities: 0, lineItems: 0 },
    }),
    initiative({
      id: 'I-1002',
      name: 'Events',
      startDate: null,
      endDate: null,
      budget: null,
      timing: null,
    }),
  ],
});
export const RIMULA = campaign({
  id: 'C-1002',
  name: 'Rimula fleet',
  startDate: '2026-03-01',
  endDate: '2026-12-31',
  budget: usd('200000.00'),
  budgetAndCost: budgetAndCost({ remainingForChildren: usd('200000.00') }),
  counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
  initiatives: [],
});
export const NO_BUDGET = campaign({
  id: 'C-1003',
  name: 'Sponsorships',
  startDate: null,
  endDate: null,
  budget: null,
  timing: null,
  counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
  initiatives: [],
});
export const DATA = planCampaigns([HELIX, RIMULA, NO_BUDGET], {
  campaigns: 3,
  initiatives: 2,
  activities: 1,
  allocatedToCampaigns: usd('320000.00'),
  plannedCost: usd('150000.00'),
});
