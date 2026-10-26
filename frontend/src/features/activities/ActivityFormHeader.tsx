import { PageHeader } from '@/ui/PageHeader/PageHeader';
import type { BreadcrumbItem } from '@/ui/Breadcrumb/Breadcrumb';
import { recordHref } from '@/ui/record-links';
import type { ActivityFormData } from './api';
import { activityHref } from './activity-text';

export interface ActivityFormHeaderProps {
  title: string;
  form?: ActivityFormData;
  activity?: { id: string; name: string };
}

/** The breadcrumb names where the activity lives, so no eyebrow repeats it (UI audit X-03). */
export function ActivityFormHeader({ title, form, activity }: ActivityFormHeaderProps) {
  const items: BreadcrumbItem[] = [{ label: 'Plans', to: '/plans' }];
  if (form) {
    const { plan, campaign, initiative } = form.context;
    items.push(
      { label: plan.name, to: recordHref('plan', plan.id) },
      { label: campaign.name, to: recordHref('campaign', campaign.id, plan.id) },
      { label: initiative.name, to: recordHref('initiative', initiative.id, plan.id) },
    );
    if (activity) {
      items.push({ label: activity.name, to: activityHref(plan.id, activity.id) });
    }
  }
  items.push({ label: activity || title.startsWith('Edit') ? 'Edit' : title });
  return <PageHeader breadcrumb={items} title={title} />;
}
