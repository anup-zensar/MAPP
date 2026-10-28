import { z } from 'zod';
import type { PurchaseRequestListQuery, PurchaseRequestStatus } from './api';

export const LIST_SORTS = [
  'id',
  '-id',
  'name',
  '-name',
  'status',
  '-status',
  'amount',
  '-amount',
  'created',
  '-created',
] as const;
export type ListSort = (typeof LIST_SORTS)[number];

export const LIST_PAGE_SIZE = 50;
const MAX_SEARCH = 120;

const statusSchema = z.enum(['created', 'submitted', 'inReview', 'declined', 'approved', 'closed']);
const textSchema = z.string().trim().min(1).max(MAX_SEARCH);
const idSchema = z.string().regex(/^[A-Za-z0-9-]{1,40}$/);
const daySchema = z.iso.date();
const flagSchema = z.literal('true').transform(() => true);

/** Keeps the values a schema accepts, each once, in their order. */
function each<T>(schema: z.ZodType<T>) {
  return z
    .array(z.unknown())
    .transform((values) =>
      values.flatMap((value) => {
        const result = schema.safeParse(value);
        return result.success ? [result.data] : [];
      }),
    )
    .transform((values) => values.filter((value, index) => values.indexOf(value) === index));
}

/**
 * The Purchase requests page's address (FR-045): anything it cannot use falls back to its
 * default, so an old or edited link still opens a working page.
 */
export const purchaseRequestListSearchSchema = z.object({
  q: z
    .string()
    .catch('')
    .transform((text) => text.slice(0, MAX_SEARCH)),
  status: each(statusSchema),
  type: each(idSchema),
  typeNotCompleted: flagSchema.catch(false),
  vendor: each(textSchema),
  vendorNotCompleted: flagSchema.catch(false),
  owner: each(idSchema),
  from: daySchema.catch(''),
  to: daySchema.catch(''),
  waiting: flagSchema.catch(false),
  sort: z.enum(LIST_SORTS).catch('id'),
  page: z.coerce.number().int().min(1).catch(1),
});

export type ListSearch = z.infer<typeof purchaseRequestListSearchSchema>;

export const DEFAULT_LIST_SEARCH: ListSearch = {
  q: '',
  status: [],
  type: [],
  typeNotCompleted: false,
  vendor: [],
  vendorNotCompleted: false,
  owner: [],
  from: '',
  to: '',
  waiting: false,
  sort: 'id',
  page: 1,
};

const LISTS = ['status', 'type', 'vendor', 'owner'] as const;
const FLAGS = ['typeNotCompleted', 'vendorNotCompleted', 'waiting'] as const;

export function readListSearch(params: URLSearchParams): ListSearch {
  return purchaseRequestListSearchSchema.parse({
    q: params.get('q') ?? '',
    ...Object.fromEntries(LISTS.map((key) => [key, params.getAll(key)])),
    ...Object.fromEntries(FLAGS.map((key) => [key, params.get(key)])),
    from: params.get('from'),
    to: params.get('to'),
    sort: params.get('sort'),
    page: params.get('page'),
  });
}

/** The address of a search, defaults left out ("?status=approved&waiting=true"). */
export function writeListSearch(search: ListSearch): URLSearchParams {
  const params = new URLSearchParams();
  if (search.q.trim() !== '') {
    params.set('q', search.q);
  }
  for (const key of LISTS) {
    search[key].forEach((value) => params.append(key, value));
  }
  for (const key of FLAGS) {
    if (search[key]) {
      params.set(key, 'true');
    }
  }
  if (search.from) {
    params.set('from', search.from);
  }
  if (search.to) {
    params.set('to', search.to);
  }
  if (search.sort !== DEFAULT_LIST_SEARCH.sort) {
    params.set('sort', search.sort);
  }
  if (search.page !== 1) {
    params.set('page', String(search.page));
  }
  return params;
}

/** A search or filter is in use; the Waiting for GSAP view, the sort and the page are not filters. */
export function listFiltersInUse(search: ListSearch): boolean {
  return (
    search.q.trim() !== '' ||
    LISTS.some((key) => search[key].length > 0) ||
    search.typeNotCompleted ||
    search.vendorNotCompleted ||
    search.from !== '' ||
    search.to !== ''
  );
}

/** Clear filters: the view and the sort stay. */
export function clearedListSearch(search: ListSearch): ListSearch {
  return { ...DEFAULT_LIST_SEARCH, waiting: search.waiting, sort: search.sort };
}

export function toListQuery(search: ListSearch): PurchaseRequestListQuery {
  const query: PurchaseRequestListQuery = {};
  const q = search.q.trim();
  if (q !== '') {
    query.q = q;
  }
  if (search.status.length > 0) {
    query.status = search.status satisfies PurchaseRequestStatus[];
  }
  if (search.type.length > 0) {
    query.type = search.type;
  }
  if (search.typeNotCompleted) {
    query.typeNotCompleted = true;
  }
  if (search.vendor.length > 0) {
    query.vendor = search.vendor;
  }
  if (search.vendorNotCompleted) {
    query.vendorNotCompleted = true;
  }
  if (search.owner.length > 0) {
    query.owner = search.owner;
  }
  if (search.from) {
    query.createdFrom = search.from;
  }
  if (search.to) {
    query.createdTo = search.to;
  }
  if (search.waiting) {
    query.waitingForGsap = true;
  }
  return { ...query, sort: search.sort, page: search.page, pageSize: LIST_PAGE_SIZE };
}
