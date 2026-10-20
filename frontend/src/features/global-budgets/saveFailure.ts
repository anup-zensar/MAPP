import { ApiError, conflictMessage } from '@/api/errors';
import type { FieldProblem, SaveFailure } from '@/components';
import type { DateFormat } from '@/lib/format';

/** What to do after someone else saved first (feature 001's edit-form rule). */
export const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

export const BUDGET_GONE = 'This global budget no longer exists — someone may have deleted it.';

/** How a failed save shows on a form: field problems, or one failure above the fields. */
export interface SaveFailureView {
  problems: FieldProblem[];
  failure: SaveFailure | null;
  conflict: boolean;
  /** The session ended: the sign-in page says so. */
  sessionLost: boolean;
}

/**
 * Every field problem at once; a conflict naming who changed the record and when; "no longer
 * exists"; or the API's own message (a refusal such as "Allocated from …" or a 403's reason).
 */
export function saveFailureOf(
  error: unknown,
  noun: string,
  dateFormat?: DateFormat,
): SaveFailureView {
  const apiError = error instanceof ApiError ? error : new ApiError('other');
  const view = { problems: [], failure: null, conflict: false, sessionLost: false };
  switch (apiError.kind) {
    case 'validation':
      return apiError.fields.length > 0
        ? { ...view, problems: apiError.fields }
        : { ...view, failure: { message: apiError.message } };
    case 'conflict':
      return {
        ...view,
        failure: {
          message: conflictMessage(apiError, noun, dateFormat),
          nextStep: CONFLICT_NEXT_STEP,
        },
        conflict: true,
      };
    case 'signInRequired':
    case 'sessionExpired':
      return { ...view, sessionLost: true };
    default:
      return { ...view, failure: { message: apiError.message } };
  }
}
