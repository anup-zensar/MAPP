import { z } from 'zod';
import type { AttributeValues } from '@/ui/ShellAttributeFields/rules';
import { PLAN_TYPES } from './plan-currency';
import { GOAL_STAGES, SPREAD_METHODS } from './plan-text';

/**
 * The plan form's values as typed (plan "Plan form"). The form checks nothing itself: every
 * problem comes from the API, all at once (FR-010).
 */
export const planFormSchema = z.object({
  name: z.string(),
  /** The year as chosen ("2026"); fixed after the first save. */
  year: z.string(),
  modelId: z.string(),
  teamIds: z.array(z.string()),
  ownerId: z.string(),
  aop: z.boolean(),
  description: z.string(),
  /** Operational (default), Strategic or Cross-market (016 FR-032). */
  planType: z.enum(PLAN_TYPES),
  /** The plan currency (016 FR-011): the budget and every amount of the plan are kept in it. */
  currency: z.string(),
  /** The plan budget in the plan currency as typed ("1,200,000.00"); "" for none. */
  budget: z.string(),
  /** "" for no default. */
  defaultFinanceBudgetId: z.string(),
  defaultSpread: z.enum(SPREAD_METHODS),
  bcpir: z.boolean(),
  goalStage: z.enum(GOAL_STAGES),
  /** As typed: a whole number for INQ and MQL, USD for Revenue; "" for none. */
  goalAmount: z.string(),
  attributes: z.custom<AttributeValues>(),
});

export type PlanFormValues = z.infer<typeof planFormSchema>;

// The name and year stay null until the person changes the API's suggestion (FR-026).
export const copyPlanSchema = z.object({
  name: z.string().nullable(),
  year: z.number().nullable(),
  includeContents: z.boolean(),
});

export type CopyPlanValues = z.infer<typeof copyPlanSchema>;

export const unpublishSchema = z.object({ reason: z.string() });

export type UnpublishValues = z.infer<typeof unpublishSchema>;

export const PLAN_TABS = ['tree', 'details', 'history'] as const;

export type PlanTab = (typeof PLAN_TABS)[number];

const planTabSchema = z.enum(PLAN_TABS).catch('tree');

/** The chosen tab, kept in the address (?tab=details); the tree when none or an unknown one. */
export function planTabOf(search: URLSearchParams): PlanTab {
  return planTabSchema.parse(search.get('tab'));
}

const focusSchema = z.string().min(1).nullable().catch(null);

export function focusOf(search: URLSearchParams): string | null {
  return focusSchema.parse(search.get('focus'));
}

/** A four-digit year from a number or text; null for anything else. */
const yearSchema = z
  .union([
    z.int().min(1000).max(9999),
    z
      .string()
      .regex(/^\d{4}$/)
      .transform(Number),
  ])
  .nullable()
  .catch(null);

/** The known plan types among one value or a list, in the order the filter offers them. */
const typesSchema = z
  .preprocess((value) => (Array.isArray(value) ? value : [value]), z.array(z.unknown()))
  .transform((values) => PLAN_TYPES.filter((type) => values.includes(type)));

/**
 * One filter set of the Plans list from the address or the browser's storage (FR-003; 016
 * FR-033): anything invalid becomes its default. A single `year` (an old address, or what was
 * remembered before 016) reads as a one-year range (D-269).
 */
export const looseFiltersSchema = z
  .object({
    status: z.enum(['all', 'draft', 'published']).catch('all'),
    yearFrom: yearSchema,
    yearTo: yearSchema,
    year: yearSchema,
    type: typesSchema,
    scope: z.enum(['all', 'myTeams']).catch('all'),
    q: z.string().catch(''),
    layout: z.enum(['cards', 'table']).catch('cards'),
  })
  .transform(({ yearFrom, yearTo, year, type, ...rest }) => ({
    ...rest,
    ...rangeOf(yearFrom, yearTo, year),
    types: type,
  }));

/** The year range: from–to in order, one year when only one end is valid. */
function rangeOf(yearFrom: number | null, yearTo: number | null, year: number | null) {
  const from = yearFrom ?? yearTo ?? year;
  const to = yearTo ?? yearFrom ?? year;
  if (from === null || to === null) {
    return { yearFrom: null, yearTo: null };
  }
  return { yearFrom: Math.min(from, to), yearTo: Math.max(from, to) };
}
