import { z } from 'zod';

/** Currency and rate values, as text exactly as typed; the API checks them (FR-028, 016 FR-003). */
export const currencySchema = z.object({
  code: z.string(),
  name: z.string(),
  symbol: z.string(),
  rate: z.string(),
});

export type CurrencyValues = z.infer<typeof currencySchema>;

export const rateSchema = z.object({
  date: z.string(),
  rate: z.string(),
});

export type RateValues = z.infer<typeof rateSchema>;
