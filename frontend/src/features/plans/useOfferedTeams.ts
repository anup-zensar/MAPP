import { useAccess } from '@/session/access';

// Where the person may create or edit plans; undefined for All markets.
export function useOfferedTeams(mode: 'create' | 'edit'): string[] | undefined {
  const scope = useAccess().scopeOf(mode === 'create' ? 'plans.create' : 'plans.edit');
  return scope && !scope.all ? scope.teams.map((team) => team.id) : undefined;
}
