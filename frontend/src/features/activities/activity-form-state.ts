/**
 * The activity form's values (006 plan, "Activity form page"): what a new or an edited activity
 * starts with, the changes that restart approval and the request bodies; schemas.ts checks them.
 */
import type { AttributeValues } from '@/ui/ShellAttributeFields/rules';
import {
  formatAmount,
  formatMoneyText,
  formatWhole,
  parseMoneyInput,
  parseWholeInput,
} from '@/format/decimal';
import type { Activity, ActivityCreate, ActivityFormData, ActivityUpdate, Money } from './api';
import { EMPTY } from './activity-text';

/**
 * The edited activity as it is stored: its status and saved ongoing decide the locks; its Shell
 * attribute values show which dependent values saving would clear.
 */
export interface StoredActivity {
  id: string;
  status: Activity['status'];
  ongoing: boolean;
  attributes: AttributeValues;
}

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
  /** The planned cost in the plan currency as typed ("120,000.00"), "" for none (016 FR-011). */
  plannedCost: string;
  approvalWorkflowId: string;
  description: string;
  attributes: AttributeValues;
  goals: GoalsState;
}

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

/** Two activity names are the same name when they differ only in letter case and outer spaces. */
export function sameName(a: string, b: string): boolean {
  return a.trim().toLowerCase() === b.trim().toLowerCase();
}

/** The end date as it is saved: none while ongoing. */
export function endOf(state: FormState): string {
  return state.ongoing ? '' : state.endDate;
}

/** A typed amount in USD as the API takes it; null when empty. Call after the checks. */
export function usdOf(text: string): Money | null {
  return amountOf(text, 'USD');
}

/** A typed amount in a currency as the API takes it; null when empty. Call after the checks. */
export function amountOf(text: string, currency: string): Money | null {
  const parsed = text.trim() === '' ? null : parseMoneyInput(text);
  return parsed ? { amount: parsed.amount, currency } : null;
}

/**
 * The plan currency the planned cost is typed and kept in (016 FR-011): the stored planned cost's,
 * else the activity types' defaults' (the API converts them into it, FR-016), else USD. The form's
 * context does not name it (contract gap, reported).
 */
export function planCurrencyOf(
  form: { activityTypes: { defaultPlannedCost: Money | null }[] },
  stored?: { plannedCost: Money | null },
): string {
  return (
    stored?.plannedCost?.currency ??
    form.activityTypes.find((type) => type.defaultPlannedCost)?.defaultPlannedCost?.currency ??
    'USD'
  );
}

function wholeOf(text: string): number | null {
  const value = text.trim() === '' ? null : parseWholeInput(text);
  return value === null ? null : Number(value);
}

/** The new activity's request body; the planned cost in the plan currency (016 FR-011). */
export function toCreateBody(state: FormState, planCurrency = 'USD'): Required<ActivityCreate> {
  return {
    name: state.name.trim(),
    externalName: state.externalName.trim(),
    activityTypeId: state.activityTypeId,
    ownerId: state.ownerId || null,
    startDate: state.startDate || null,
    endDate: endOf(state) || null,
    ongoing: state.ongoing,
    plannedCost: amountOf(state.plannedCost, planCurrency),
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
  planCurrency = 'USD',
): ActivityUpdate {
  return {
    ...toCreateBody(state, planCurrency),
    version,
    restartApproval,
  };
}

function goalsBody(goals: GoalsState): Required<ActivityCreate>['goals'] {
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
  planCurrency = 'USD',
): ReapprovalChange[] {
  const changes: ReapprovalChange[] = [];
  const oldCost = amountOf(original.plannedCost, planCurrency);
  const newCost = amountOf(state.plannedCost, planCurrency);
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
