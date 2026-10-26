import type { components } from '@/api/generated/schema';
import type { Answer } from '../replies';
import { MADE, NOT_USED, USED_BY_TWO_PLANS } from './settings';

type FinanceBudget = components['schemas']['FinanceBudget'];
type FinanceBudgetItem = components['schemas']['FinanceBudgetItem'];

export const YEAR_LOCK =
  "The year can't change once the item is saved — create a new item for another year.";

export function item(overrides: Partial<FinanceBudgetItem> & { id: string; name: string }) {
  return {
    budgetId: 'FB-1001',
    budgetName: '2026 Lubricants Budget',
    year: 2026,
    code: '',
    active: true,
    usage: NOT_USED,
    locks: [{ field: 'year', reason: YEAR_LOCK }],
    version: 1,
    ...MADE,
    ...overrides,
  } satisfies FinanceBudgetItem;
}

export const DIGITAL = item({
  id: 'FBI-1001',
  name: 'Europe – Digital',
  code: 'LUB-EU-DIG',
  usage: {
    inUse: true,
    summary: 'Linked to 6 line items',
    parts: [{ kind: 'lineItem', text: 'Linked to 6 line items', count: 6, inRecycleBin: 0 }],
    records: [],
  },
  version: 2,
});
export const ASIA = item({ id: 'FBI-1002', name: 'Asia – Retail' });
export const OLD = item({ id: 'FBI-1003', name: 'Old item', active: false });
const FLEET = item({
  id: 'FBI-1004',
  name: 'Fleet',
  code: 'MOB-FLT',
  budgetId: 'FB-1002',
  budgetName: '2026 Mobility Budget',
});

/** "2026 Lubricants Budget", the default budget of two plans. */
export const LUBRICANTS_BUDGET: FinanceBudget = {
  id: 'FB-1001',
  year: 2026,
  name: '2026 Lubricants Budget',
  items: [ASIA, DIGITAL, OLD],
  activeItemCount: 2,
  usage: {
    ...USED_BY_TWO_PLANS,
    summary: 'Default finance budget of 2 plans',
    parts: [{ kind: 'plan', text: 'Default finance budget of 2 plans', count: 2, inRecycleBin: 0 }],
  },
  version: 4,
  ...MADE,
};

export const MOBILITY_BUDGET: FinanceBudget = {
  ...LUBRICANTS_BUDGET,
  id: 'FB-1002',
  name: '2026 Mobility Budget',
  items: [FLEET],
  activeItemCount: 1,
  usage: NOT_USED,
  version: 1,
};

const YEARS = [2025, 2026, 2027, 2028, 2029, 2030, 2031];

export const PAGE_2026 = {
  items: [LUBRICANTS_BUDGET, MOBILITY_BUDGET],
  total: 2,
  year: 2026,
  years: YEARS,
};
const PAGE_2027 = { items: [], total: 0, year: 2027, years: YEARS };

/** The finance budgets of the year asked for (this year, 2026, when none is given). */
export const BUDGETS: Answer<'GET /api/v1/finance-budgets'> = (request) =>
  new URL(request.url).searchParams.get('year') === '2027'
    ? { body: PAGE_2027 }
    : { body: PAGE_2026 };
