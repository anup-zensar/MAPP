/**
 * The line-item form's state and its rules (007 plan, "Line-item form"): the defaults with their
 * sources (FR-008), the currency change that keeps the pre-filled balance (FR-012), the custom
 * grid's start (D-152), the finance budget item a year change clears (FR-015, D-151), the request
 * bodies, and when a save needs "This activity will need approval again" first (FR-031, FR-032).
 */
import { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { parseMoneyInput, toCents } from '@/format/decimal';
import type {
  LineItem,
  LineItemActivity,
  LineItemCreate,
  LineItemFormOptions,
} from './lineItemsApi';
import {
  amountText,
  centsInput,
  convert,
  customGridMonths,
  parseAmount,
  splitEvenly,
  spreadMonths,
} from './lineItemMath';
import type { ActivityDates, SpreadMethod } from './lineItemMath';
import {
  changedToText,
  costChangeReason,
  newLineItemReason,
  NO_COST_TO_PREFILL,
  NOTHING_LEFT,
  preselectedText,
  ROUNDING_HINT,
  SOURCE,
} from './lineItemText';

type AttributeValues = components['schemas']['AttributeValues'];
type FinanceBudgetItemRef = components['schemas']['FinanceBudgetItemRef'];
type LineItemCurrency = components['schemas']['LineItemCurrency'];

/** One month of the custom grid, as typed. */
export interface CustomMonth {
  /** "2026-02". */
  month: string;
  /** What is typed; "" for nothing. */
  amount: string;
}

/** The fields that show where their pre-filled value came from. */
export type SourceField =
  | 'name'
  | 'ownerId'
  | 'currency'
  | 'plannedCost'
  | 'spread'
  | 'financeBudgetYear'
  | 'financeBudgetItemId'
  | 'requestSwbs';

export interface LineItemFormState {
  name: string;
  /** A person id; "" for nobody. */
  ownerId: string;
  description: string;
  currency: string;
  /** The planned cost as typed ("20,000.00"). */
  plannedCost: string;
  /**
   * The pre-filled unassigned balance in the plan currency ("20000.00"; USD before 016) while the
   * planned cost still holds it (FR-012, 016 FR-015); null once the person types an amount.
   */
  prefilled: string | null;
  /** "Changed to 18,400.00 EUR = 20,000.00 USD", and the rounding hint when needed (FR-012). */
  costNotes: string[];
  spreadMethod: SpreadMethod;
  /** The custom grid; kept when another method is chosen, so choosing Custom again restores it. */
  customMonths: CustomMonth[];
  /** The custom amounts were typed (or saved): leaving Custom replaces them (FR-018, D-151). */
  customTyped: boolean;
  attributes: AttributeValues;
  contractedAmount: string;
  requestSwbs: boolean;
  financeBudgetYear: number | null;
  /** "" for none. */
  financeBudgetItemId: string;
  /** The item a year change will clear on save, with the note at the field (FR-015, D-151). */
  clearedItem: FinanceBudgetItemRef | null;
  /** Where pre-filled values came from, shown until the person changes them (FR-008). */
  sources: Partial<Record<SourceField, string>>;
}

/** A new line item's form, filled in with the defaults and their sources (FR-008). */
export function newFormState(options: LineItemFormOptions): LineItemFormState {
  const { defaults } = options;
  const preselected = defaults.financeBudgetItemPreselected
    ? options.financeBudgetItems.find((item) => item.id === defaults.financeBudgetItemId)
    : undefined;
  const sources: LineItemFormState['sources'] = {
    name: SOURCE.name,
    currency: SOURCE.currency,
    plannedCost:
      defaults.plannedCostSource === 'unassignedBalance'
        ? SOURCE.plannedCost
        : defaults.plannedCostSource === 'nothingLeftToAssign'
          ? NOTHING_LEFT
          : NO_COST_TO_PREFILL,
    spread: SOURCE.spread,
    financeBudgetYear: SOURCE.year,
    requestSwbs: SOURCE.requestSwbs,
  };
  if (defaults.owner) {
    sources.ownerId = SOURCE.owner;
  }
  if (preselected) {
    sources.financeBudgetItemId = preselectedText(preselected.budgetName);
  }
  return {
    name: defaults.name,
    ownerId: defaults.owner?.id ?? '',
    description: '',
    currency: defaults.currency,
    plannedCost: defaults.plannedCost ? centsInput(toCents(defaults.plannedCost.amount)) : '',
    prefilled: defaults.plannedCost?.amount ?? null,
    costNotes: [],
    spreadMethod: defaults.spreadMethod,
    customMonths: [],
    customTyped: false,
    attributes: {},
    contractedAmount: '',
    requestSwbs: defaults.requestSwbs,
    financeBudgetYear: defaults.financeBudgetYear,
    financeBudgetItemId: defaults.financeBudgetItemId ?? '',
    clearedItem: null,
    sources,
  };
}

/** A stored line item's form (FR-027). */
export function editFormState(lineItem: LineItem): LineItemFormState {
  const custom = lineItem.spread.method === 'custom';
  return {
    name: lineItem.name,
    ownerId: lineItem.owner?.id ?? '',
    description: lineItem.description,
    currency: lineItem.currency.code,
    plannedCost: centsInput(toCents(lineItem.plannedCost.amount)),
    prefilled: null,
    costNotes: [],
    spreadMethod: lineItem.spread.method,
    customMonths: custom
      ? lineItem.spread.months.map(({ month, amount }) => ({
          month,
          amount: centsInput(toCents(amount.amount)),
        }))
      : [],
    customTyped: custom,
    attributes: lineItem.attributes,
    contractedAmount: lineItem.contractedAmount
      ? centsInput(toCents(lineItem.contractedAmount.amount))
      : '',
    requestSwbs: lineItem.requestSwbs,
    financeBudgetYear: lineItem.financeBudgetYear,
    financeBudgetItemId: lineItem.financeBudgetItem?.id ?? '',
    clearedItem: null,
    sources: {},
  };
}

/** The state with one field changed by the person: its source note goes. */
export function withField<K extends keyof LineItemFormState>(
  state: LineItemFormState,
  field: K,
  value: LineItemFormState[K],
): LineItemFormState {
  const sources = { ...state.sources };
  delete sources[field as SourceField];
  return { ...state, [field]: value, sources };
}

/** A typed planned cost: no longer the pre-filled balance (FR-012). */
export function changePlannedCost(state: LineItemFormState, text: string): LineItemFormState {
  return { ...withField(state, 'plannedCost', text), prefilled: null, costNotes: [] };
}

/**
 * Another currency. While the planned cost still holds the pre-filled balance, it is converted so
 * that its value in the plan currency stays the balance, and the form says so (FR-012; 016 FR-015:
 * "the pre-filled balance keeps its plan-currency value"). The plan currency is USD before 016.
 */
export function changeCurrency(
  state: LineItemFormState,
  code: string,
  currencies: LineItemCurrency[],
  planCode = 'USD',
): LineItemFormState {
  const next = withField(state, 'currency', code);
  const currency = currencies.find((candidate) => candidate.code === code);
  const planRate = rateOf(currencies, planCode);
  if (state.prefilled === null || !currency || planRate === null) {
    return next;
  }
  const balance = toCents(state.prefilled);
  const local = convert(balance, planRate, currency.ratePerUsd);
  const back = convert(local, currency.ratePerUsd, planRate);
  const notes =
    code === planCode ? [] : [changedToText(amountText(local, code), amountText(back, planCode))];
  if (code !== planCode && back !== balance) {
    notes.push(ROUNDING_HINT);
  }
  return { ...next, plannedCost: centsInput(local), costNotes: notes };
}

/** A currency's rate per 1 USD from the form's currencies; USD is 1; null when it is not there. */
export function rateOf(currencies: LineItemCurrency[], code: string): string | null {
  if (code === 'USD') {
    return '1';
  }
  return currencies.find((candidate) => candidate.code === code)?.ratePerUsd ?? null;
}

/**
 * Another spread method. Custom starts from the current spread over the grid's months (an even
 * spread when nothing is spread yet), unless the grid already has amounts (D-152).
 */
export function changeSpreadMethod(
  state: LineItemFormState,
  method: SpreadMethod,
  dates: ActivityDates,
  planYear: number,
): LineItemFormState {
  const next = withField(state, 'spreadMethod', method);
  if (method !== 'custom' || state.customMonths.length > 0) {
    return next;
  }
  const months = customGridMonths(dates, planYear);
  const cents = parseAmount(state.plannedCost) ?? 0n;
  const current =
    state.spreadMethod === 'custom' ? null : spreadMonths(state.spreadMethod, cents, dates);
  const amounts =
    current && current.state === 'spread' ? current.months : splitEvenly(cents, months);
  return {
    ...next,
    customMonths: months.map((month) => {
      const amount = amounts.find((candidate) => candidate.month === month);
      return { month, amount: amount ? centsInput(amount.cents) : '' };
    }),
  };
}

/** A typed custom month amount. */
export function changeCustomMonth(
  state: LineItemFormState,
  month: string,
  amount: string,
): LineItemFormState {
  return {
    ...state,
    customTyped: true,
    customMonths: state.customMonths.map((candidate) =>
      candidate.month === month ? { month, amount } : candidate,
    ),
  };
}

/** A month added before or after the custom grid (FR-018). */
export function addCustomMonth(state: LineItemFormState, month: string): LineItemFormState {
  const months = [...state.customMonths, { month, amount: '' }].sort((a, b) =>
    a.month.localeCompare(b.month),
  );
  return { ...state, customTyped: true, customMonths: months };
}

/**
 * Another finance budget year: an item of another year is cleared on save and the field says so;
 * going back to its year brings it back (FR-015, D-151).
 */
export function changeBudgetYear(
  state: LineItemFormState,
  year: number | null,
  items: FinanceBudgetItemRef[],
): LineItemFormState {
  const next = withField(state, 'financeBudgetYear', year);
  delete next.sources.financeBudgetItemId;
  if (state.clearedItem && state.clearedItem.year === year) {
    return { ...next, financeBudgetItemId: state.clearedItem.id, clearedItem: null };
  }
  const chosen = items.find((item) => item.id === state.financeBudgetItemId);
  if (chosen && chosen.year !== year) {
    return { ...next, financeBudgetItemId: '', clearedItem: chosen };
  }
  return next;
}

/**
 * The request of the form (D-149): amounts as typed, in the line item's currency, with at most two
 * decimals (the money field rounds more, D-190); an empty amount is null; custom months only for
 * Custom, without the empty ones. `attributes` are the values to send (cleared ones removed).
 */
export function toCreateBody(
  state: LineItemFormState,
  attributes: AttributeValues = state.attributes,
): LineItemCreate {
  return {
    name: state.name,
    ownerId: state.ownerId || null,
    description: state.description,
    currency: state.currency,
    plannedCost: amountForApi(state.plannedCost),
    spread: {
      method: state.spreadMethod,
      months:
        state.spreadMethod === 'custom'
          ? state.customMonths
              .filter((month) => month.amount.trim() !== '')
              .map((month) => ({ month: month.month, amount: amountForApi(month.amount) ?? '' }))
          : [],
    },
    attributes,
    contractedAmount: amountForApi(state.contractedAmount),
    requestSwbs: state.requestSwbs,
    financeBudgetYear: state.financeBudgetYear,
    financeBudgetItemId: state.financeBudgetItemId || null,
    restartApproval: false,
  };
}

/** The update request: the whole form with the version the person started from (D-153). */
export function toUpdateBody(
  state: LineItemFormState,
  version: number,
  attributes: AttributeValues = state.attributes,
) {
  return { ...toCreateBody(state, attributes), version };
}

/**
 * True when saving needs "This activity will need approval again" first (FR-031): a new line item,
 * or a changed planned cost or currency, while the activity is submitted or approved. Other edits
 * never restart approval (FR-032). `stored` is null on the create form.
 */
export function needsRestartConfirmation(
  activity: LineItemActivity,
  stored: LineItemFormState | null,
  current: LineItemFormState,
): boolean {
  if (!activity.restartsApproval) {
    return false;
  }
  if (!stored) {
    return true;
  }
  return (
    stored.currency !== current.currency ||
    parseAmount(stored.plannedCost) !== parseAmount(current.plannedCost)
  );
}

/** The changes the restart dialog lists (FR-031). */
export function restartReasons(
  stored: LineItemFormState | null,
  current: LineItemFormState,
): string[] {
  const after = costText(current);
  if (!stored) {
    return [newLineItemReason(current.name.trim(), after)];
  }
  return [costChangeReason(stored.name, costText(stored), after)];
}

/** The API refused a save because the person must confirm the restart first (D-146, D-189). */
export function isRestartRefusal(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    error.kind === 'validation' &&
    error.fields.some((problem) => problem.field === 'restartApproval')
  );
}

