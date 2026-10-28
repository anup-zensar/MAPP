import { formatMoneyText } from '@/format/decimal';
import type { Money } from './api';

export const EDIT = 'Edit';
export const DETAILS_TITLE = 'Details';
export const GSAP_TITLE = 'GSAP data';
export const GSAP_SUBTITLE = 'Read-only: filled by GSAP once it answers this purchase request.';
export const GSAP_EMPTY = 'Empty until GSAP answers this purchase request.';
export const LINES_TITLE = 'Lines';
export const ROUTING_TITLE = 'Routing history';
export const ROUTING_SUBTITLE = 'Each hand-off of the PR form to the requisitioner, newest first';
export const APPROVED_VERSION_TITLE = 'Approved version';
export const CHANGED = 'Changed';
export const NEW_LINE = 'New';
export const SHELL_ATTRIBUTES = 'Shell attributes';
export const ATTRIBUTES_FAILED = "The Shell attributes couldn't be loaded.";
export const TRY_AGAIN = 'Try again';

export const DOWNLOAD_FORM = 'Download PR form';
export const RESEND_FORM = 'Resend PR form';
export const CLOSE_PR = 'Close purchase request';
export const DELETE_PR = 'Delete purchase request';
export const VIEW_HISTORY = 'View history';
export const GO_TO_PURCHASE_REQUESTS = 'Go to Purchase requests';
export const GO_TO_PLANS = 'Go to Plans';
export const CANCEL = 'Cancel';

export const RESEND_TEXT =
  "SP&A One records a new routing with today's values. Sent dates do not change.";
export const CLOSE_INTRO =
  'Each line is reduced to what GSAP has invoiced; the rest returns to its line item.';
export const CLOSE_FINAL = "A closed purchase request can't be changed or reopened.";

export const CLOSE_COLUMNS = {
  line: 'Line item',
  amount: 'Current amount',
  invoiced: 'Invoiced',
  closed: 'Amount after closing',
  released: 'To be released',
} as const;

export function resendTitle(name: string): string {
  return `Resend the PR form of '${name}'?`;
}

export function closeTitle(name: string): string {
  return `Close purchase request '${name}'?`;
}

export function closedToast(name: string): string {
  return `Purchase request '${name}' closed`;
}

export function resentToast(name: string): string {
  return `PR form of '${name}' routed again`;
}

export function releasedText(total: Money): string {
  return `${formatMoneyText(total)} returns to the plan in total.`;
}

export function deleteTitle(name: string): string {
  return `Delete purchase request '${name}'?`;
}

/** "Its 2 lines will be removed and the money reserved on line items L-1006 and L-1011 will be released. …" */
export function deleteText(lineItemIds: string[]): string {
  const lines = lineItemIds.length === 1 ? 'Its 1 line' : `Its ${lineItemIds.length} lines`;
  const items =
    lineItemIds.length === 1 ? `line item ${lineItemIds[0]}` : `line items ${andList(lineItemIds)}`;
  return `${lines} will be removed and the money reserved on ${items} will be released. You can restore it from the recycle bin.`;
}

function andList(parts: string[]): string {
  return parts.length <= 1
    ? (parts[0] ?? '')
    : `${parts.slice(0, -1).join(', ')} and ${parts[parts.length - 1]}`;
}

export function routedText(type: string, recipients: string[]): string {
  return `${type} · routed to ${recipients.join(', ')}`;
}

export function routedByText(when: string, who: string, lineCount: number): string {
  return `${when} by ${who} · ${lineCount === 1 ? '1 line' : `${lineCount} lines`}`;
}

export function downloadLabel(routingId: string, format: string): string {
  return `Download PR form ${routingId} as ${format}`;
}

export function bookkeepingText(
  created: string,
  createdOn: string,
  updated: string,
  updatedOn: string,
) {
  return `Created by ${created} on ${createdOn}, last changed by ${updated} on ${updatedOn}`;
}

export function approvedOnText(when: string): string {
  return `The values approved on ${when}. They stay in force until the change is approved or declined.`;
}

export function poSourceText(source: string): string {
  return `PO number, line, status, date and value: filled by ${source}`;
}

export function inactive(name: string, active: boolean): string {
  return active ? name : `${name} (inactive)`;
}
