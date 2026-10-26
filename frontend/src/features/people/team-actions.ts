import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { Team } from './api';

/** What the list and the details panel let people do with a team. */
export interface TeamActions {
  edit: (team: Team) => void;
  setStatus: (team: Team, active: boolean) => void;
  remove: (team: Team) => void;
}

/** Set team inactive or Reactivate · Delete (last, after a line), in the row's and panel's ⋯. */
export function teamMoreItems(team: Team, actions: TeamActions): ActionsMenuItem[] {
  return [
    {
      label: team.active ? 'Set team inactive' : 'Reactivate',
      onSelect: () => actions.setStatus(team, !team.active),
    },
    {
      label: 'Delete',
      tone: 'danger',
      separatorBefore: true,
      onSelect: () => actions.remove(team),
    },
  ];
}
