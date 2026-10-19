import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import type { FieldProblem, SaveFailure } from '@/components';

interface Leaving {
  to: string;
  state?: unknown;
}

/**
 * Leaves a form after a save. The form first stops counting as changed, then navigates, so the
 * "Discard changes?" question (useUnsavedChanges) never stops a saved form from closing.
 */
export function useLeaveAfterSave() {
  const navigate = useNavigate();
  const [target, setTarget] = useState<Leaving | null>(null);

  useEffect(() => {
    if (target) {
      void navigate(target.to, { state: target.state });
    }
  }, [target, navigate]);

  return {
    /** A save is done and the form is closing: it no longer counts as changed. */
    leaving: target !== null,
    leave: (to: string, state?: unknown) => setTarget({ to, state }),
  };
}

/** What a failed save shows: every field problem at once, or one failure above the form. */
export function failureOf(error: unknown): {
  problems: FieldProblem[];
  failure: SaveFailure | null;
} {
  const apiError = error instanceof ApiError ? error : new ApiError('other');
  if (apiError.kind === 'validation') {
    return { problems: apiError.fields, failure: null };
  }
  return { problems: [], failure: { message: apiError.message } };
}
