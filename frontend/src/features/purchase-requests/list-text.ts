import { formatWhole } from '@/format/decimal';
import type { ListSort } from './schemas';

export const LIST_DESCRIPTION =
  'Purchase requests of your plans, grouped by vendor. The requisitioner works from Waiting for GSAP.';
export const SEARCH_LABEL = 'Search purchase requests';
export const SEARCH_PLACEHOLDER = 'Search ID, name, vendor, HX key, GSAP PR or PO number';
export const VIEW_ALL = 'All';
export const VIEW_WAITING = 'Waiting for GSAP';
export const WAITING_MARK = 'Waiting for GSAP';
export const NOT_COMPLETED = 'Not completed';
export const NO_PURCHASE_REQUESTS =
  "No purchase requests yet — purchase requests are raised from a line item's Purchase requests section.";
export const NOTHING_WAITING = 'No purchase request is waiting for GSAP.';
export const NO_MATCH = 'Nothing matches these filters';
export const CLEAR_FILTERS = 'Clear filters';
export const TABLE_LABEL = 'Purchase requests by vendor';

export const FILTER_LABELS = {
  status: 'Status',
  type: 'PR type',
  vendor: 'Vendor',
  owner: 'Owner',
  from: 'Created from',
  to: 'Created to',
} as const;

export const SORT_WORDS: Record<ListSort, string> = {
  id: 'ID, lowest first',
  '-id': 'ID, highest first',
  name: 'Name, A to Z',
  '-name': 'Name, Z to A',
  status: 'Status, A to Z',
  '-status': 'Status, Z to A',
  amount: 'Amount, smallest first',
  '-amount': 'Amount, largest first',
  created: 'Created, oldest first',
  '-created': 'Created, newest first',
};

/** "1 purchase request", "12 purchase requests": the matching ones, or a vendor's. */
export function countText(count: number): string {
  return count === 1 ? '1 purchase request' : `${formatWhole(count)} purchase requests`;
}

/** "Page 2 of 3". */
export function pageText(page: number, pages: number): string {
  return `Page ${formatWhole(page)} of ${formatWhole(pages)}`;
}
