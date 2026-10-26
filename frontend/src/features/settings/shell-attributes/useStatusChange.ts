import { ApiError, conflictMessage } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import type { DateFormat } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';

/** "Shell attribute" or "option": the record type as it reads mid-sentence. */
export type RecordType = 'Shell attribute' | 'option';

/** The record type at the start of a sentence: "Option 'TV' set inactive". */
function sentenceStart(recordType: RecordType): string {
  return `${recordType.charAt(0).toUpperCase()}${recordType.slice(1)}`;
}

/**
 * Why a change was refused, in plain words, or null when the session has ended (the sign-in page
 * says so): who changed the record meanwhile for a conflict (FR-007), the field problems of a
 * refusal (a move at the edge, a lock), or the API's message.
 */
export function failureText(
  error: unknown,
  recordType: RecordType,
  dateFormat?: DateFormat,
): string | null {
  const failure = error instanceof ApiError ? error : new ApiError('other');
  if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
    return null;
  }
  if (failure.kind === 'conflict') {
    return conflictMessage(failure, recordType, dateFormat);
  }
  if (failure.kind === 'validation' && failure.fields.length > 0) {
    return failure.fields.map((field) => field.message).join(' ');
  }
  return failure.message;
}

/** Shows failureText as a message. */
export function useFailureToast(recordType: RecordType) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  return (error: unknown) => {
    const message = failureText(error, recordType, dateFormat);
    if (message) {
      show({ message });
    }
  };
}

/** What a status change needs: the record's name, the new status and the call that makes it. */
export interface StatusRequest {
  name: string;
  active: boolean;
  run: () => Promise<unknown>;
}

/**
 * Set inactive and Reactivate (FR-004, plan "useSettingStatus"): runs the change and says what
 * happened — "Option 'TV' set inactive", "Shell attribute 'Region' reactivated" — or why not.
 * Answers whether it worked.
 */
export function useStatusChange(recordType: RecordType) {
  const { show } = useToast();
  const showFailure = useFailureToast(recordType);

  return async function changeStatus({ name, active, run }: StatusRequest): Promise<boolean> {
    try {
      await run();
      const what = active ? 'reactivated' : 'set inactive';
      show({ message: `${sentenceStart(recordType)} '${name}' ${what}` });
      return true;
    } catch (error) {
      showFailure(error);
      return false;
    }
  };
}
