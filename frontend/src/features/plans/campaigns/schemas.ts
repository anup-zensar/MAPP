import { z } from 'zod';
import type { AttributeValues } from '@/ui/ShellAttributeFields/rules';
import type { Destination } from './api';
import { CHOOSE_WHERE_TO_MOVE } from './texts';

/** What the form's fields hold. Dates are "YYYY-MM-DD" or ""; the budget is the typed text. */
export const recordFormSchema = z.object({
  name: z.string(),
  ownerId: z.string(),
  startDate: z.string(),
  endDate: z.string(),
  description: z.string(),
  budget: z.string(),
  overspend: z.enum(['warn', 'refuse']),
  attributes: z.custom<AttributeValues>(),
});

export type RecordFormValues = z.infer<typeof recordFormSchema>;

// Both stay null until the person changes the suggested name or the default destination.
export const copySchema = z.object({
  name: z.string().nullable(),
  destinationId: z.string().nullable(),
});

export type CopyValues = z.infer<typeof copySchema>;

export function moveSchema(destinations: Destination[]) {
  return z.object({
    destinationId: z
      .string()
      .refine(
        (id) =>
          destinations.length === 0 ||
          destinations.some((item) => item.id === id && item.available),
        CHOOSE_WHERE_TO_MOVE,
      ),
  });
}

export type MoveValues = z.infer<ReturnType<typeof moveSchema>>;

const highlightSchema = z.object({ highlight: z.string() });

/** The record to highlight, from the location state a create, move or copy leaves (FR-004). */
export function highlightOf(state: unknown): string | undefined {
  const parsed = highlightSchema.safeParse(state);
  return parsed.success ? parsed.data.highlight : undefined;
}
