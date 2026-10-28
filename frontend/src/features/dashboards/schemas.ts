import { z } from 'zod';
import { DEFAULT_FILTERS } from './dashboard-links';
import type { DashboardFilters } from './dashboard-links';

const yearSchema = z.coerce.number().int().min(2000).max(2100);
const planTypeSchema = z.enum(['operational', 'strategic', 'crossMarket']);
const currencySchema = z.string().regex(/^[A-Z]{3}$/);
const idSchema = z.string().regex(/^[A-Z]+-[0-9]{1,12}$/);

function parsed<T, F>(schema: z.ZodType<T>, value: string | null, fallback: F): T | F {
  const result = schema.safeParse(value ?? undefined);
  return result.success ? result.data : fallback;
}

/** The filters of an address; a value the page cannot use falls back to its default (F3). */
export function readFilters(params: URLSearchParams): DashboardFilters {
  return {
    year: parsed(yearSchema, params.get('year'), DEFAULT_FILTERS.year),
    planType: parsed(planTypeSchema, params.get('planType'), DEFAULT_FILTERS.planType),
    currency: parsed(currencySchema, params.get('currency'), DEFAULT_FILTERS.currency),
    team: parsed(idSchema, params.get('team'), DEFAULT_FILTERS.team),
    plan: parsed(idSchema, params.get('plan'), DEFAULT_FILTERS.plan),
  };
}
