// The one place that knows each record's address (D-194): pages link to records only through these.
export type PlanningRecordType = 'plan' | 'campaign' | 'initiative' | 'activity' | 'lineItem';

const PLANNING_RECORD_TYPES: readonly PlanningRecordType[] = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
];

export function isPlanningType(type: string): type is PlanningRecordType {
  return PLANNING_RECORD_TYPES.some((known) => known === type);
}

export type PurchaseRecordType = 'purchaseRequest' | 'purchaseRequestLine';

export function purchaseRequestHref(id: string): string {
  return `/purchase-requests/${encodeURIComponent(id)}`;
}

/** Approval rows and notifications name their subject by ID; a purchase request's starts "PR-". */
export function isPurchaseRequestId(id: string): boolean {
  return id.startsWith('PR-');
}

/** Campaigns and initiatives open over their plan (`planId`); else an activity gets its short link. */
export function recordHref(
  type: PlanningRecordType | PurchaseRecordType,
  id: string,
  planId?: string,
): string {
  const path = encodeURIComponent(id);
  switch (type) {
    case 'purchaseRequest':
      return purchaseRequestHref(id);
    case 'purchaseRequestLine':
      return '/purchase-requests';
    case 'plan':
      return `/plans/${path}`;
    case 'campaign':
      return `${planPath(type, planId)}/campaigns/${path}`;
    case 'initiative':
      return `${planPath(type, planId)}/initiatives/${path}`;
    case 'activity':
      return planId
        ? `/plans/${encodeURIComponent(planId)}/activities/${path}`
        : `/activities/${path}`;
    case 'lineItem':
      return `/line-items/${path}`;
  }
}

/** An edit form with one field focused: `?focus=` above activities, else `?field=`. */
export function fieldHref(
  type: PlanningRecordType,
  id: string,
  field: string,
  planId?: string,
): string {
  if (type === 'activity' && !planId) {
    throw new Error(`A link to a field of activity ${id} needs its plan ID.`);
  }
  const parameter = type === 'activity' || type === 'lineItem' ? 'field' : 'focus';
  return `${recordHref(type, id, planId)}/edit?${new URLSearchParams({ [parameter]: field })}`;
}

function planPath(type: PlanningRecordType, planId: string | undefined): string {
  if (!planId) {
    throw new Error(`A link to a ${type} needs its plan ID.`);
  }
  return `/plans/${encodeURIComponent(planId)}`;
}
