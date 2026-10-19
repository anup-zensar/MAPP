import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Shell attributes (feature 002, US2): the attributes of each level, by section — STUB of the
 * shared foundation. Feature 002 replaces this file with the real page; the route stays.
 */
export function ShellAttributesPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Shell attributes"
      description="Shell's own fields on each level's form. Changes apply to every form straight away."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Shell attributes' }]}
    />
  );
}
