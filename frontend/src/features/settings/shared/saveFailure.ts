import { ApiError, conflictMessage } from '@/api/errors';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import type { DateFormat } from '@/format/intl';
import { SETTING_NOUNS } from './settingsNouns';
import type { SettingKind } from './settingsNouns';

/** What to do after someone else saved first (feature 001's edit-form rule). */
export const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

/** How a failed save shows on a Settings form. */
export interface SaveFailureView {
  /** Every field problem at once (validation). */
  problems: FieldProblem[];
  /** A failure that is not about one field, shown at the top of the form. */
  failure: SaveFailure | null;
  /** Someone else saved first: the form takes the latest version and keeps what was typed. */
  conflict: boolean;
}

/**
 * The save flow's failures (feature 002 plan, "Shared Settings parts"): every problem at once
 * under the fields, the conflict message naming who changed the record and when, the "no longer
 * exists" message, or the API's plain-language message. An ended session is left to the sign-in
 * page.
 */
export function saveFailureOf(
  error: unknown,
  kind: SettingKind,
  dateFormat?: DateFormat,
): SaveFailureView {
  const apiError = error instanceof ApiError ? error : new ApiError('other');
  switch (apiError.kind) {
    case 'validation':
      return { problems: apiError.fields, failure: null, conflict: false };
    case 'conflict':
      return {
        problems: [],
        failure: {
          message: conflictMessage(apiError, SETTING_NOUNS[kind].noun, dateFormat),
          nextStep: CONFLICT_NEXT_STEP,
        },
        conflict: true,
      };
    case 'notFound':
      return { problems: [], failure: { message: SETTING_NOUNS[kind].notFound }, conflict: false };
    case 'signInRequired':
    case 'sessionExpired':
      return { problems: [], failure: null, conflict: false };
    default:
      return { problems: [], failure: { message: apiError.message }, conflict: false };
  }
}
