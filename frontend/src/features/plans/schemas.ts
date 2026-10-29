import { z } from 'zod';
import type { AttributeValues } from '@/ui/ShellAttributeFields/rules';
import { PLAN_TYPES } from './plan-currency';
import { GOAL_STAGES, SPREAD_METHODS } from './plan-text';

export const planFormSchema = z.object({
  name: z.string(),
  year: z.string(),
  modelId: z.string(),
  teamIds: z.array(z.string()),
  ownerId: z.string(),
  aop: z.boolean(),
  description: z.string(),
  planType: z.enum(PLAN_TYPES),
  currency: z.string(),
  budget: z.string(),
  defaultFinanceBudgetId: z.string(),
  defaultSpread: z.enum(SPREAD_METHODS),
  bcpir: z.boolean(),
  goalStage: z.enum(GOAL_STAGES),
  goalAmount: z.string(),
  overspend: z.enum(['warn', 'refuse']),
  attributes: z.custom<AttributeValues>(),
});

export type PlanFormValues = z.infer<typeof planFormSchema>;

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

export function planTabOf(search: URLSearchParams): PlanTab {
  return planTabSchema.parse(search.get('tab'));
}

const focusSchema = z.string().min(1).nullable().catch(null);

export function focusOf(search: URLSearchParams): string | null {
  return focusSchema.parse(search.get('focus'));
}

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

const typesSchema = z
  .preprocess((value) => (Array.isArray(value) ? value : [value]), z.array(z.unknown()))
  .transform((values) => PLAN_TYPES.filter((type) => values.includes(type)));

export const looseFiltersSchema = z
  .object({
    status: z.enum(['all', 'draft', 'published']).catch('all'),
    yearFrom: yearSchema,
    yearTo: yearSchema,
    year: yearSchema,
    type: typesSchema,
    scope: z.enum(['all', 'myTeams']).catch('all'),
    bcpir: z.enum(['include', 'only', 'exclude']).catch('include'),
    q: z.string().catch(''),
    layout: z.enum(['cards', 'table']).catch('cards'),
  })
  .transform(({ yearFrom, yearTo, year, type, ...rest }) => ({
    ...rest,
    ...rangeOf(yearFrom, yearTo, year),
    types: type,
  }));

function rangeOf(yearFrom: number | null, yearTo: number | null, year: number | null) {
  const from = yearFrom ?? yearTo ?? year;
  const to = yearTo ?? yearFrom ?? year;
  if (from === null || to === null) {
    return { yearFrom: null, yearTo: null };
  }
  return { yearFrom: Math.min(from, to), yearTo: Math.max(from, to) };
}
