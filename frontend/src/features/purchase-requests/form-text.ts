import { formatMoneyText } from '@/format/decimal';
import type { Money } from './api';

export const HEADER_SECTION = 'Purchase request';
export const LINES_SECTION = 'Lines';
export const LINES_DESCRIPTION =
  'Each line claims money of one line item; it is reserved as soon as you save.';
export const ADD_LINE = 'Add line';
export const REMOVE_LINE = 'Remove line';
export const SAVE = 'Save';
export const SAVE_AND_RESUBMIT = 'Save and re-submit';
export const SUBMIT_CHANGES = 'Submit changes';
export const CHANGE_ROUND_STARTS =
  'A change approval round starts. The purchase request stays approved as it was until the change is approved.';
export const STAYS_RESERVED = 'The approved amounts stay reserved until the change is approved.';

export function submitChangesTitle(name: string): string {
  return `Submit changes to '${name}'?`;
}
export const SAVING = 'Saving…';
export const CANCEL = 'Cancel';
export const RELOAD = 'Reload';
export const ATTRIBUTES_TITLE = 'Shell attributes';
export const LINE_ATTRIBUTES_TITLE = 'PR line attributes';

export const LABELS = {
  name: 'Name',
  description: 'Description',
  owner: 'Owner',
  moaApprover: 'MoA approver',
  approvalWorkflow: 'Approval workflow',
  modificationWorkflow: 'Approval workflow for modification',
  currency: 'Currency',
  amount: 'Amount',
  deliveryDate: 'Delivery date',
  overallLimit: 'Overall limit',
  lineDescription: 'Line description',
  hxKey: 'HX key',
  swbsCode: 'SWBS code',
} as const;

export const MOA_HINT =
  'The person whose approval is named in the PR form (Manual of Authorities).';
export const WORKFLOW_HINT = 'Only active workflows for purchase requests are offered.';
export const MODIFICATION_HINT = 'Used when an approved purchase request is changed.';
export const OVERALL_LIMIT_HINT = 'Defaults to the amount.';
export const SWBS_HINT = 'Filled by GSAP through the line item';
export const NO_SWBS = 'Not received yet';
export const LAST_LINE =
  'A purchase request needs at least one line — delete the purchase request instead.';
export const NOT_SAVED_LINE = 'Not saved yet: nothing is reserved until you save.';

export const PROBLEMS = {
  name: 'Enter a name for the purchase request.',
  nameLength: 'The name can be at most 120 characters.',
  descriptionLength: 'The description can be at most 2,000 characters.',
  owner: 'Choose the owner.',
  approvalWorkflow: 'Choose the approval workflow.',
  modificationWorkflow: 'Choose the approval workflow for modification.',
  lines: 'Add at least one line.',
  currency: 'Choose the currency.',
  amount: 'Enter an amount above 0.00, with at most two decimals.',
  deliveryDate: 'Enter the delivery date.',
  overallLimit: 'Enter an overall limit of 0.00 or more, or leave it empty.',
} as const;

export const CONFLICT_NEXT_STEP =
  'Your entries are still here. Reload to see their changes; saving then replaces them with yours.';
export const NOT_READY_TITLE = 'Not saved — these things are needed first:';

export const PICKER = {
  title: 'Add a line',
  description:
    'Line items of this plan and company code, with what each still has available to request.',
  search: 'Search line items',
  placeholder: 'Search name, ID or HX key',
  available: 'Available to request',
  loading: 'Looking for line items…',
  failed: "The line items couldn't be loaded.",
  tryAgain: 'Try again',
  close: 'Close',
  alreadyOnForm: 'Already on this purchase request',
  choose: 'Choose',
  stepTitle: 'Choose a line item',
  stepDescription:
    'A purchase request starts from a line item: choose the one to request money from.',
} as const;

export function noMatchText(query: string): string {
  return query ? `No line items match '${query}'.` : 'No line items to choose from.';
}

/** "Add line Gift boxes (L-1011)": the visible action first, so the name contains it (WCAG 2.5.3). */
export function chooseLabel(action: string, name: string, id: string): string {
  return `${action} ${name} (${id})`;
}

export function removeTitle(lineItemId: string, amount: Money): string {
  return `Remove the line for line item ${lineItemId} (${formatMoneyText(amount)}) from this purchase request?`;
}

export function removeText(amount: Money): string {
  return `The ${formatMoneyText(amount)} reserved on the line item will be released.`;
}

export const REMOVE_AFTER_SAVE = 'The line goes to the recycle bin when you save.';

export function lineTitle(position: number, name: string, id: string): string {
  return `Line ${position} · ${name} (${id})`;
}

export function createdToast(name: string): string {
  return `Purchase request '${name}' created`;
}

export function savedToast(name: string): string {
  return `Purchase request '${name}' saved`;
}
