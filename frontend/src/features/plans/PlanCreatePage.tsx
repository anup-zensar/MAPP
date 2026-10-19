import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * New plan (feature 004, US1): the plan form — STUB of the shared foundation. Feature 004
 * replaces this file with the real page; the route stays.
 */
export function PlanCreatePage() {
  return (
    <BuildingPage
      eyebrow="Plans"
      title="New plan"
      description="A name, a model and a business unit are enough to start — everything else can follow."
    />
  );
}
