import { z } from 'zod';
import type { components } from '@/api/generated/schema';
import { parseMoneyInput, parseWholeInput, toCents } from '@/format/decimal';
import type { ActivityDestination, ActivityFormData } from './api';
import { endOf, sameName } from './activity-form-state';
import type { FormState } from './activity-form-state';
import { CHOOSE_ANOTHER_INITIATIVE } from './activity-text';
import { NOTHING_CHOSEN } from './split-text';

type AttributeValues = components['schemas']['AttributeValues'];

const NAME_LIMIT = 120;
const DESCRIPTION_LIMIT = 2000;

const ENTER_NAME = 'Enter a name.';
const NAME_TOO_LONG = 'Use 120 characters or fewer.';
const DESCRIPTION_TOO_LONG = 'Use 2,000 characters or fewer.';
const CHOOSE_TYPE = 'Choose an activity type.';
const ENTER_AMOUNT = 'Enter an amount, such as 12,500.00.';
const COST_NEGATIVE = "Planned cost can't be negative.";
const REVENUE_NEGATIVE = "Revenue can't be negative.";
const WHOLE_NUMBER = 'Enter a whole number of 0 or more.';

export interface ActivityCheckContext {
  form: ActivityFormData;
  mode: 'new' | 'edit';
  /** The values the edit form opened with: dates are checked against the initiative only when changed. */
  original?: FormState;
  date: (date: string) => string;
}

type AddProblem = (field: string, message: string) => void;

const activityValuesSchema = z.object({
  name: z.string(),
  externalName: z.string(),
  activityTypeId: z.string(),
  ownerId: z.string(),
  startDate: z.string(),
  endDate: z.string(),
  ongoing: z.boolean(),
  plannedCost: z.string(),
  approvalWorkflowId: z.string(),
  description: z.string(),
  attributes: z.custom<AttributeValues>(),
  goals: z.object({
    inq: z.string(),
    setLaterStagesMyself: z.boolean(),
    mql: z.string(),
    revenue: z.string(),
  }),
}) satisfies z.ZodType<FormState, FormState>;

/**
 * The activity form's checks (FR-006, FR-009 – FR-011, FR-018), in the form's order: they repeat
 * the server's rules for instant feedback, and the API checks everything again on save.
 */
export function activityFormSchema(context: ActivityCheckContext) {
  return activityValuesSchema.superRefine((values, issues) => {
    const add: AddProblem = (field, message) =>
      issues.addIssue({ code: 'custom', path: field.split('.'), message });
    checkNames(values, context.form, add);
    if (values.activityTypeId === '') {
      add('activityTypeId', CHOOSE_TYPE);
    }
    checkDates(values, context, add);
    checkMoney('plannedCost', values.plannedCost, COST_NEGATIVE, add);
    if (!isWholeOrEmpty(values.goals.inq)) {
      add('goals.inq', WHOLE_NUMBER);
    }
    if (values.goals.setLaterStagesMyself) {
      if (!isWholeOrEmpty(values.goals.mql)) {
        add('goals.mql', WHOLE_NUMBER);
      }
      checkMoney('goals.revenue', values.goals.revenue, REVENUE_NEGATIVE, add);
    }
    if (length(values.description) > DESCRIPTION_LIMIT) {
      add('description', DESCRIPTION_TOO_LONG);
    }
  });
}

function checkNames(values: FormState, form: ActivityFormData, add: AddProblem) {
  const name = values.name.trim();
  if (name === '') {
    add('name', ENTER_NAME);
  } else if (length(name) > NAME_LIMIT) {
    add('name', NAME_TOO_LONG);
  } else {
    const taken = form.existingNames.find((existing) => sameName(existing, name));
    if (taken !== undefined) {
      add('name', `'${form.context.initiative.name}' already has an activity called '${taken}'.`);
    }
  }
  if (length(values.externalName.trim()) > NAME_LIMIT) {
    add('externalName', NAME_TOO_LONG);
  }
}

