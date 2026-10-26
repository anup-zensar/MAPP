import { z } from 'zod';
import type { components } from '@/api/generated/schema';
import { SOURCE_FIELDS } from './line-item-form-state';
import type { LineItemFormState } from './line-item-form-state';

type AttributeValues = components['schemas']['AttributeValues'];
type FinanceBudgetItemRef = components['schemas']['FinanceBudgetItemRef'];

/** The line-item form checks nothing itself: the API judges every value on save (D-149). */
export const lineItemFormSchema = z.object({
  name: z.string(),
  ownerId: z.string(),
  description: z.string(),
  currency: z.string(),
  plannedCost: z.string(),
  prefilled: z.string().nullable(),
  costNotes: z.array(z.string()),
  spreadMethod: z.enum(['evenly', 'firstMonth', 'lastMonth', 'custom']),
  customMonths: z.array(z.object({ month: z.string(), amount: z.string() })),
  customTyped: z.boolean(),
  attributes: z.custom<AttributeValues>(),
  contractedAmount: z.string(),
  requestSwbs: z.boolean(),
  financeBudgetYear: z.number().nullable(),
  financeBudgetItemId: z.string(),
  clearedItem: z.custom<FinanceBudgetItemRef>().nullable(),
  sources: z.partialRecord(z.enum(SOURCE_FIELDS), z.string()),
}) satisfies z.ZodType<LineItemFormState, LineItemFormState>;

export const LINE_ITEM_FIELDS = lineItemFormSchema.keyof().options;

const gridPeriodSchema = z.enum(['monthly', 'quarterly']);

export type GridPeriod = z.infer<typeof gridPeriodSchema>;

/** The spread grid's period as the browser remembers it; Monthly for anything else. */
export function gridPeriodOf(stored: unknown): GridPeriod {
  const parsed = gridPeriodSchema.safeParse(stored);
  return parsed.success ? parsed.data : 'monthly';
}

const highlightStateSchema = z.object({ highlightLineItemId: z.string() });

export function highlightedLineItemOf(routerState: unknown): string | undefined {
  const parsed = highlightStateSchema.safeParse(routerState);
  return parsed.success ? parsed.data.highlightLineItemId : undefined;
}

export function focusFieldOf(search: URLSearchParams): string | null {
  const parsed = z.string().safeParse(search.get('field'));
  return parsed.success ? parsed.data : null;
}
