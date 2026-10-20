/**
 * Feature 014's card on the Settings page (listed by features/settings/sections.ts, after the
 * existing ones — consolidation D-305), with the number of roles from its own data.
 */
import { ShieldCheck } from 'lucide-react';
import type { SettingsSection } from '@/features/settings/sections';
import { ACCESS_PATH } from './accessPaths';
import { RolesCount } from './RolesCount';

export const ACCESS_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: 'Roles and access',
    description: 'Who can see and change what, by role and market.',
    path: ACCESS_PATH,
    icon: ShieldCheck,
    ActiveCount: RolesCount,
  },
];
