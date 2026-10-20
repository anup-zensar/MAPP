/**
 * Feature 016's card on the Settings page (listed by ../sections.ts after feature 002's own
 * cards): Field help, with the number of edited texts.
 */
import { CircleHelp } from 'lucide-react';
import type { SettingsSection } from '../sections';
import { FieldHelpCount } from './FieldHelpCount';

/** Settings › Field help (016 FR-045). */
export const FIELD_HELP_PATH = '/settings/field-help';

export const FIELD_HELP_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: 'Field help',
    description: 'What each built-in field means, shown as ⓘ tips.',
    path: FIELD_HELP_PATH,
    icon: CircleHelp,
    ActiveCount: FieldHelpCount,
  },
];
