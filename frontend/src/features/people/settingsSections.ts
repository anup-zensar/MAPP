/**
 * Feature 003's cards on the Settings page (listed by features/settings/sections.ts). Feature 003
 * adds `ActiveCount` components that show "<n> active" from its own data.
 */
import { Users, UsersRound } from 'lucide-react';
import type { SettingsSection } from '@/features/settings/sections';

export const PEOPLE_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: 'People',
    description: 'Everyone who can sign in, with their job title and teams.',
    path: '/settings/people',
    icon: Users,
  },
  {
    title: 'Teams',
    description: 'Groups of people that plans are assigned to.',
    path: '/settings/teams',
    icon: UsersRound,
  },
];
