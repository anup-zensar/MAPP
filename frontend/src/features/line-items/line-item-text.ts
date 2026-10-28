/** Every web text of the line-item screens; code and tests quote them from here. */
import { plural } from '@/format/text';
import type { SpreadMethod } from './line-item-math';

export const EMPTY = '—';

export const SECTION_TITLE = 'Line items';
export const SECTION_SUBTITLE = 'The costs of this activity, each in its own currency';
export const ADD_LINE_ITEM = 'Add line item';
export const EMPTY_SECTION =
  "No line items yet — split this activity's planned cost into the costs you will pay";
export const BALANCE_LABEL = 'Unassigned planned cost (Sys_Gen_Balance)';
export const BALANCE_TIP =
  "Unassigned planned cost = the activity's planned cost − the sum of its line items' planned costs in USD. Aim for 0.00: line items above the planned cost stop the activity from being submitted.";
export const METER_LABEL = 'Assigned to line items';
export const ALL_ASSIGNED = 'All of the planned cost is assigned';
export const NO_PLANNED_COST =
  'This activity has no planned cost yet — add one to see how much is left to assign.';
export const ROUNDING_HINT =
  "If a currency rate leaves a few cents unassigned, adjust the activity's planned cost instead.";

/** "Planned cost (USD)", or with a display currency "Planned cost (EUR)". */
export function usdColumn(code: string): string {
  return `Planned cost (${code})`;
}

/** "In plan currency (GBP)": the plan-currency column of the Line items section. */
export function planCurrencyColumn(code: string): string {
  return `In plan currency (${code})`;
}

export function meterText(assigned: string, plannedCost: string, code: string): string {
  return `${assigned} of ${plannedCost} ${code} assigned`;
}

export function leftText(amount: string): string {
  return `${amount} not yet assigned to a line item`;
}

export function exceededText(amount: string): string {
  return `Line items exceed the activity's planned cost by ${amount}`;
}

export const GRID_TITLE = 'Spread over time';
export const MONTHLY = 'Monthly';
export const QUARTERLY = 'Quarterly';
export const LINE_ITEM_COLUMN = 'Line item';
export const TOTAL = 'Total';
export const OUTSIDE_DATES = 'Outside dates';
export const OUTSIDE_TIP = "Outside the activity's dates — allowed, e.g. a deposit";
export const GRID_NO_DATES = "Add the activity's dates to spread its line items over months.";

export function gridCaption(code: string): string {
  return `Line items spread over time, in ${code}`;
}

export const SPREAD_LABELS: Record<SpreadMethod, string> = {
  evenly: 'Evenly',
  firstMonth: 'All in first month',
  lastMonth: 'All in last month',
  custom: 'Custom',
};
export const NOT_SPREAD_YET = "Not spread yet — add the activity's dates to see the spread";
export const ONGOING_EVENLY =
  "This activity is ongoing (no end date), so 'Evenly' puts everything in the first month — choose Custom to spread it";
export const ONGOING_LAST =
  "This activity is ongoing (no end date), so 'All in last month' puts everything in the first month — choose Custom to spread it";
export const ADD_MONTH_BEFORE = 'Add month before';
export const ADD_MONTH_AFTER = 'Add month after';
export const USE_MONTH_TOTAL = 'Use month total as planned cost';
export const LEAVING_CUSTOM =
  'The month amounts you typed will be replaced by this spread when you save.';
export const SPREAD_NOT_ADDING_UP = "Spread doesn't add up to the planned cost";

export function previewLine(month: string, amount: string): string {
  return `${month} · ${amount}`;
}

export function monthAmountLabel(month: string): string {
  return `Amount for ${month}`;
}

export function customTotalText(
  sum: string,
  cost: string,
  code: string,
  difference: { sign: number; amount: string },
): string {
  const start = `${sum} of ${cost} ${code} spread — `;
  if (difference.sign === 0) {
    return `${start}adds up`;
  }
  return `${start}${difference.amount} ${code} ${difference.sign > 0 ? 'left' : 'too much'}`;
}

export const EYEBROW = 'Line item';
export const NEW_TITLE = 'New line item';
export const SECTION_BASICS = 'Basics';
export const SECTION_COST = 'Cost and currency';
export const SECTION_SPREAD = 'Spread over time';
export const SECTION_FINANCE_CODING = 'Finance coding';
export const SECTION_BUDGET = 'Link to finance budget';
export const SECTION_GSAP = 'GSAP request';

export const LABEL = {
  name: 'Line item name',
  owner: 'Owner',
  description: 'Description',
  currency: 'Currency',
  plannedCost: 'Planned cost',
  spread: 'Spread over time',
  contractedAmount: 'Contracted amount',
  financeBudgetYear: 'Finance budget year',
  financeBudgetItem: 'Finance budget item',
  requestSwbs: 'Request SWBS code from GSAP',
  swbsCode: 'SWBS code',
} as const;

