import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * My profile (feature 003, US4), from the account menu — STUB of the shared foundation. Feature
 * 003 replaces this file with the real page; the route in app/router.tsx stays.
 */
export function ProfilePage() {
  return (
    <BuildingPage
      eyebrow="My profile"
      title="Your profile"
      tabTitle="My profile"
      description="Your details and teams, and how NorthStar shows dates and money to you."
    />
  );
}
