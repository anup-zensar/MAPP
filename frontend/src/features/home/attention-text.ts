/** The words of Needs your attention (feature 009, FR-011, FR-012). */
import { formatWhole } from '@/format/decimal';
import { localDate } from '@/format/intl';
import type { HomeAttentionItem, HomeAttentionReason, Money } from './api';
import { lineItemsNeedAttentionText } from './home-text';

/** How a reason's sentence shows amounts and dates: the signed-in person's formatters. */
export interface ReasonFormatters {
  /** "$1,200.00" for an amount. */
  money: (value: Money | null | undefined) => string;
  /** "12 Mar 2026" (or the person's format) for a calendar date "YYYY-MM-DD". */
  date: (date: string) => string;
}

/**
 * "3 fields needed · 1 line item needs attention": what is missing on a record and below it
 * (FR-011 e); only the parts that apply. An activity's items below are its line items, so it says
 * so — the same words My activities uses for the same activity (functional review X-07).
 */
export function missingFieldsText(
  missing: number,
  itemsBelow: number,
  recordType: HomeAttentionItem['recordType'] = 'plan',
): string {
  const parts: string[] = [];
  if (missing > 0) {
    parts.push(missing === 1 ? '1 field needed' : `${formatWhole(missing)} fields needed`);
  }
  if (itemsBelow > 0) {
    parts.push(
      recordType === 'activity'
        ? lineItemsNeedAttentionText(itemsBelow)
        : itemsBelow === 1
          ? '1 item below needs attention'
          : `${formatWhole(itemsBelow)} items below need attention`,
    );
  }
  return parts.join(' · ');
}

/** The sentence of one reason a record needs attention (FR-011). */
export function reasonText(
  reason: HomeAttentionReason,
  format: ReasonFormatters,
  recordType: HomeAttentionItem['recordType'] = 'plan',
): string {
  switch (reason.kind) {
    case 'declined': {
      const by = reason.declinedBy ? ` by ${reason.declinedBy.name}` : '';
      const on = reason.declinedAt ? ` on ${format.date(localDate(reason.declinedAt))}` : '';
      const comment = reason.comment ? `: '${reason.comment}'` : '';
      return `Declined${by}${on}${comment}`;
    }
    case 'negativeBalance':
      return `Line items exceed the planned cost by ${format.money(reason.amount)}`;
    case 'overAllocated':
      return `Over-allocated by ${format.money(reason.amount)}`;
    case 'overBudget':
      return `Planned cost is ${format.money(reason.amount)} over budget`;
    case 'missingFields':
      return missingFieldsText(reason.missing.length, reason.itemsBelow, recordType);
  }
}

/** A link to where a reason is fixed: its words and the section of the record's page. */
export interface ReasonLink {
  label: string;
  hash: string;
}

/** Where each reason is fixed: the link's words and the section of the record's page (FR-012). */
export const REASON_LINK: Record<HomeAttentionReason['kind'], ReasonLink> = {
  declined: { label: 'Open approval', hash: '#approval' },
  negativeBalance: { label: 'Open line items', hash: '#line-items' },
  overAllocated: { label: 'Open budget', hash: '#budget' },
  overBudget: { label: 'Open budget', hash: '#budget' },
  missingFields: { label: 'Show fields', hash: '#completeness' },
};

/**
 * The link of a reason. "Show fields" only where fields are missing: when only items below need
 * attention, an activity opens its line items and other records show what is needed below them
 * (functional review X-07).
 */
export function reasonLinkOf(
  reason: HomeAttentionReason,
  recordType: HomeAttentionItem['recordType'],
): ReasonLink {
  if (reason.kind === 'missingFields' && reason.missing.length === 0) {
    return recordType === 'activity'
      ? REASON_LINK.negativeBalance
      : { label: 'Show items below', hash: '#completeness' };
  }
  return REASON_LINK[reason.kind];
}

/** "Open approval for activity 'Launch film'": a reason link's name names its record. */
export function reasonLinkName(label: string, typeLabel: string, name: string): string {
  return `${label} for ${typeLabel} '${name}'`;
}

/** Shown when nothing the person owns has a problem (FR-021, US3 scenario 6). */
export const NOTHING_NEEDS_ATTENTION = 'Nothing needs your attention right now.';
