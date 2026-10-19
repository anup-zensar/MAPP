import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Settings › Approval workflows (feature 008, US4), at /settings/approval-workflows/:workflowId?
 * (with an id the details panel is open) — STUB of the shared foundation. Feature 008 replaces
 * this file; the route stays.
 */
export function ApprovalWorkflowsPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Approval workflows"
      description="Who approves activities, level by level."
      breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Approval workflows' }]}
    />
  );
}
