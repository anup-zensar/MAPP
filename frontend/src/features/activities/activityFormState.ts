/**
 * The activity form's values and rules (006 plan, "Activity form page"): what a new or an edited
 * activity starts with, the client checks that repeat the server's rules for instant feedback
 * (all problems at once, the messages-table texts), the changes that restart approval, and the
 * request bodies. The API stays the judge: it checks everything again on save.
 */
import type { AttributeValues, FieldProblem } from '@/components';
import {
  formatAmount,
  formatMoneyText,
  formatWhole,
  parseMoneyInput,
  parseWholeInput,
} from '@/lib/decimal';
import type {
  Activity,
  ActivityCreate,
  ActivityFormData,
  ActivityUpdate,
  Money,
} from './activitiesApi';
import { EMPTY } from './activityText';

/** The goals as typed: INQ, and MQL and revenue only while Set later stages myself is on. */
export interface GoalsState {
  inq: string;
  setLaterStagesMyself: boolean;
  mql: string;
  revenue: string;
}

/** Everything the form holds, as typed (amounts and numbers as text). */
export interface FormState {
  name: string;
  externalName: string;
  activityTypeId: string;
  /** "" for nobody. */
  ownerId: string;
  /** "YYYY-MM-DD" or "". */
  startDate: string;
  endDate: string;
  ongoing: boolean;
  /** The planned cost in USD as typed ("120,000.00"), "" for none. */
  plannedCost: string;
  approvalWorkflowId: string;
  description: string;
  attributes: AttributeValues;
  goals: GoalsState;
}

/** The longest names the API takes. */
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

export function emptyGoals(): GoalsState {
  return { inq: '', setLaterStagesMyself: false, mql: '', revenue: '' };
}

/** A new activity: the defaults of the form (owner and dates from the initiative, the workflow). */
export function fromDefaults(form: ActivityFormData): FormState {
  const defaults = form.defaults;
  return {
    name: '',
    externalName: '',
    activityTypeId: '',
    ownerId: defaults?.owner?.id ?? '',
    startDate: defaults?.startDate ?? '',
    endDate: defaults?.endDate ?? '',
    ongoing: false,
    plannedCost: '',
    approvalWorkflowId: defaults?.approvalWorkflowId ?? '',
    description: '',
    attributes: {},
    goals: emptyGoals(),
  };
}

/** An activity as the edit form starts with it. */
export function fromActivity(activity: Activity): FormState {
  const { goals } = activity;
  const typed = goals.setLaterStagesMyself;
  return {
    name: activity.name,
    externalName: activity.externalName,
    activityTypeId: activity.activityType.id,
    ownerId: activity.owner?.id ?? '',
    startDate: activity.startDate ?? '',
    endDate: activity.endDate ?? '',
    ongoing: activity.ongoing,
    plannedCost: activity.plannedCost ? formatAmount(activity.plannedCost.amount) : '',
    approvalWorkflowId: activity.approvalWorkflow?.id ?? '',
    description: activity.description,
    attributes: activity.attributes,
    goals: {
      inq: goals.inq === null ? '' : formatWhole(goals.inq),
      setLaterStagesMyself: typed,
      mql: typed && goals.mql !== null ? formatWhole(goals.mql) : '',
      revenue: typed && goals.revenue ? formatAmount(goals.revenue.amount) : '',
    },
  };
}

/**
 * The input id of a field the API names: "field-ownerId", "field-goals.inq"; Shell attributes keep
 * 002's "attributes.<attributeId>". Links of the Ready to submit panel use the API's names.
 */
export function fieldId(field: string): string {
  return field.startsWith('attributes.') ? field : `field-${field}`;
}

/** Whether something was typed since the form opened. */
export function isDirty(state: FormState, initial: FormState): boolean {
  return JSON.stringify(state) !== JSON.stringify(initial);
}

/** Two activity names are the same name when they differ only in letter case and outer spaces. */
export function sameName(a: string, b: string): boolean {
  return a.trim().toLowerCase() === b.trim().toLowerCase();
}

/** What validate needs besides the values. */
export interface ValidationContext {
  form: ActivityFormData;
  mode: 'new' | 'edit';
  /** The values the edit form opened with: dates are checked against the initiative only when changed. */
  original?: FormState;
  /** Writes a date in the person's format ("01 Mar 2026"). */
  date: (date: string) => string;
}

/**
 * Every problem of the typed values at once, in the form's order, with the API's field names
 * (FR-006, FR-009 – FR-011, FR-018). Nothing is sent while there is one.
 */
export function validate(state: FormState, context: ValidationContext): FieldProblem[] {
  const { form } = context;
  const problems: FieldProblem[] = [];
  const add = (field: string, message: string) => problems.push({ field, message });

  const name = state.name.trim();
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
  if (length(state.externalName.trim()) > NAME_LIMIT) {
    add('externalName', NAME_TOO_LONG);
  }
  if (state.activityTypeId === '') {
    add('activityTypeId', CHOOSE_TYPE);
  }

  for (const problem of dateProblems(state, context)) {
    problems.push(problem);
  }

  const cost = moneyProblem(state.plannedCost, COST_NEGATIVE);
  if (cost) {
    add('plannedCost', cost);
  }

  const { goals } = state;
  if (!isWholeOrEmpty(goals.inq)) {
    add('goals.inq', WHOLE_NUMBER);
  }
  if (goals.setLaterStagesMyself) {
    if (!isWholeOrEmpty(goals.mql)) {
      add('goals.mql', WHOLE_NUMBER);
    }
    const revenue = moneyProblem(goals.revenue, REVENUE_NEGATIVE);
    if (revenue) {
      add('goals.revenue', revenue);
    }
  }

  if (length(state.description) > DESCRIPTION_LIMIT) {
    add('description', DESCRIPTION_TOO_LONG);
  }
  return problems;
}

