import { SettingsCount } from '@/features/settings/shared/SettingsCount';
import { usePeople, useTeams } from './peopleApi';

/** "6 active" on the People card of the Settings page, as every Settings card shows its count. */
export function ActivePeopleCount() {
  const people = usePeople();
  return <SettingsCount loading={people.isPending} count={activeOf(people.data?.items)} />;
}

/** "3 active" on the Teams card of the Settings page, as every Settings card shows its count. */
export function ActiveTeamsCount() {
  const teams = useTeams();
  return <SettingsCount loading={teams.isPending} count={activeOf(teams.data?.items)} />;
}

function activeOf(records: { active: boolean }[] | undefined): number | undefined {
  return records?.filter((record) => record.active).length;
}
