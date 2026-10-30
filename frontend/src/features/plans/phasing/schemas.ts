import { z } from 'zod';
import type { PhasingMethod } from './api';
import { monthLabel, readAmount } from './phasing-math';

const PHASING_METHODS: readonly PhasingMethod[] = ['none', 'evenly', 'custom'];

export const MONTH_PROBLEM = 'Type an amount of 0.00 or more.';

export const phasingSchema = z
  .object({
    method: z.enum(PHASING_METHODS),
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

export function monthProblems(months: string[], keys: string[]): string {
  return keys
    .flatMap((key, index) =>
      readAmount(months[index] ?? '') === null ? [`${monthLabel(key)}: ${MONTH_PROBLEM}`] : [],
    )
    .join(' ');
}

/** The API's problems, "months[2].amount" naming the month of the sent months it is about. */
export function namedProblems(
  fields: { field: string; message: string }[],
  sent: { month: string }[],
): string {
  return fields
    .map(({ field, message }) => {
      const index = /^months\[(\d+)\]/.exec(field)?.[1];
      const month = index === undefined ? undefined : sent[Number(index)]?.month;
      return month ? `${monthLabel(month)}: ${message}` : message;
    })
    .join(' ');
}
