/**
 * Every web text of the line-item screens (007 plan, "User-visible messages"): constants and small
 * functions for the texts built from data. Code and tests quote them from here.
 */
import type { components } from '@/api/generated/schema';
import { plural } from '@/format/text';
import type { SpreadMethod } from './line-item-math';

type SwbsStatus = components['schemas']['SwbsStatus'];
type ActivityStatus = components['schemas']['ActivityStatus'];

export const EMPTY = '—';

// Section on the activity page
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

/** "Planned cost (USD)", or with a display currency "Planned cost (EUR)" (FR-005). */
export function usdColumn(code: string): string {
  return `Planned cost (${code})`;
}

/** "In plan currency (GBP)": the plan-currency column of the Line items section (016 FR-015). */
export function planCurrencyColumn(code: string): string {
  return `In plan currency (${code})`;
}

/** "100,000.00 of 120,000.00 USD assigned". */
export function meterText(assigned: string, plannedCost: string, code: string): string {
  return `${assigned} of ${plannedCost} ${code} assigned`;
}

/** "20,000.00 USD not yet assigned to a line item". */
export function leftText(amount: string): string {
  return `${amount} not yet assigned to a line item`;
}

/** "Line items exceed the activity's planned cost by 500.00 USD". */
export function exceededText(amount: string): string {
  return `Line items exceed the activity's planned cost by ${amount}`;
}

// Grid
export const GRID_TITLE = 'Spread over time';
export const MONTHLY = 'Monthly';
export const QUARTERLY = 'Quarterly';
export const LINE_ITEM_COLUMN = 'Line item';
export const TOTAL = 'Total';
export const OUTSIDE_DATES = 'Outside dates';
export const OUTSIDE_TIP = "Outside the activity's dates — allowed, e.g. a deposit";
export const GRID_NO_DATES = "Add the activity's dates to spread its line items over months.";

/** "Line items spread over time, in USD". */
export function gridCaption(code: string): string {
  return `Line items spread over time, in ${code}`;
}

// Spread
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

/** "Mar 2026 · 3,333.33 EUR". */
export function previewLine(month: string, amount: string): string {
  return `${month} · ${amount}`;
}

/** "Amount for Feb 2026". */
export function monthAmountLabel(month: string): string {
  return `Amount for ${month}`;
}

/**
 * "12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much", "… — 500.00 EUR left" or "… —
 * adds up" (FR-018); amounts without their code, `difference` = planned cost − month total.
 */
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

// SWBS request
export const SWBS_TEXT: Record<SwbsStatus, string> = {
  notRequested: 'Not requested',
  waitingForApproval: 'Waiting for activity approval',
  readyToSend: 'Ready to send to GSAP',
};
export const SWBS_LATER = 'Filled by GSAP later';
export const SWBS_HELP =
  'MAPP asks GSAP for an SWBS code once the activity is approved. Sending requests to GSAP comes later.';

// Pages and form
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

/** "In activity 'Launch film' (A-1042)". */
export function inActivity(name: string, id: string): string {
  return `In activity '${name}' (${id})`;
}

/** The edit form's title (design system §2.3): the line item's name is in the breadcrumb. */
export const EDIT_TITLE = 'Edit line item';

// Summary bar
export const SUMMARY_NO_COST =
  'This activity has no planned cost yet — the unassigned planned cost appears once it has one.';

// Conversion
export const CALCULATED = 'Calculated';

/** A currency's rate in a conversion sentence: 0.92 units of EUR per 1 USD. */
export interface RatePart {
  rate: string;
  code: string;
}

/**
 * "= 10,869.57 USD at 0.92 EUR per USD"; into a plan currency other than USD "= 19,750.00 GBP at
 * 0.92 EUR and 0.79 GBP per USD"; with the rates' dates when known "… (rates of 19 Oct 2026)"
 * (016 FR-009, FR-015).
 */
export function conversionText(
  amount: string,
  code: string,
  rates: RatePart[],
  dates?: string | null,
): string {
  const parts = rates.map((part) => `${part.rate} ${part.code}`).join(' and ');
  return `= ${amount} ${code} at ${parts} per USD${dates ? ` (${dates})` : ''}`;
}

/** "rate of 19 Oct 2026", "rates of 19 Oct 2026", "rates of 1 Mar 2026 and 19 Oct 2026". */
export function rateDatesText(dates: string[]): string {
  const distinct = dates.filter((date, index) => dates.indexOf(date) === index);
  return `${dates.length === 1 ? 'rate' : 'rates'} of ${distinct.join(' and ')}`;
}

/** The ⓘ of the conversion line, into USD or into the plan currency (016 FR-015). */
export function conversionTip(code: string, planCode = 'USD'): string {
  if (planCode === 'USD') {
    return `Planned cost in USD = planned cost ÷ the rate in Settings (units of ${code} per 1 USD), rounded to the cent.`;
  }
  return `Planned cost in ${planCode}, the plan's currency = planned cost ÷ the ${code} rate × the ${planCode} rate (units per 1 USD), rounded to the cent.`;
}

