/**
 * The texts of Settings › Approval workflows built from counts (008 plan, "User-visible
 * messages"): who uses a workflow, what editing or setting it inactive changes.
 */
import { activitiesCount } from '@/features/approvals/messages';

/** "This approval workflow no longer exists — someone may have deleted it." (API and web). */
export const WORKFLOW_GONE =
  'This approval workflow no longer exists — someone may have deleted it.';

/** "Used by 3 activities (1 waiting for approval)", or "Not used yet". */
export function usedByText(usedByCount: number, waitingCount: number): string {
  if (usedByCount === 0) {
    return 'Not used yet';
  }
  const waiting = waitingCount > 0 ? ` (${waitingCount} waiting for approval)` : '';
  return `Used by ${activitiesCount(usedByCount)}${waiting}`;
}

/** The note of the edit form while activities wait with the workflow's levels (FR-006). */
export function openRoundsNote(waitingCount: number): string {
  const keep =
    waitingCount === 1
      ? '1 activity waiting for approval keeps the levels it was submitted with.'
      : `${waitingCount} activities waiting for approval keep the levels they were submitted with.`;
  return `Changes apply to activities submitted from now on. ${keep}`;
}

/** What setting a workflow inactive changes, sentence by sentence (FR-007). */
export function setInactiveEffects(usedByCount: number, waitingCount: number): string[] {
  const effects = ['It will no longer be offered on activities.'];
  if (usedByCount === 1) {
    effects.push("The 1 activity that uses it keeps it, shown as '(inactive)'.");
  } else if (usedByCount > 1) {
    effects.push(`The ${usedByCount} activities that use it keep it, shown as '(inactive)'.`);
  }
  if (waitingCount === 1) {
    effects.push('1 activity waiting for approval finishes with the levels it was submitted with.');
  } else if (waitingCount > 1) {
    effects.push(
      `${waitingCount} activities waiting for approval finish with the levels they were submitted with.`,
    );
  }
  return effects;
}

/** "Used by 2 activities — set it inactive instead." (also the API's in_use message). */
export function cantDeleteText(usedByCount: number): string {
  return `Used by ${activitiesCount(usedByCount)} — set it inactive instead.`;
}
