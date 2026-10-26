import type { QueryClient } from '@tanstack/react-query';
import { ApiError } from '@/api/errors';
import type { DateFormat } from '@/format/intl';
import { alreadyResolvedText } from './bin-text';

/** The failure as an ApiError (call() throws nothing else that reaches the screen). */
export function asApiError(error: unknown): ApiError {
  return error instanceof ApiError ? error : new ApiError('other');
}

/**
 * The failures that end a recycle-bin dialog (plan "Recycle bin page"): someone else already
 * restored or deleted the entry (FR-032), or its parent is missing after all (FR-023, FR-024).
 * They become a message, and the bin is asked again; the text to show, or undefined for any other
 * failure, which the dialog shows itself.
 */
export function closingMessage(
  error: ApiError,
  name: string,
  dateFormat: DateFormat,
): string | undefined {
  switch (error.kind) {
    case 'alreadyResolved':
      return alreadyResolvedText(error, name, dateFormat);
    case 'restoreParentFirst':
    case 'parentDeletedPermanently':
      return error.message;
    default:
      return undefined;
  }
}

/** Asks for the recycle bin again, so a row someone else resolved disappears. */
export function refetchBin(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: ['recycle-bin'], exact: true });
}
