/**
 * One word per status colour (design system §4.1, §11 rule 6; functional review A-02, UI audit
 * X-14): "In progress" means only a running activity (blue). The approval roll-up of plans,
 * campaigns and initiatives — which the API still calls "In progress", in amber — is written
 * "Partly approved", so the same word never shows in two colours.
 */
import type { AllWorkStatus } from './all-work-types';

type AllWorkTone = AllWorkStatus['tone'];

/** The roll-up's word: approval has started for some activities below, not for all. */
export const PARTLY_APPROVED = 'Partly approved';

const IN_PROGRESS = 'In progress';

/** The word shown for a status chip of All work. */
export function statusWord(label: string, tone: AllWorkTone | undefined): string {
  return label === IN_PROGRESS && tone !== undefined && tone !== 'active' ? PARTLY_APPROVED : label;
}
