/**
 * The one place that knows the address of every planning record (D-194; owned by feature 004,
 * entries of 005–007 included). Completeness lists, the publish dialog, the Ready to submit panel,
 * Home, All work and the activity log link to records only through these functions.
 *
 * | Record     | Page or panel                              | A field to fill in                 |
 * | plan       | /plans/<id>                                | /plans/<id>/edit?focus=<field>     |
 * | campaign   | /plans/<planId>/campaigns/<id>             | …/campaigns/<id>/edit?focus=<field> |
 * | initiative | /plans/<planId>/initiatives/<id>           | …/initiatives/<id>/edit?focus=…    |
 * | activity   | /plans/<planId>/activities/<id> (no plan:  | …/activities/<id>/edit?field=…     |
 * |            | the short /activities/<id>, which redirects) |                                  |
 * | lineItem   | /line-items/<id>                           | /line-items/<id>/edit?field=<field> |
 */

/** The planning records that have their own page or panel. */
export type PlanningRecordType = 'plan' | 'campaign' | 'initiative' | 'activity' | 'lineItem';

const PLANNING_RECORD_TYPES: readonly PlanningRecordType[] = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
];

/** Whether a record type of the API (a string in the contract) is one of these records. */
export function isPlanningType(type: string): type is PlanningRecordType {
  return PLANNING_RECORD_TYPES.some((known) => known === type);
}

/**
 * The address of a record's page or panel. Campaigns and initiatives open as panels over their
 * plan's page, so they need `planId`; an activity without it gets the short address that finds
 * its plan (D-194).
 */
export function recordHref(type: PlanningRecordType, id: string, planId?: string): string {
  const path = encodeURIComponent(id);
  switch (type) {
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

/**
 * The address of a record's edit form with one field focused, for "3 fields needed" lists: plans,
 * campaigns and initiatives use `?focus=`, activities and line items `?field=` (their plans).
 * Activities need `planId` here: their edit page has no short address.
 */
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
