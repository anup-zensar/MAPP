/**
 * The Financials page's address (015 plan, "Route and navigation"): the address is the state.
 * Tab, search, year, plan, record filter, sort and page are kept in it, so a link reproduces the
 * view (FR-021). financialsHref() builds those links for the Financials cards, Home and the log.
 */
import type { operations } from '@/api/schema';

/** The query of GET /api/financials/transactions, as the contract types it. */
export type TransactionsQuery = NonNullable<
  operations['listFinancialTransactions']['parameters']['query']
>;

/** The tabs: the four types, everything, and the transactions without a line item. */
export type FinancialTab =
  'all' | 'purchaseOrder' | 'invoice' | 'payment' | 'accrual' | 'notLinked';

export const TABS: FinancialTab[] = [
  'all',
  'purchaseOrder',
  'invoice',
  'payment',
  'accrual',
  'notLinked',
];

/** The columns a person can sort by (contract FinancialSort). */
export type SortColumn =
  'postingDate' | 'documentNumber' | 'type' | 'vendor' | 'amount' | 'linkedTo';
export type SortDirection = 'asc' | 'desc';

export const PAGE_SIZES = [50, 100, 200] as const;
export type PageSize = (typeof PAGE_SIZES)[number];

/** Everything the address can hold; undefined or '' means the default. */
export interface FinancialFilters {
  tab: FinancialTab;
  q: string;
  /** The linked plan's year; null for the API's default, the current year. */
  year: number | null;
  planId: string;
  /** A plan, campaign, initiative, activity or line item (the cards' "View all" links). */
  recordId: string;
  sort: SortColumn;
  direction: SortDirection;
  page: number;
  pageSize: PageSize;
}

export const DEFAULT_FILTERS: FinancialFilters = {
  tab: 'all',
  q: '',
  year: null,
  planId: '',
  recordId: '',
  sort: 'postingDate',
  direction: 'desc',
  page: 1,
  pageSize: 50,
};

const SORT_COLUMNS: SortColumn[] = [
  'postingDate',
  'documentNumber',
  'type',
  'vendor',
  'amount',
  'linkedTo',
];
const TYPE_TABS: FinancialTab[] = ['purchaseOrder', 'invoice', 'payment', 'accrual'];
const PLAN_ID = /^P-[0-9]+$/;
const RECORD_ID = /^[PCIAL]-[0-9]+$/;

/**
 * The filters an address holds. Anything the page cannot use (an unknown tab, a bad year, page 0)
 * falls back to its default, so an old or edited link still opens a working page.
 */
export function readFilters(params: URLSearchParams): FinancialFilters {
  const type = params.get('type') ?? '';
  const link = params.get('link') ?? '';
  const year = Number(params.get('year'));
  const [sort, direction] = (params.get('sort') ?? '').split(':');
  const page = Number(params.get('page'));
  const pageSize = Number(params.get('pageSize'));
  const planId = params.get('planId') ?? '';
  const recordId = params.get('recordId') ?? '';

  return {
    tab:
      link === 'notLinked'
        ? 'notLinked'
        : TYPE_TABS.includes(type as FinancialTab)
          ? (type as FinancialTab)
          : 'all',
    q: (params.get('q') ?? '').slice(0, 200),
    year: Number.isInteger(year) && year >= 2000 && year <= 2100 ? year : null,
    planId: PLAN_ID.test(planId) ? planId : '',
    recordId: RECORD_ID.test(recordId) ? recordId : '',
    sort: SORT_COLUMNS.includes(sort as SortColumn) ? (sort as SortColumn) : DEFAULT_FILTERS.sort,
    direction:
      direction === 'asc' || direction === 'desc'
        ? direction
        : SORT_COLUMNS.includes(sort as SortColumn)
          ? 'asc'
          : DEFAULT_FILTERS.direction,
    page: Number.isInteger(page) && page >= 1 ? page : 1,
    pageSize: PAGE_SIZES.includes(pageSize as PageSize) ? (pageSize as PageSize) : 50,
  };
}

/** The address parameters of the filters, defaults left out ("?type=invoice&q=keynote"). */
export function writeFilters(filters: Partial<FinancialFilters>): URLSearchParams {
  const full = { ...DEFAULT_FILTERS, ...filters };
  const params = new URLSearchParams();
  if (full.tab === 'notLinked') {
    params.set('link', 'notLinked');
  } else if (full.tab !== 'all') {
    params.set('type', full.tab);
  }
  if (full.q.trim() !== '') {
    params.set('q', full.q);
  }
  if (full.year !== null) {
    params.set('year', String(full.year));
  }
  if (full.planId) {
    params.set('planId', full.planId);
  }
  if (full.recordId) {
    params.set('recordId', full.recordId);
  }
  if (full.sort !== DEFAULT_FILTERS.sort || full.direction !== DEFAULT_FILTERS.direction) {
    params.set('sort', `${full.sort}:${full.direction}`);
  }
  if (full.page !== 1) {
    params.set('page', String(full.page));
  }
  if (full.pageSize !== DEFAULT_FILTERS.pageSize) {
    params.set('pageSize', String(full.pageSize));
  }
  return params;
}

/**
 * The Financials page showing these filters: financialsHref({ recordId: 'A-1002' }) is
 * "/financials?recordId=A-1002"; financialsHref({ tab: 'notLinked' }) "/financials?link=notLinked".
 */
export function financialsHref(filters: Partial<FinancialFilters> = {}): string {
  const query = writeFilters(filters).toString();
  return query ? `/financials?${query}` : '/financials';
}

/** A transaction's details panel, keeping the list's filters behind it (FR-022). */
export function transactionHref(transactionId: string, search = ''): string {
  return `/financials/transactions/${encodeURIComponent(transactionId)}${search}`;
}

/** The API query for the filters (the same parameters, typed for the contract). */
export function toQuery(filters: FinancialFilters): TransactionsQuery {
  const query: TransactionsQuery = {
    type: filters.tab === 'all' || filters.tab === 'notLinked' ? 'all' : filters.tab,
    link: filters.tab === 'notLinked' ? 'notLinked' : 'all',
    sort: `${filters.sort}:${filters.direction}`,
    page: filters.page,
    pageSize: filters.pageSize,
  };
  if (filters.q.trim() !== '') {
    query.q = filters.q.trim();
  }
  if (filters.year !== null) {
    query.year = filters.year;
  }
  if (filters.planId) {
    query.planId = filters.planId;
  }
  if (filters.recordId) {
    query.recordId = filters.recordId;
  }
  return query;
}
