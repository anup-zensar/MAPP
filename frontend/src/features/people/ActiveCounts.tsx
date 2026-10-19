import { usePeople, useTeams } from './peopleApi';

/** "6 active" on the People card of the Settings page; nothing until the people have arrived. */
export function ActivePeopleCount() {
  const people = usePeople();
  return people.data ? activeText(people.data.items) : null;
}

/** "3 active" on the Teams card of the Settings page; nothing until the teams have arrived. */
export function ActiveTeamsCount() {
  const teams = useTeams();
  return teams.data ? activeText(teams.data.items) : null;
}

function activeText(records: { active: boolean }[]): string {
  return `${records.filter((record) => record.active).length} active`;
}
