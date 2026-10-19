import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Activity types (feature 002, US5) — STUB of the shared foundation. Feature 002 replaces this
 * file with the real page; the route in app/router.tsx stays.
 */
export function ActivityTypesPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Activity types"
      description="The kinds of work activities can be."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Activity types' }]}
    />
  );
}