function checkDates(values: FormState, context: ActivityCheckContext, add: AddProblem) {
  const { form, mode, original, date } = context;
  const start = values.startDate;
  const end = endOf(values);
  if (start && end && end < start) {
    add('endDate', `End date must be on or after the start date (${date(start)}).`);
  }
  // Saved dates are checked against the initiative only when they change: the initiative's own
  // dates may have moved since (edge case), and other edits must still save.
  const changed =
    mode === 'new' ||
    !original ||
    start !== original.startDate ||
    end !== endOf(original) ||
    values.ongoing !== original.ongoing;
  if (!changed) {
    return;
  }
  const { initiative, latestEndDate } = form.context;
  const { startDate: first, endDate: last } = initiative;
  if (first && last) {
    const message = `Dates must be inside the initiative's dates (${date(first)} – ${date(last)}).`;
    const outside = (day: string) => day < first || day > last;
    if (start && outside(start)) {
      add('startDate', message);
    }
    if (end && outside(end)) {
      add('endDate', message);
    }
  }
  if (end && end > latestEndDate) {
    add(
      'endDate',
      `The end date can't be after ${date(latestEndDate)} — 10 years after the plan's year.`,
    );
  }
}

function checkMoney(field: string, text: string, negative: string, add: AddProblem) {
  if (text.trim() === '') {
    return;
  }
  const parsed = parseMoneyInput(text);
  if (!parsed) {
    add(field, ENTER_AMOUNT);
  } else if (parsed.amount.startsWith('-')) {
    add(field, negative);
  }
}

function isWholeOrEmpty(text: string): boolean {
  if (text.trim() === '') {
    return true;
  }
  const value = parseWholeInput(text);
  return value !== null && value >= 0n;
}

/** Characters as people count them (an emoji is one). */
function length(text: string): number {
  return [...text].length;
}

/** The copy's initiative: the chosen one, else the activity's own when a copy can go there. */
export function copyTargetOf(chosen: string, items: ActivityDestination[]): string {
  return (
    chosen || (items.find((item) => item.current && item.copyProblem === null)?.initiative.id ?? '')
  );
}

export function copySchema(items: ActivityDestination[]) {
  return z
    .object({ initiativeId: z.string() })
    .refine(
      ({ initiativeId }) =>
        items.some(
          (item) =>
            item.initiative.id === copyTargetOf(initiativeId, items) && item.copyProblem === null,
        ),
      { message: CHOOSE_ANOTHER_INITIATIVE, path: ['initiativeId'] },
    );
}

export function moveSchema(items: ActivityDestination[]) {
  return z
    .object({ initiativeId: z.string(), search: z.string() })
    .refine(
      ({ initiativeId }) =>
        !items.some((item) => !item.current) ||
        items.some((item) => item.initiative.id === initiativeId && item.moveProblem === null),
      { message: CHOOSE_ANOTHER_INITIATIVE, path: ['initiativeId'] },
    );
}

export function isNothingChosen(lineItemIds: string[], amount: string): boolean {
  const parsed = amount.trim() === '' ? null : parseMoneyInput(amount);
  return lineItemIds.length === 0 && (!parsed || toCents(parsed.amount) <= 0n);
}

/** Split activity: null keeps the suggested name, the own initiative and the ticked items' total. */
export const splitSchema = z
  .object({
    name: z.string().nullable(),
    initiativeId: z.string().nullable(),
    lineItemIds: z.array(z.string()),
    plannedCost: z.string().nullable(),
  })
  .refine((values) => !isNothingChosen(values.lineItemIds, values.plannedCost ?? '0.00'), {
    message: NOTHING_CHOSEN,
    path: ['lineItemIds'],
  });

export type SplitValues = z.infer<typeof splitSchema>;

const activityTabSchema = z.enum(['overview', 'details', 'approval', 'history']);

export type ActivityTab = z.infer<typeof activityTabSchema>;

const partSchema = z.enum([
  '#approval',
  '#history',
  '#line-items',
  '#financials',
  '#ready',
  '#completeness',
]);

/** The tab of each #part other pages link to (Home, notifications, chips, View history). */
const TAB_OF_PART: Record<z.infer<typeof partSchema>, ActivityTab> = {
  '#approval': 'approval',
  '#history': 'history',
  '#line-items': 'overview',
  '#financials': 'overview',
  '#ready': 'overview',
  '#completeness': 'overview',
};

/** The tab an address shows: its #part's tab first, then ?tab=, else Overview. */
export function tabOf(search: string, hash: string): ActivityTab {
  const part = partSchema.safeParse(hash);
  if (part.success) {
    return TAB_OF_PART[part.data];
  }
  const chosen = activityTabSchema.safeParse(new URLSearchParams(search).get('tab'));
  return chosen.success ? chosen.data : 'overview';
}

export function tabSearch(tab: ActivityTab): string {
  return tab === 'overview' ? '' : `?tab=${tab}`;
}

export function focusFieldOf(search: URLSearchParams): string | null {
  const parsed = z.string().safeParse(search.get('field'));
  return parsed.success ? parsed.data : null;
}
