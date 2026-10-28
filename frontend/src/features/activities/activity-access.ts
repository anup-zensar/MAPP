import { allows } from '@/session/gate-action';
import type { Activity } from './api';

// Seeing its financial data is not a change, so it does not count.
export function changesNothingIn(access: Activity['access']): boolean {
  if (!access) {
    return false;
  }
  return Object.entries(access).every(([action, availability]) => {
    return action === 'seeFinance' || !allows(availability);
  });
}
