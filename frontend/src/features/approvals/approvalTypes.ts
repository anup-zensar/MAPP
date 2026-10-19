import type { components } from '@/api/schema';

/**
 * What feature 006's activity page gives feature 008's approval parts about the activity (008
 * plan, "Activity page parts"): enough to choose the status action and to send the version.
 */
export interface ApprovalActivity {
  id: string;
  name: string;
  status: components['schemas']['ActivityStatus'];
  version: number;
}

/** 006's function that shows "Submitting needs <n> things — view list" and opens Ready to submit. */
export type ShowSubmitProblems = (count: number) => void;
