import { APPROVAL_WORKFLOW_SETTINGS_SECTIONS } from '@/features/approval-workflows/settingsSections';
import { ACCESS_SETTINGS_SECTIONS } from '@/features/people/access/settingsSections';
import { PEOPLE_SETTINGS_SECTIONS } from '@/features/people/settingsSections';
import { FIELD_HELP_SETTINGS_SECTIONS } from '@/features/settings/field-help/settingsSections';
import { SETTINGS_OWN_SECTIONS } from '@/features/settings/sections';
import type { SettingsSection } from '@/features/settings/sections';

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
