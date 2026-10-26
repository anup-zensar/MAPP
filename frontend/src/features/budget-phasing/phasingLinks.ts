/**
 * Links to a record's **Budget by month** (016 T053): a phasing that does not add up to the budget
 * is a needed-to-publish problem named "budgetPhasing" in the record's completeness (data model
 * §3.3), and its link leads to the card — on the plan page, or in the campaign's or initiative's
 * panel — instead of an edit form that has no such field.
 */
import { fieldHref, recordHref } from '@/ui/record-links';
import type { PhasingLevel } from './phasingApi';

/** The completeness field of a phasing problem. */
export const PHASING_FIELD = 'budgetPhasing';

/** The id of a record's Budget by month card, and the address's hash that leads to it. */
export function phasingAnchor(recordId: string): string {
  return `budget-by-month-${recordId}`;
}

/** The record's page or panel, scrolled to its Budget by month card. */
export function phasingHref(level: PhasingLevel, recordId: string, planId: string): string {
  return `${recordHref(level, recordId, planId)}#${phasingAnchor(recordId)}`;
}

/**
 * Where a "fields needed" list sends a missing field: Budget by month for the phasing problem, the
 * edit form with that field focused for every other field.
 */
export function completenessFieldHref(
  level: PhasingLevel,
  recordId: string,
  field: string,
  planId: string,
): string {
  return field === PHASING_FIELD
    ? phasingHref(level, recordId, planId)
    : fieldHref(level, recordId, field, planId);
}
