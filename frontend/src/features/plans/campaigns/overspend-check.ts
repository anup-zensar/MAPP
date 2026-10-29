import { budgetCheck } from '@/features/overspend';
import type { Campaign, Initiative } from './api';
import type { RecordFormValues } from './schemas';

/** What the API would refuse for the typed budget (037 FR-015), before sending it; null when nothing. */
export function overspendRefusal(
  typed: RecordFormValues,
  opened: Campaign | Initiative,
): string | null {
  return budgetCheck({
    mode: typed.overspend,
    typed: typed.budget,
    stored: opened.budget?.amount ?? null,
    planned: opened.budgetAndCost.plannedCost,
  }).error;
}
