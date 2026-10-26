import { z } from 'zod';

/** The activity type form's values, as text exactly as typed; the API checks them (FR-013). */
export const activityTypeSchema = z.object({
  name: z.string(),
  defaultPlannedCost: z.string(),
  modelIds: z.array(z.string()),
});

export type ActivityTypeValues = z.infer<typeof activityTypeSchema>;
