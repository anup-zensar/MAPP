import type { components } from '@/api/generated/schema';
import { formatMoneyText, formatPercent } from '@/format/decimal';
import { fieldHref, recordHref } from '@/ui/record-links';
import { plural } from '@/format/text';
import type { Formatters } from '@/format/useFormatters';

export type ModelFacts = components['schemas']['ActivityModelFacts'];

export const EMPTY = '—';

export const MOVE_TO = 'Move to…';

export type DateWriters = Pick<Formatters, 'date' | 'dateRange'>;

/** "01 Mar – 31 May 2026", "Ongoing from 01 Mar 2026", or "—" until both are set (D-216). */
export function datesText(
  start: string | null | undefined,
  end: string | null | undefined,
  ongoing: boolean,
  format: DateWriters,
): string {
  if (ongoing && start) {
    return `Ongoing from ${format.date(start)}`;
  }
  if (start && end) {
    return format.dateRange(start, end);
  }
  return EMPTY;
}

export function thingsText(count: number): string {
  return plural(count, 'thing');
}

export function lineItemsText(count: number): string {
  return plural(count, 'line item');
}

export function lineItemsAttentionText(count: number): string {
  return count === 1 ? '1 line item needs attention' : `${lineItemsText(count)} need attention`;
}

export function activityHref(planId: string, id: string): string {
  return recordHref('activity', id, planId);
}

export function activityEditHref(planId: string, id: string, field?: string): string {
  return field ? fieldHref('activity', id, field, planId) : `${activityHref(planId, id)}/edit`;
}

export function newActivityHref(planId: string, initiativeId: string): string {
  return `/plans/${encodeURIComponent(planId)}/initiatives/${encodeURIComponent(initiativeId)}/activities/new`;
}

/** "MQL = INQ × 2.00 % · … — each rounded to a whole number (model 'Lubricants 1.0')". */
export function goalsFormula(model: ModelFacts): string {
  return (
    `MQL = INQ × ${formatPercent(model.inqToMqlRate)} · deals = MQL × ${formatPercent(model.mqlToDealRate)} · ` +
    `revenue = deals × ${formatMoneyText(model.averageDealSize)} — each rounded to a whole number ` +
    `(model '${model.name}').`
  );
}

export function moveTitle(name: string): string {
  return `Move activity '${name}'`;
}
export function moveIntro(plan: string): string {
  return `Choose another initiative in '${plan}'. The activity keeps its status, line items and values.`;
}
export const SEARCH_INITIATIVES = 'Search initiatives';
export const THIS_INITIATIVE = '(this initiative)';
export const NO_OTHER_INITIATIVE = "There's no other initiative in this plan to move it to.";
/** Move without a chosen initiative: the API's own words for it. */
export const CHOOSE_ANOTHER_INITIATIVE = 'Choose another initiative of this plan.';
/** The search finds no initiative (the wording of 002's searchable lists). */
export function noInitiativeMatches(query: string): string {
  return `No initiative matches '${query}'.`;
}
export function movedToast(name: string, initiative: string): string {
  return `Activity '${name}' moved to '${initiative}'`;
}
export function copyTitle(name: string): string {
  return `Copy activity '${name}'`;
}
export const COPY_TO = 'Copy to';
export function copyIntro(lineItems: number): string {
  return `The copy starts as Created, with copies of its ${lineItemsText(lineItems)}. Approvals, comments and history are not copied.`;
}
export function createdToast(name: string): string {
  return `Activity '${name}' created`;
}
