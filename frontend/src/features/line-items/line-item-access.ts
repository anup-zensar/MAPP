import { allows } from '@/session/gate-action';
import type { LineItemDetails } from './api';

// Seeing its financial data is not a change, so it does not count.
export function changesNothingIn(access: LineItemDetails['access']): boolean {
  if (!access) {
    return false;
  }
  return Object.entries(access).every(([action, availability]) => {
    return action === 'seeFinance' || !allows(availability);
  });
}
