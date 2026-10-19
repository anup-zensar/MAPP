/**
 * How activities read on screen (006 plan, "User-visible messages"), and their addresses.
 */
import type { components } from '@/api/schema';
import { formatMoneyText, formatPercent } from '@/lib/decimal';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import { plural } from '@/lib/text';

/** The plan model's facts the goals are calculated with. */
export type ModelFacts = components['schemas']['ActivityModelFacts'];

/** Shown for an empty value (design system §6.3). */
export const EMPTY = '—';

/**
 * An activity's dates with both written in full: "01 Mar 2026 – 31 May 2026", "Ongoing from
 * 01 Mar 2026", or "—" while they are not both set. `date` writes one date in the person's format.
 */
export function datesText(
  start: string | null | undefined,
  end: string | null | undefined,
  ongoing: boolean,
  date: (date: string) => string,
): string {
  if (ongoing && start) {
    return `Ongoing from ${date(start)}`;
  }
  if (start && end) {
    return `${date(start)} – ${date(end)}`;
  }
  return EMPTY;
}

/** "1 thing", "5 things": what submitting still needs. */
export function thingsText(count: number): string {
  return plural(count, 'thing');
}

/** "1 line item", "3 line items". */
export function lineItemsText(count: number): string {
  return plural(count, 'line item');
}

/** The completeness chip's second text: "1 line item needs attention", "3 line items need attention". */
export function lineItemsAttentionText(count: number): string {
  return count === 1 ? '1 line item needs attention' : `${lineItemsText(count)} need attention`;
}

/** The activity page: /plans/:planId/activities/:id. */
export function activityHref(planId: string, id: string): string {
  return recordHref('activity', id, planId);
}

/** The edit form, with one field focused when given: …/edit?field=ownerId. */
export function activityEditHref(planId: string, id: string, field?: string): string {
  return field ? fieldHref('activity', id, field, planId) : `${activityHref(planId, id)}/edit`;
}

/** The new-activity form of an initiative. */
export function newActivityHref(planId: string, initiativeId: string): string {
  return `/plans/${encodeURIComponent(planId)}/initiatives/${encodeURIComponent(initiativeId)}/activities/new`;
}

/**
 * How MQL and revenue are calculated (FR-018): "MQL = INQ × 2.00 % · deals = MQL × 5.00 % ·
 * revenue = deals × 10,000.00 USD — each rounded to a whole number (model 'Lubricants 1.0')."
 */
export function goalsFormula(model: ModelFacts): string {
  return (
    `MQL = INQ × ${formatPercent(model.inqToMqlRate)} · deals = MQL × ${formatPercent(model.mqlToDealRate)} · ` +
    `revenue = deals × ${formatMoneyText(model.averageDealSize)} — each rounded to a whole number ` +
    `(model '${model.name}').`
  );
}

// Move and copy (US5, FR-029, FR-030; plan "Actions")
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
