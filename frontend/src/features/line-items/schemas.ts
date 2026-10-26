import { z } from 'zod';

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
