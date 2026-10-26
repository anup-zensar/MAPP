import { z } from 'zod';
import type { PhasingMethod } from './api';
import { readAmount } from './phasing-math';

const PHASING_METHODS: readonly PhasingMethod[] = ['none', 'evenly', 'custom'];

/** The problem of one month whose text is not an amount of 0.00 or more. */
export const MONTH_PROBLEM = 'Type an amount of 0.00 or more.';

/** The Budget by month editor (016 FR-020 – FR-023): Custom months must all be amounts. */
export const phasingSchema = z
  .object({
    method: z.enum(PHASING_METHODS),
    /** The text typed for each month of the year. */
    months: z.array(z.string()),
    /** The quarters as typed while typing by quarter; null while they follow the months. */
    quarters: z.array(z.string()).nullable(),
  })
  .refine(
    ({ method, months }) =>
      method !== 'custom' || months.every((text) => readAmount(text) !== null),
    { path: ['months'], message: 'Type each month as an amount of 0.00 or more.' },
  );

export type PhasingValues = z.infer<typeof phasingSchema>;
