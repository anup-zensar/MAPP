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
