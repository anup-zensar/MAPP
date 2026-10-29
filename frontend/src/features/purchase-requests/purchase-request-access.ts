import { allows } from '@/session/gate-action';
import type { PurchaseRequest } from './api';

/** Whether the person may change nothing on the purchase request: the View only note (038 FR-025). */
export function changesNothingIn(pr: PurchaseRequest): boolean {
  return (
    Object.values(pr.access).every((availability) => !allows(availability)) &&
    pr.lines.every((line) => !allows(line.access.remove))
  );
}
