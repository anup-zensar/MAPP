import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * People (feature 003, US1), at /settings/people/:personId? (with an id the details panel is
 * open) — STUB of the shared foundation. Feature 003 replaces this file with the real page; the
 * route in app/router.tsx stays.
 */
export function PeoplePage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="People"
      description="Everyone who can sign in to NorthStar, with their job title and teams."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'People' }]}
    />
  );
}
