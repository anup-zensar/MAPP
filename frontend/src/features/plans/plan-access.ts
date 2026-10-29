import { allows } from '@/session/gate-action';
import type { Plan } from './api';

type PlanAccess = NonNullable<Plan['access']>;

// Copying is left out: a copy is a new plan, which changes nothing in this one.
export function changesNothingIn(access: PlanAccess | undefined): boolean {
  if (!access) {
    return false;
  }
  const actions = [access.edit, access.publish, access.delete, ...Object.values(access.content)];
  return actions.every((availability) => !allows(availability));
}