/** The end date as it is saved: none while ongoing. */
function endOf(state: FormState): string {
  return state.ongoing ? '' : state.endDate;
}

/** The date rules of FR-009 and FR-010 (server: checkDates). */
function dateProblems(state: FormState, context: ValidationContext): FieldProblem[] {
  const { form, mode, original, date } = context;
  const problems: FieldProblem[] = [];
  const start = state.startDate;
  const end = endOf(state);

  if (start && end && end < start) {
    problems.push({
      field: 'endDate',
      message: `End date must be on or after the start date (${date(start)}).`,
    });
  }

  // Saved dates are checked against the initiative only when they change: the initiative's own
  // dates may have moved since (edge case), and other edits must still save.
  const changed =
    mode === 'new' ||
    !original ||
    start !== original.startDate ||
    end !== endOf(original) ||
    state.ongoing !== original.ongoing;
  if (!changed) {
    return problems;
  }

  const { initiative, latestEndDate } = form.context;
  if (initiative.startDate && initiative.endDate) {
    const message = `Dates must be inside the initiative's dates (${date(initiative.startDate)} – ${date(initiative.endDate)}).`;
    const outside = (day: string) => day < initiative.startDate! || day > initiative.endDate!;
    if (start && outside(start)) {
      problems.push({ field: 'startDate', message });
    }
    if (end && outside(end)) {
      problems.push({ field: 'endDate', message });
    }
  }
  if (end && end > latestEndDate) {
    problems.push({
      field: 'endDate',
      message: `The end date can't be after ${date(latestEndDate)} — 10 years after the plan's year.`,
    });
  }
  return problems;
}

/** The problem of a typed amount that must be 0 or more, or undefined when it is fine or empty. */
function moneyProblem(text: string, negative: string): string | undefined {
  if (text.trim() === '') {
    return undefined;
  }
  const parsed = parseMoneyInput(text);
  if (!parsed) {
    return ENTER_AMOUNT;
  }
  return parsed.amount.startsWith('-') ? negative : undefined;
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

/** A typed amount in USD as the API takes it; null when empty. Call after validate. */
export function usdOf(text: string): Money | null {
  const parsed = text.trim() === '' ? null : parseMoneyInput(text);
  return parsed ? { amount: parsed.amount, currency: 'USD' } : null;
}

function wholeOf(text: string): number | null {
  const value = text.trim() === '' ? null : parseWholeInput(text);
  return value === null ? null : Number(value);
}

/** The new activity's request body. */
export function toCreateBody(state: FormState): ActivityCreate {
  return {
    name: state.name.trim(),
    externalName: state.externalName.trim(),
    activityTypeId: state.activityTypeId,
    ownerId: state.ownerId || null,
    startDate: state.startDate || null,
    endDate: endOf(state) || null,
    ongoing: state.ongoing,
    plannedCost: usdOf(state.plannedCost),
    approvalWorkflowId: state.approvalWorkflowId || null,
    description: state.description,
    attributes: state.attributes,
    goals: goalsBody(state.goals),
  };
}

/** The edited activity's request body: every field, the version the form opened with. */
export function toUpdateBody(
  state: FormState,
  version: number,
  restartApproval: boolean,
): ActivityUpdate {
  return { ...(toCreateBody(state) as Required<ActivityCreate>), version, restartApproval };
}

function goalsBody(goals: GoalsState): ActivityCreate['goals'] {
  const typed = goals.setLaterStagesMyself;
  return {
    inq: wholeOf(goals.inq),
    setLaterStagesMyself: typed,
    mql: typed ? wholeOf(goals.mql) : null,
    revenue: typed ? usdOf(goals.revenue) : null,
  };
}

/** One change that sends a submitted or approved activity back for approval. */
export interface ReapprovalChange {
  label: string;
  old: string;
  new: string;
}

/**
 * The changes of planned cost, start date, end date and ongoing between the opened and the typed
 * values (FR-027): "Planned cost: 120,000.00 USD → 150,000.00 USD".
 */
export function reapprovalChanges(
  original: FormState,
  state: FormState,
  date: (date: string) => string,
): ReapprovalChange[] {
  const changes: ReapprovalChange[] = [];
  const oldCost = usdOf(original.plannedCost);
  const newCost = usdOf(state.plannedCost);
  if (oldCost?.amount !== newCost?.amount) {
    changes.push({ label: 'Planned cost', old: moneyText(oldCost), new: moneyText(newCost) });
  }
  const dateText = (day: string) => (day ? date(day) : EMPTY);
  if (original.startDate !== state.startDate) {
    changes.push({
      label: 'Start date',
      old: dateText(original.startDate),
      new: dateText(state.startDate),
    });
  }
  if (endOf(original) !== endOf(state)) {
    changes.push({
      label: 'End date',
      old: dateText(endOf(original)),
      new: dateText(endOf(state)),
    });
  }
  if (original.ongoing !== state.ongoing) {
    changes.push({
      label: 'Ongoing',
      old: original.ongoing ? 'Yes' : 'No',
      new: state.ongoing ? 'Yes' : 'No',
    });
  }
  return changes;
}

function moneyText(money: Money | null): string {
  return money ? formatMoneyText(money) : EMPTY;
}
