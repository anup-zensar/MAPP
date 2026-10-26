import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { Person, PersonDetails } from './api';
import { DEACTIVATE_TIP, SELF_DEACTIVATION } from './texts';

/** What the list and the details panel let people do with a person. */
export interface PersonActions {
  edit: (person: Person | PersonDetails) => void;
  deactivate: (person: Person | PersonDetails) => void;
  reactivate: (person: Person | PersonDetails) => void;
}

/**
 * Deactivate or Reactivate, after a line, in the row's ⋯ and the details panel's ⋯. There is no
 * Delete: Deactivate sits in its place with the reason (FR-008); for yourself it says why you
 * can't (FR-006).
 */
export function personStatusItem(
  person: Person | PersonDetails,
  isYou: boolean,
  actions: PersonActions,
): ActionsMenuItem {
  return person.active
    ? {
        label: 'Deactivate',
        hint: isYou ? SELF_DEACTIVATION : DEACTIVATE_TIP,
        separatorBefore: true,
        onSelect: () => actions.deactivate(person),
      }
    : { label: 'Reactivate', separatorBefore: true, onSelect: () => actions.reactivate(person) };
}
