import { APPROVAL_WORKFLOW_SETTINGS_SECTIONS } from '@/features/approval-workflows';
import { ACCESS_SETTINGS_SECTIONS, PEOPLE_SETTINGS_SECTIONS } from '@/features/people';
import { FIELD_HELP_SETTINGS_SECTIONS, SETTINGS_OWN_SECTIONS } from '@/features/settings';
import type { SettingsSection } from '@/features/settings';

/**
 * Every card of the Settings page, in order: 002's five and Field help (016), then People and
 * Teams (003), Approval workflows (008) and Roles and access (014).
 */
export const SETTINGS_SECTIONS: SettingsSection[] = [
  ...SETTINGS_OWN_SECTIONS,
  ...FIELD_HELP_SETTINGS_SECTIONS,
  ...PEOPLE_SETTINGS_SECTIONS,
  ...APPROVAL_WORKFLOW_SETTINGS_SECTIONS,
  ...ACCESS_SETTINGS_SECTIONS,
];
