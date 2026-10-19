import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Teams (feature 003, US2), at /settings/teams/:teamId? (with an id the details panel is open) —
 * STUB of the shared foundation. Feature 003 replaces this file with the real page; the route in
 * app/router.tsx stays.
 */
export function TeamsPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Teams"
      description="Groups of people that plans are assigned to. A plan's owner must belong to one of its teams."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Teams' }]}
    />
  );
}
