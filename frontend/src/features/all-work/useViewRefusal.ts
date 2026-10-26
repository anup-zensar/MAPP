import { useCallback } from 'react';
import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { VIEW_GONE } from './all-work-text';
import { problemsOf } from './view-names';

/**
 * What a refused saved-view change shows (plan "States"): the field problems for the form; a view
 * deleted meanwhile, or any other failure, as a message. Returns the field problems (or none).
 */
export function useViewRefusal(): (error: unknown, nameField: string) => FieldProblem[] {
  const { show } = useToast();
  return useCallback(
    (error: unknown, nameField: string) => {
      const problems = problemsOf(error, nameField);
      if (problems) {
        return problems;
      }
      const failure = error instanceof ApiError ? error : new ApiError('other');
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        show({ message: failure.kind === 'notFound' ? VIEW_GONE : failure.message });
      }
      return [];
    },
    [show],
  );
}