export const SOURCE = {
  name: "From the activity's name",
  owner: 'From the activity',
  currency: 'Default',
  plannedCost: "The activity's unassigned planned cost",
  spread: "The plan's default",
  year: "The plan's year",
  requestSwbs: 'On by default',
} as const;

export const NOTHING_LEFT =
  "Nothing left to assign — the activity's planned cost is fully assigned";
export const NO_COST_TO_PREFILL =
  'This activity has no planned cost yet, so there is nothing to pre-fill';

export function inActivity(name: string, id: string): string {
  return `In activity '${name}' (${id})`;
}

export const EDIT_TITLE = 'Edit line item';

export const SUMMARY_NO_COST =
  'This activity has no planned cost yet — the unassigned planned cost appears once it has one.';

export const CALCULATED = 'Calculated';

export interface RatePart {
  rate: string;
  code: string;
}

export function conversionText(
  amount: string,
  code: string,
  rates: RatePart[],
  dates?: string | null,
): string {
  const parts = rates.map((part) => `${part.rate} ${part.code}`).join(' and ');
  return `= ${amount} ${code} at ${parts} per USD${dates ? ` (${dates})` : ''}`;
}

export function rateDatesText(dates: string[]): string {
  const distinct = dates.filter((date, index) => dates.indexOf(date) === index);
  return `${dates.length === 1 ? 'rate' : 'rates'} of ${distinct.join(' and ')}`;
}

export function conversionTip(code: string, planCode = 'USD'): string {
  if (planCode === 'USD') {
    return `Planned cost in USD = planned cost ÷ the rate in Settings (units of ${code} per 1 USD), rounded to the cent.`;
  }
  return `Planned cost in ${planCode}, the plan's currency = planned cost ÷ the ${code} rate × the ${planCode} rate (units per 1 USD), rounded to the cent.`;
}

export function changedToText(local: string, usd: string): string {
  return `Changed to ${local} = ${usd}`;
}

export function currencyLabel(code: string, name: string, active: boolean): string {
  return `${code} – ${name}${active ? '' : ' (inactive)'}`;
}

export function inactiveCurrencyNote(code: string, rate: string): string {
  return `${code} is no longer active — it still converts at its last rate (${rate} per USD).`;
}

export const BUDGET_SEARCH = 'Search item, budget or code';

export function listedFirstText(budget: string): string {
  return `Items of '${budget}', the plan's default finance budget, are listed first.`;
}

export function preselectedText(budget: string): string {
  return `Pre-selected — the only active item of '${budget}'`;
}

export function noBudgetItemsText(year: number): string {
  return `No finance budget items for ${year} yet — an administrator adds them in Settings.`;
}

export function itemClearedText(name: string, year: number): string {
  return `Finance budget item '${name}' will be cleared — it belongs to ${year}.`;
}

export function inactiveItemText(name: string): string {
  return `Choose an active finance budget item — '${name}' is no longer active.`;
}

export const SAVE = 'Save';
export const SAVE_AND_ADD = 'Save and add another';
export const CANCEL = 'Cancel';
export const SAVING = 'Saving…';

export function createdToast(name: string): string {
  return `Line item '${name}' created`;
}

export function savedToast(name: string): string {
  return `Line item '${name}' saved`;
}

export function deleteTitle(name: string): string {
  return `Delete line item '${name}'?`;
}

export function deleteEffect(usd: string): string {
  return `The activity's unassigned planned cost goes up by ${usd}.`;
}

export const DELETE_TO_BIN = 'It moves to the recycle bin; you can restore it.';
export const DELETE_CONFIRM = 'Delete line item';

export const EDIT = 'Edit';
export const MORE = 'More';
export const DUPLICATE = 'Duplicate';
export const VIEW_HISTORY = 'View history';
export const DELETE = 'Delete';
export const VIEW = 'View';

export function bookkeepingText(
  createdBy: string,
  createdOn: string,
  changedBy: string,
  changedOn: string,
): string {
  return `Created by ${createdBy} on ${createdOn}, last changed by ${changedBy} on ${changedOn}`;
}

export function actionsFor(name: string): string {
  return `Actions for ${name}`;
}

export const LINE_ITEM_GONE = 'This line item no longer exists — someone may have deleted it.';
export const ACTIVITY_GONE =
  'This activity was deleted — restore it from the recycle bin to keep working on its line items.';
export const GO_TO_PLANS = 'Go to Plans';
export const GO_TO_BIN = 'Go to the recycle bin';

export function completenessText(count: number): string {
  return count === 0 ? 'All filled in' : `${plural(count, 'field')} needed`;
}
