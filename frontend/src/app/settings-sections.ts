import type { SettingsSection } from '@/features/settings';

/** Every Settings card, in order; loaded with the Settings page, off the first page. */
export async function loadSettingsSections(): Promise<SettingsSection[]> {
  const [settings, people, approvalWorkflows, purchaseRequests, overspend] = await Promise.all([
    import('@/features/settings'),
    import('@/features/people'),
    import('@/features/approval-workflows'),
    import('@/features/purchase-requests'),
    import('@/features/overspend'),
  ]);
  return [
    ...settings.SETTINGS_OWN_SECTIONS,
    ...settings.FIELD_HELP_SETTINGS_SECTIONS,
    ...people.PEOPLE_SETTINGS_SECTIONS,
    ...approvalWorkflows.APPROVAL_WORKFLOW_SETTINGS_SECTIONS,
    ...purchaseRequests.PURCHASE_REQUEST_SETTINGS_SECTIONS,
    ...overspend.OVERSPEND_SETTINGS_SECTIONS,
    ...people.ACCESS_SETTINGS_SECTIONS,
  ];
}
