import { Plus } from 'lucide-react';
import { useId } from 'react';
import { useAccess } from '@/session/access';
import { Button, LinkButton } from '@/ui/Button/Button';
import { LockedNote } from '@/ui/LockedNote/LockedNote';

const NO_TEAM_FOR_PLANS =
  'Plans belong to teams, and you are not in a team yet. Ask an administrator to add you to one.';

/** **New plan** for people who may create plans; locked, saying why, while they are in no team. */
export function NewPlanButton() {
  const { has, scopeOf } = useAccess();
  const hintId = useId();
  const scope = scopeOf('plans.create');
  if (!has('plans.create') || !scope) {
    return null;
  }
  if (!scope.all && scope.teams.length === 0) {
    return (
      <>
        <LockedNote id={hintId}>{NO_TEAM_FOR_PLANS}</LockedNote>
        <Button icon={<Plus />} disabled aria-describedby={hintId}>
          New plan
        </Button>
      </>
    );
  }
  return (
    <LinkButton to="/plans/new" variant="primary" icon={<Plus />}>
      New plan
    </LinkButton>
  );
}
