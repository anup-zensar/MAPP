import type { BreadcrumbItem } from '@/ui/Breadcrumb/Breadcrumb';
import type { MissingField } from '@/ui/CompletenessChip/CompletenessChip';
import { isPlanningType, recordHref } from '@/ui/record-links';
import type { PurchaseRequest } from './api';

const RECORD = 'record:';

/** The breadcrumb's first step: Purchase requests for a person who may see it, else Plans. */
export function breadcrumbRoot(seesList: boolean): BreadcrumbItem {
  return seesList
    ? { label: 'Purchase requests', to: '/purchase-requests' }
    : { label: 'Plans', to: '/plans' };
}

/** Where the PR's lines live (plan › campaign › initiative › activity), then the PR itself. */
export function purchaseRequestBreadcrumb(
  pr: PurchaseRequest,
  seesList: boolean,
): BreadcrumbItem[] {
  return [
    breadcrumbRoot(seesList),
    ...pr.path.map((step) => ({
      label: step.name,
      to: isPlanningType(step.type) ? recordHref(step.type, step.id, pr.plan.id) : undefined,
    })),
    { label: pr.name },
  ];
}

/** The gates' problems (032 FR-052): a field opens the edit page, a record gate its record. */
export function missingOf(pr: PurchaseRequest): MissingField[] {
  const problems = pr.readiness.problems.map((problem) => ({
    id: problem.field || `${RECORD}${problem.group}:${problem.recordId}`,
    label: problem.message,
  }));
  const header = pr.missing.filter((field) => !problems.some((p) => p.id === field.field));
  return [...problems, ...header.map((field) => ({ id: field.field, label: field.label }))];
}

export function fieldTarget(field: string, prHref: string): string {
  if (!field.startsWith(RECORD)) {
    return `${prHref}/edit?${new URLSearchParams({ field })}`;
  }
  const [group = '', recordId = ''] = field.slice(RECORD.length).split(':');
  if (group === 'activity') {
    return recordHref('activity', recordId);
  }
  return group === 'lineItem' ? recordHref('lineItem', recordId) : `${prHref}/edit`;
}
