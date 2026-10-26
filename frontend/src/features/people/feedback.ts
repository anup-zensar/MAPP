/**
 * How the people and team screens tell what went wrong (plan "States"): a failed save in its form,
 * a failed status change as a message.
 */
import { ApiError, conflictMessage } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import type { DateFormat } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';

/** What the person can do after a conflict (feature 001's next step). */
const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

/**
 * The noun of the conflict message: "Not saved — Alex Morgan changed this team at 14:32"; on My
 * profile, "… changed this profile …".
 */
export type RecordNoun = 'person' | 'team' | 'profile';

/** A failed save, split as the form shows it. */
export interface FormFailure {
  /** Every field problem at once (validation, also from an impact check). */
  problems: FieldProblem[];
  /** A failure that is not about one field: conflict, not found, server unreachable. */
  failure: SaveFailure | null;
  /** Someone else saved first: the form takes the record's latest version. */
  conflict: boolean;
}

/** Splits a failed save into field problems, or one failure above the form. */
export function formFailure(error: unknown, noun: RecordNoun, dateFormat: DateFormat): FormFailure {
  const apiError = error instanceof ApiError ? error : new ApiError('other');
  if (apiError.kind === 'validation') {
    return { problems: apiError.fields, failure: null, conflict: false };
  }
  if (apiError.kind === 'conflict') {
    return {
      problems: [],
      failure: {
        message: conflictMessage(apiError, noun, dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      },
      conflict: true,
    };
  }
  return { problems: [], failure: { message: apiError.message }, conflict: false };
}

/**
 * Shows why a status change failed, as a message: the conflict sentence, the not-found message or
 * the API's message. A session that has ended is left to the sign-in page, which says so.
 */
export function useShowFailure(noun: RecordNoun) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();

  return function showFailure(error: unknown) {
    const failure = error instanceof ApiError ? error : new ApiError('other');
    if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
      return;
    }
    show({
      message:
        failure.kind === 'conflict' ? conflictMessage(failure, noun, dateFormat) : failure.message,
    });
  };
}
