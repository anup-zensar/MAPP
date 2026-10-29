export const MORE_FILTERS = 'More filters';
export const SEARCH_ATTRIBUTES = 'Search attributes and values';
export const NOTHING_TO_CHOOSE = 'Nothing to choose from.';

export const BREAKDOWN = 'Breakdown';
export const BREAK_DOWN_BY = 'Break down by';
export const THEN_BY = 'then by';
export const NOTHING_ELSE = 'Nothing else';
export const PLANNED_COST_BY = 'Planned cost by';
export const NO_PLANNED_COST = 'No planned cost to break down yet.';
export const NO_PLANNED_COST_TEXT =
  'Planned cost appears here once activities or line items have amounts.';

export const LEVEL_LABELS: Record<string, string> = {
  plan: 'Plan',
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
  lineItem: 'Line item',
};

export const COLUMN_LABELS = {
  activities: 'Activities',
  lineItems: 'Line items',
  plannedCost: 'Planned cost',
  share: 'Share of planned cost',
  budget: 'Budget',
  budgetUsed: 'Budget used',
} as const;

export function showInGridText(label: string): string {
  return `Show '${label}' in the Grid`;
}

export function groupToggleText(label: string, open: boolean): string {
  return open ? `Hide the groups of '${label}'` : `Show the groups of '${label}'`;
}
