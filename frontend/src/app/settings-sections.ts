import type { SettingsSection } from '@/features/settings';

/**
 * Every card of the Settings page, in order: 002's five and Field help (016), then People and
 * Teams (003), Approval workflows (008) and Roles and access (014). Loaded with the Settings page,
 * so these features' code stays out of the first page.
 */
export async function loadSettingsSections(): Promise<SettingsSection[]> {
  const [settings, people, approvalWorkflows] = await Promise.all([
    import('@/features/settings'),
    import('@/features/people'),
    import('@/features/approval-workflows'),
  ]);
  return [
    ...settings.SETTINGS_OWN_SECTIONS,
    ...settings.FIELD_HELP_SETTINGS_SECTIONS,
    ...people.PEOPLE_SETTINGS_SECTIONS,
    ...approvalWorkflows.APPROVAL_WORKFLOW_SETTINGS_SECTIONS,
    ...people.ACCESS_SETTINGS_SECTIONS,
  ];
}
