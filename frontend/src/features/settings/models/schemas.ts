import { z } from 'zod';

/** The model form's values, as text exactly as typed; the API checks them (FR-009, FR-010). */
export const modelSchema = z.object({
  name: z.string(),
  inqToMqlRate: z.string(),
  mqlToDealRate: z.string(),
  averageDealSize: z.string(),
  activityTypeIds: z.array(z.string()),
});

export type ModelValues = z.infer<typeof modelSchema>;
