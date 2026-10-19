import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { ApiError, conflictMessage } from '@/api/errors';
import { useToast } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { SETTING_NOUNS } from './settingsNouns';
import type { SettingKind } from './settingsNouns';

/** The record whose status changes. */
export interface StatusTarget {
  /** Its id in the address: MOD-1001, or the currency's code. */
  id: string;
  name: string;
  /** The version people saw when they chose the action. */
  version: number;
}

export interface SettingStatusOptions {
  kind: SettingKind;
  /** POST …/status with `{active, version}`. */
  request: (id: string, body: { active: boolean; version: number }) => Promise<unknown>;
  /** The lists that show the record's status. */
  invalidate: QueryKey[];
}

/**
 * Set inactive and Reactivate of a Settings record (FR-004): the request, the message "Model
 * 'Lubricants 1.0' set inactive" or "… reactivated", and when it fails the reason — the conflict
 * message naming who changed the record first. Answers whether the change was made.
 */
export function useSettingStatus({ kind, request, invalidate }: SettingStatusOptions) {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const showFailure = useFailureToast();
  const { dateFormat } = useFormatters();
  const nouns = SETTING_NOUNS[kind];
  const mutation = useMutation({
    mutationFn: ({ id, active, version }: { id: string; active: boolean; version: number }) =>
      request(id, { active, version }),
    onSettled: () =>
      Promise.all(invalidate.map((queryKey) => queryClient.invalidateQueries({ queryKey }))),
  });

  return async function changeStatus(target: StatusTarget, active: boolean): Promise<boolean> {
    try {
      await mutation.mutateAsync({ id: target.id, active, version: target.version });
      show({
        message: `${nouns.title} '${target.name}' ${active ? 'reactivated' : 'set inactive'}`,
      });
      return true;
    } catch (error) {
      if (error instanceof ApiError && error.kind === 'conflict') {
        show({ message: conflictMessage(error, nouns.noun, dateFormat) });
      } else if (error instanceof ApiError && error.kind === 'notFound') {
        show({ message: nouns.notFound });
      } else {
        showFailure(error, () => void changeStatus(target, active));
      }
      return false;
    }
  };
}

/**
 * Says why an action failed, in plain words: with Try again when the server could not be reached
 * or failed, without it when the API refused for good. An ended session is left to the sign-in
 * page, which says so.
 */
export function useFailureToast() {
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
