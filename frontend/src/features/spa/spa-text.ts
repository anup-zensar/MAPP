export const TITLE = 'SP&A view';
export const WHOLE_PLAN = 'Whole plan';

export const HEADLINE = {
  plannedCost: {
    label: 'Planned cost',
    tip: 'The planned cost of the activities in scope that falls in the plan’s year.',
  },
  committed: {
    label: 'Committed',
    tip: 'Purchase orders GSAP has issued for their line items, posted in the plan’s year.',
  },
  actualSpend: {
    label: 'Actual spend',
    tip: 'Payments GSAP has released for their line items in the plan’s year — Shell’s actuals.',
  },
  remaining: { label: 'Remaining', tip: 'Planned cost minus actual spend.' },
  utilisation: { label: 'Utilisation', tip: 'Actual spend as a share of planned cost.' },
  variance: {
    label: 'Variance',
    tip: 'Actual spend minus planned cost, as a share of planned cost.',
  },
} as const;

export const OVERSPEND = 'Overspend';

export const PERIODS = {
  title: 'Spend by period',
  tip: 'Planned cost by the line items’ spreads (an activity’s cost not on line items evenly over its months); actual spend by the month GSAP posted it.',
  planned: 'Planned',
  actual: 'Actual spend',
  notScheduled: 'Not yet scheduled',
  notScheduledTip:
    'Planned cost that is not in a month yet, such as that of activities without dates.',
  linesAbove: 'Line items above planned cost',
  linesAboveTip:
    'How much the line items of some activities exceed their planned cost. The periods include those line items, so periods + not yet scheduled − this = planned cost.',
};

export const DIMENSION = {
  title: 'Spend by dimension',
  choose: 'Group by',
  share: 'Share',
  countsInEach: 'Activities with several values count in each of them; the total counts them once.',
};

export const ACTIVITIES = {
  title: 'Activities',
  raise: 'Raise purchase request',
  lineItems: (name: string) => `Line items of ${name}`,
  noLineItems: 'No line items yet.',
};

export const COLUMNS: Record<string, string> = {
  name: 'Activity',
  campaign: 'Campaign',
  owner: 'Owner',
  activityType: 'Activity type',
  channel: 'Channel / audience',
  startQuarter: 'Starts',
  status: 'Status',
  plannedCost: 'Planned cost',
  committed: 'Committed',
  actualSpend: 'Actual spend',
  variancePercent: 'Variance',
};

export const BUDGET = {
  title: 'Budget',
  budget: 'Budget',
  plannedVsBudget: 'Planned vs budget',
  linked: 'Linked to a finance budget item',
  notLinked: 'Not linked',
  overBudget: 'Over budget',
  tip: 'The plan’s budget, or the campaign’s, against the planned cost and what is committed.',
};

export const PROCUREMENT = {
  title: 'Purchase requests',
  tip: 'Purchase requests raised on the line items in scope; only their lines on those line items count.',
  count: 'Purchase requests',
  approved: 'Approved',
  pendingApproval: 'Pending approval',
  draft: 'Draft',
  invoiced: 'Invoiced',
  none: 'No purchase requests on these line items yet.',
  notCounted: 'Not counted',
};

export const TRACE = {
  title: (figure: string) => `What makes up ${figure}`,
  none: 'No activity adds to this figure.',
  total: 'Total',
  close: 'Close',
};

export const NOT_FOUND = 'This plan no longer exists, or you don’t have access to it.';
export const FINANCE_NONE = 'No finance data received yet.';
export const financeAsOf = (when: string) => `Finance data as of ${when}`;

export function percentText(percent: string | null | undefined, signed = false): string {
  if (percent === null || percent === undefined) {
    return '—';
  }
  const sign = signed && !percent.startsWith('-') && percent !== '0.0' ? '+' : '';
  return `${sign}${percent.replace('-', '−')}%`;
}
