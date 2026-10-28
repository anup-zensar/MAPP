/**
 * Feature 003's cards on the Settings page (listed by features/settings/sections.ts), each with
 * the count of active records from feature 003's own data.
 */
import { Users, UsersRound } from 'lucide-react';
import type { SettingsSection } from '@/features/settings';
import { ActivePeopleCount, ActiveTeamsCount } from './ActiveCounts';

export const PEOPLE_SETTINGS_SECTIONS: SettingsSection[] = [
  {
    title: 'People',
    description: 'Everyone who can sign in to SP&A One.',
    path: '/settings/people',
    icon: Users,
    ActiveCount: ActivePeopleCount,
  },
  {
    title: 'Teams',
    description: 'The teams plans belong to.',
    path: '/settings/teams',
    icon: UsersRound,
    ActiveCount: ActiveTeamsCount,
  },
];
