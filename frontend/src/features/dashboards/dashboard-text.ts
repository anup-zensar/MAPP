import type { PlanType } from './dashboard-links';

export const NOTHING = '—';
export const FINANCE_HIDDEN = "You can't see financial data";
export const SHARED_PLAN_NOTE =
  'A plan shared by two markets counts in both rows and once in the totals.';

export const PLAN_TYPES: { value: PlanType; label: string }[] = [
  { value: 'operational', label: 'Operational' },
  { value: 'strategic', label: 'Strategic' },
  { value: 'crossMarket', label: 'Cross-market' },
];

const WORDS: Record<string, string> = {
  draft: 'Draft',
  published: 'Published',
  operational: 'Operational',
  strategic: 'Strategic',
  crossMarket: 'Cross-market',
  created: 'Created',
  submitted: 'Submitted',
  declined: 'Declined',
  approved: 'Approved',
  inProgress: 'In progress',
  complete: 'Complete',
  inReview: 'In review',
  closed: 'Closed',
  plan: 'Plans',
  campaign: 'Campaigns',
  initiative: 'Initiatives',
  activity: 'Activities',
  lineItem: 'Line items',
};

export function word(value: string): string {
  return WORDS[value] ?? value;
}

export function percentText(percent: number | null): string {
  return percent === null ? NOTHING : `${percent}%`;
}

export function monthLabel(month: string): string {
  const [year, number] = month.split('-');
  const name = new Intl.DateTimeFormat('en-GB', { month: 'short', timeZone: 'UTC' }).format(
    Date.UTC(Number(year), Number(number) - 1, 1),
  );
  return `${name} ${year}`;
}

export function emptyText(year: number, teamName: string | null): string {
  return teamName ? `No plans of ${teamName} to show for ${year}` : `No plans to show for ${year}`;
}

export function financeCoverage(financePlans: number, plans: number): string | null {
  if (financePlans === 0) {
    return `${FINANCE_HIDDEN} in these plans.`;
  }
  return financePlans < plans ? `Financial figures cover ${financePlans} of ${plans} plans.` : null;
}
