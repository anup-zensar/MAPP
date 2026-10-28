import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { availabilityProps, isShown } from './access';
import type { Availability } from './access';

// A menu item as the record's access allows it: as it is, disabled with the reason, or left out.
export function gateItem(
  item: ActionsMenuItem,
  availability: Availability | undefined,
): ActionsMenuItem[] {
  if (!isShown(availability)) {
    return [];
  }
  const { disabled, hint } = availabilityProps(availability);
  return disabled ? [{ ...item, disabled: true, hint }] : [item];
}

export function allows(availability: Availability | undefined): boolean {
  return !availabilityProps(availability).disabled;
}

// A header button's action, listed in More (disabled, with its reason) while the button is not shown.
export function moreItemFor(
  item: ActionsMenuItem,
  availability: Availability | undefined,
): ActionsMenuItem[] {
  return allows(availability) ? [] : gateItem(item, availability);
}
