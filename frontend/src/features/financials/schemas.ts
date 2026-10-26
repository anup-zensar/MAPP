import { z } from 'zod';
import { DEFAULT_FILTERS, PAGE_SIZES } from './financials-links';
import type { FinancialFilters, PageSize } from './financials-links';

/** The largest file the API takes (FR-004): 700 KB. */
const MAX_FILE_BYTES = 700 * 1024;

const typeTabSchema = z.enum(['purchaseOrder', 'invoice', 'payment', 'accrual']);
const sortColumnSchema = z.enum([
  'postingDate',
  'documentNumber',
  'type',
  'vendor',
  'amount',
  'linkedTo',
]);
const directionSchema = z.enum(['asc', 'desc']);
const yearSchema = z.coerce.number().int().min(2000).max(2100);
const pageSchema = z.coerce.number().int().min(1);
const pageSizeSchema = z.coerce.number().pipe(z.literal([...PAGE_SIZES]));
const planIdSchema = z.string().regex(/^P-[0-9]+$/);
const recordIdSchema = z.string().regex(/^[PCIAL]-[0-9]+$/);

/**
 * The filters an address holds. Anything the page cannot use (an unknown tab, a bad year, page 0)
 * falls back to its default, so an old or edited link still opens a working page.
 */
export function readFilters(params: URLSearchParams): FinancialFilters {
  const [sortText, directionText] = (params.get('sort') ?? '').split(':');
  const sort = sortColumnSchema.safeParse(sortText);
  const typeTab = typeTabSchema.safeParse(params.get('type'));
  return {
    tab: params.get('link') === 'notLinked' ? 'notLinked' : typeTab.success ? typeTab.data : 'all',
    q: (params.get('q') ?? '').slice(0, 200),
    year: parsed(yearSchema, params.get('year'), null),
    planId: parsed(planIdSchema, params.get('planId'), ''),
    recordId: parsed(recordIdSchema, params.get('recordId'), ''),
    sort: sort.success ? sort.data : DEFAULT_FILTERS.sort,
    direction: parsed(
      directionSchema,
      directionText,
      sort.success ? 'asc' : DEFAULT_FILTERS.direction,
    ),
    page: parsed(pageSchema, params.get('page'), 1),
    pageSize: parsed(pageSizeSchema, params.get('pageSize'), DEFAULT_FILTERS.pageSize),
  };
}

export function pageSizeOf(text: string): PageSize | null {
  return parsed(pageSizeSchema, text, null);
}

export const importFileSchema = z.object({
  file: z
    .instanceof(File)
    .nullable()
    .refine((file) => file !== null, 'Choose a .csv file — GSAP extracts are imported as CSV.')
    .refine(
      (file) => file === null || file.size <= MAX_FILE_BYTES,
      'The file is larger than 700 KB — split it into smaller files.',
    ),
});

export const linkSchema = z.object({
  lineItemId: z
    .string()
    .refine((text) => isLineItemId(lineItemIdOf(text)), 'Line item ID must look like L-1003.'),
});

export type LinkValues = z.infer<typeof linkSchema>;

export function lineItemIdOf(text: string): string {
  return text.trim().toUpperCase();
}

export function isLineItemId(id: string): boolean {
  return /^L-[0-9]+$/.test(id);
}

function parsed<T, F>(schema: z.ZodType<T>, value: unknown, fallback: F): T | F {
  const result = schema.safeParse(value);
  return result.success ? result.data : fallback;
}
