import { z } from 'zod';

export const financeBudgetItemSchema = z.object({
  year: z.string(),
  budgetName: z.string(),
  name: z.string(),
  code: z.string(),
});

export type FinanceBudgetItemValues = z.infer<typeof financeBudgetItemSchema>;

export const renameBudgetSchema = z.object({ name: z.string() });

export type RenameBudgetValues = z.infer<typeof renameBudgetSchema>;

const yearParamSchema = z
  .string()
  .transform((text) => Number.parseInt(text, 10))
  .pipe(z.number().int());

/** The budget year of the address (`?year=`); undefined for none, so the API picks this year. */
export function budgetYearOf(params: URLSearchParams): number | undefined {
  const parsed = yearParamSchema.safeParse(params.get('year'));
  return parsed.success ? parsed.data : undefined;
}
