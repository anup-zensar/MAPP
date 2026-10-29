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

export interface StoredActivity {
  id: string;
  status: Activity['status'];
  ongoing: boolean;
  attributes: AttributeValues;
  plannedCost?: Activity['plannedCost'];
}

export interface GoalsState {
  inq: string;
  setLaterStagesMyself: boolean;
  mql: string;
  revenue: string;
}

export interface FormState {
  name: string;
  externalName: string;
  activityTypeId: string;
  ownerId: string;
  startDate: string;
  endDate: string;
  ongoing: boolean;
  plannedCost: string;
  approvalWorkflowId: string;
  description: string;
  attributes: AttributeValues;
  goals: GoalsState;
}

export function emptyGoals(): GoalsState {
  return { inq: '', setLaterStagesMyself: false, mql: '', revenue: '' };
}

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

export function fieldId(field: string): string {
  return field.startsWith('attributes.') ? field : `field-${field}`;
}

export function sameName(a: string, b: string): boolean {
  return a.trim().toLowerCase() === b.trim().toLowerCase();
}

export function endOf(state: FormState): string {
  return state.ongoing ? '' : state.endDate;
}

export function usdOf(text: string): Money | null {
  return amountOf(text, 'USD');
}

export function amountOf(text: string, currency: string): Money | null {
  const parsed = text.trim() === '' ? null : parseMoneyInput(text);
  return parsed ? { amount: parsed.amount, currency } : null;
}

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

export interface ReapprovalChange {
  label: string;
  old: string;
  new: string;
}

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
