import type { components } from '@/api/generated/schema';
import { purchaseRequestHref, recordHref } from '@/ui/record-links';
import type { PurchaseRequestProblem } from './api';

type ReadinessGroup = components['schemas']['ReadinessGroup'];

/** One thing that stops a submission, with where to fix it. */
export interface ProblemLink {
  key: string;
  text: string;
  href: string;
}

function editHref(purchaseRequestId: string, field: string): string {
  const edit = `${purchaseRequestHref(purchaseRequestId)}/edit`;
  return field ? `${edit}?${new URLSearchParams({ field })}` : edit;
}

function recordTarget(type: string, recordId: string, field: string, prId: string): string {
  if (type === 'activity') {
    return recordHref('activity', recordId);
  }
  return type === 'lineItem' ? recordHref('lineItem', recordId) : editHref(prId, field);
}

/** The submission check's gates (FR-021): a field opens the edit form, another record its page. */
export function problemLinks(
  purchaseRequestId: string,
  problems: PurchaseRequestProblem[],
): ProblemLink[] {
  return problems.map((problem, index) => ({
    key: `${index}-${problem.recordId}-${problem.field}`,
    text: problem.message,
    href: problem.field
      ? editHref(purchaseRequestId, problem.field)
      : recordTarget(problem.group, problem.recordId, '', purchaseRequestId),
  }));
}

/** A refused submission's missing fields, grouped by record (409 not_ready, error.missing). */
export function missingLinks(purchaseRequestId: string, groups: ReadinessGroup[]): ProblemLink[] {
  return groups.flatMap((group) =>
    group.fields.map((field) => ({
      key: `${group.record.id}-${field.field}`,
      text: `${group.record.name}: ${field.reason || field.label}`,
      href: recordTarget(group.record.type, group.record.id, field.field, purchaseRequestId),
    })),
  );
}
