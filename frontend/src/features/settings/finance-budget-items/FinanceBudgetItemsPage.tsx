import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Finance budget items (feature 002, US5) — STUB of the shared foundation. Feature 002 replaces
 * this file with the real page; the route in app/router.tsx stays.
 */
export function FinanceBudgetItemsPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Finance budget items"
      description="The finance budget items line items link to. A simple list until the Finance module arrives."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Finance budget items' }]}
    />
  );
}