/** "Changed to 18,400.00 EUR = 20,000.00 USD" (FR-012). */
export function changedToText(local: string, usd: string): string {
  return `Changed to ${local} = ${usd}`;
}

/** "EUR – Euro", "INR – Indian rupee (inactive)". */
export function currencyLabel(code: string, name: string, active: boolean): string {
  return `${code} – ${name}${active ? '' : ' (inactive)'}`;
}

/** "INR is no longer active — it still converts at its last rate (83.50 per USD)." */
export function inactiveCurrencyNote(code: string, rate: string): string {
  return `${code} is no longer active — it still converts at its last rate (${rate} per USD).`;
}

// Finance budget
export const BUDGET_SEARCH = 'Search item, budget or code';

/** "Items of '2026 Lubricants Budget', the plan's default finance budget, are listed first." */
export function listedFirstText(budget: string): string {
  return `Items of '${budget}', the plan's default finance budget, are listed first.`;
}

/** "Pre-selected — the only active item of '2026 Lubricants Budget'". */
export function preselectedText(budget: string): string {
  return `Pre-selected — the only active item of '${budget}'`;
}

/** "No finance budget items for 2027 yet — an administrator adds them in Settings." */
export function noBudgetItemsText(year: number): string {
  return `No finance budget items for ${year} yet — an administrator adds them in Settings.`;
}

/** "Finance budget item 'Europe – Digital' will be cleared — it belongs to 2026." */
export function itemClearedText(name: string, year: number): string {
  return `Finance budget item '${name}' will be cleared — it belongs to ${year}.`;
}

/** "Choose an active finance budget item — 'Europe – Brand' is no longer active." */
export function inactiveItemText(name: string): string {
  return `Choose an active finance budget item — '${name}' is no longer active.`;
}

// Footer and toasts
export const SAVE = 'Save';
export const SAVE_AND_ADD = 'Save and add another';
export const CANCEL = 'Cancel';
export const SAVING = 'Saving…';

/** "Line item 'Agency fee' created". */
export function createdToast(name: string): string {
  return `Line item '${name}' created`;
}

/** "Line item 'Agency fee' saved". */
export function savedToast(name: string): string {
  return `Line item '${name}' saved`;
}

// Restart dialog (FR-031)
export const RESTART_TITLE = 'This activity will need approval again';
export const RESTART_CONFIRM = 'Save and restart approval';
export const KEEP_EDITING = 'Keep editing';

/** "'Launch film' is approved. Saving sends it back to Created, …". */
export function restartText(activityName: string, status: ActivityStatus): string {
  const state = status === 'approved' ? 'is approved' : 'is submitted for approval';
  return `'${activityName}' ${state}. Saving sends it back to Created, and it must be submitted for approval again.`;
}

/** "New line item 'Agency fee': 10,000.00 EUR". */
export function newLineItemReason(name: string, amount: string): string {
  return `New line item '${name}': ${amount}`;
}

/** "Copy of 'Agency fee': 50,000.00 USD". */
export function copyReason(name: string, amount: string): string {
  return `Copy of '${name}': ${amount}`;
}

/** "Planned cost of 'Agency fee': 50,000.00 USD → 55,000.00 USD". */
export function costChangeReason(name: string, before: string, after: string): string {
  return `Planned cost of '${name}': ${before} → ${after}`;
}

// Delete dialog (FR-030)
/** "Delete line item 'Agency fee'?". */
export function deleteTitle(name: string): string {
  return `Delete line item '${name}'?`;
}

/** "The activity's unassigned planned cost goes up by 50,000.00 USD." */
export function deleteEffect(usd: string): string {
  return `The activity's unassigned planned cost goes up by ${usd}.`;
}

export const DELETE_TO_BIN = 'It moves to the recycle bin; you can restore it.';
export const DELETE_RESTART = 'This activity will need approval again.';
export const DELETE_CONFIRM = 'Delete line item';

// Details page
export const EDIT = 'Edit';
export const MORE = 'More';
export const DUPLICATE = 'Duplicate';
export const VIEW_HISTORY = 'View history';
export const DELETE = 'Delete';
export const VIEW = 'View';

/** "Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026". */
export function bookkeepingText(
  createdBy: string,
  createdOn: string,
  changedBy: string,
  changedOn: string,
): string {
  return `Created by ${createdBy} on ${createdOn}, last changed by ${changedBy} on ${changedOn}`;
}

/** "Actions for Agency fee". */
export function actionsFor(name: string): string {
  return `Actions for ${name}`;
}

// Gone (D-154)
export const LINE_ITEM_GONE = 'This line item no longer exists — someone may have deleted it.';
export const ACTIVITY_GONE =
  'This activity was deleted — restore it from the recycle bin to keep working on its line items.';
export const GO_TO_PLANS = 'Go to Plans';
export const GO_TO_BIN = 'Go to the recycle bin';

/** "All filled in", "1 field needed", "3 fields needed" (FR-028). */
export function completenessText(count: number): string {
  return count === 0 ? 'All filled in' : `${plural(count, 'field')} needed`;
}
