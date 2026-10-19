import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Models (feature 002, US1) — STUB of the shared foundation. Feature 002 replaces this file with
 * the real page; the route in app/router.tsx stays.
 */
export function ModelsPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Models"
      description="The rulebooks plans use. Changes apply to every plan using the model."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Models' }]}
    />
  );
}
