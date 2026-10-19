import { useParams } from 'react-router';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * One Shell attribute (feature 002, US2, US3, FR-020): its settings, options and preview — STUB
 * of the shared foundation. Feature 002 replaces this file with the real page (tab title
 * "<label> · Shell attributes"); the route stays.
 */
export function ShellAttributePage() {
  const { attributeId = '' } = useParams();
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Shell attribute"
      description={attributeId}
      breadcrumb={[
        { label: 'Settings', to: '/settings' },
        { label: 'Shell attributes', to: '/settings/shell-attributes' },
        { label: attributeId },
      ]}
    />
  );
}