/** The input id of a custom month: "spread.month.2026-02". */
export function monthFieldId(month: string): string {
  return `spread.month.${month}`;
}

/**
 * The API's field problems with the field names of the form: a custom month's problem
 * ("spread.months[1].amount") goes to that month's input; the restart refusal is left out.
 */
export function formProblems(
  error: ApiError,
  body: LineItemCreate,
): { field: string; message: string }[] {
  return error.fields
    .filter((problem) => problem.field !== 'restartApproval')
    .map((problem) => {
      const month = /^spread\.months\[(\d+)\]/.exec(problem.field);
      const sent = month ? body.spread.months[Number(month[1])] : undefined;
      return sent ? { ...problem, field: monthFieldId(sent.month) } : problem;
    });
}

/** "20,000.00 USD", or the typed text when it is not an amount. */
function costText(state: LineItemFormState): string {
  const cents = parseAmount(state.plannedCost);
  return cents === null
    ? `${state.plannedCost} ${state.currency}`
    : amountText(cents, state.currency);
}

/** A typed amount as the API takes it; null when empty; the text itself when it is no amount. */
function amountForApi(text: string): string | null {
  const trimmed = text.trim();
  if (trimmed === '') {
    return null;
  }
  return parseMoneyInput(trimmed)?.amount ?? trimmed;
}
