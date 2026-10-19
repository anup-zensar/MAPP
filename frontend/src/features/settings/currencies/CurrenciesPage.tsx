import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Currencies (feature 002, US4) — STUB of the shared foundation. Feature 002 replaces this file
 * with the real page; the route in app/router.tsx stays.
 */
export function CurrenciesPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Currencies"
      description="Currencies for line items and display, with their rate per 1 USD."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Currencies' }]}
    />
  );
}
