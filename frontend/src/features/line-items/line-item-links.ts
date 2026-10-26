import type { BreadcrumbItem } from '@/ui/Breadcrumb/Breadcrumb';
import { recordHref } from '@/ui/record-links';
import type { PlanningRecordType } from '@/ui/record-links';
import type { LineItemActivity } from './api';

/** The plan of the activity, the first step of its path. */
export function planIdOf(activity: LineItemActivity): string | undefined {
  return activity.path.find((step) => step.type === 'plan')?.id;
}

/** The activity's page, at its Line items section. */
export function activityLineItemsHref(activity: LineItemActivity): string {
  return `${recordHref('activity', activity.id, planIdOf(activity))}#line-items`;
}

/** Plans › plan › campaign › initiative › activity: where a line item lives (FR-026). */
export function activityBreadcrumb(activity: LineItemActivity): BreadcrumbItem[] {
  const planId = planIdOf(activity);
  return [
    { label: 'Plans', to: '/plans' },
    ...activity.path.map((step) => ({
      label: step.name,
      to: recordHref(step.type as PlanningRecordType, step.id, planId),
    })),
    { label: activity.name, to: recordHref('activity', activity.id, planId) },
  ];
}
