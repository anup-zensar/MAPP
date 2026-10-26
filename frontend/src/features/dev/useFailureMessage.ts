import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';

/**
 * Shows why an action failed, in plain words (FR-020): with Try again when the server could not
 * be reached or failed, without it when the API refused for good ("Restore 'Ideas' first"). A
 * session that has ended is left to the sign-in page, which says so.
 */
export function useFailureMessage() {
  const { show } = useToast();

  return function showFailure(error: unknown, tryAgain: () => void) {
    const failure = error instanceof ApiError ? error : new ApiError('other');
    if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
      return;
    }
    const mayWorkLater = failure.kind === 'unreachable' || failure.kind === 'other';
    show({
      message: failure.message,
      action: mayWorkLater
        ? { label: 'Try again', altText: 'Try again', onClick: tryAgain }
        : undefined,
    });
  };
}
