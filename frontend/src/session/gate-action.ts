import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { availabilityProps, isShown } from './access';
import type { Availability } from './access';

/**
 * A ⋯ menu item as the record's `access` allows it (feature 014, FR-011): as it is when available,
 * disabled with the server's reason as its hint when not, and left out (an empty list) when it is
 * unavailable without a reason. Spread the answer into the menu's items.
 */
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

/** Whether a header button or link for this action shows: only when the action is available. */
export function allows(availability: Availability | undefined): boolean {
  return !availabilityProps(availability).disabled;
}
