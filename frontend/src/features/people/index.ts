/**
 * What feature 003 offers other features' forms and pages: person and team fields that load their
 * own choices, and the people and team lists.
 *
 *   import { PersonField, TeamField, usePeople } from '@/features/people';
 */
export { PersonField } from './PersonField';
export type { PersonFieldProps } from './PersonField';
export { TeamField } from './TeamField';
export type { TeamFieldProps } from './TeamField';
export { PEOPLE_KEY, TEAMS_KEY, usePeople, useTeams } from './peopleApi';
export type { Person, PersonRef, Team, TeamRef } from './peopleApi';
