import { useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { ApiError, conflictMessage } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/format/useFormatters';
import { useStatusFlow } from '@/ui/useStatusFlow';
import type { StatusFlow } from '@/ui/useStatusFlow';
import { SETTING_NOUNS } from './settings-nouns';
import type { SettingKind } from './settings-nouns';

type Usage = components['schemas']['Usage'];

/** The record whose status changes. */
export interface StatusTarget {
  /** Its id in the address: MOD-1001, or the currency's code. */
  id: string;
  name: string;
  /** The version people saw when they chose the action. */
  version: number;
  /** Where it is used: a record in use asks before it is set inactive. */
  usage: Usage;
}

export interface SettingStatusOptions {
  kind: SettingKind;
  /** POST …/status with `{active, version}`; answers the record after the change. */
  request: (id: string, body: { active: boolean; version: number }) => Promise<{ version: number }>;
  /** The lists that show the record's status. */
  invalidate: QueryKey[];
}

/**
 * Set inactive and Reactivate of a Settings record (FR-004) with the one behaviour of design
 * system §6.7: in use, the question "Set model 'Lubricants 1.0' inactive?" says where it is used
 * and what changes; not in use, it applies at once. Both end with "Model 'Lubricants 1.0' set
 * inactive" and **Undo**. A failure says why — a conflict names who changed the record first.
 */
export function useSettingStatus({
  kind,
  request,
  invalidate,
}: SettingStatusOptions): StatusFlow<StatusTarget> {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const showFailure = useFailureToast();
  const { dateFormat } = useFormatters();
  const nouns = SETTING_NOUNS[kind];

  return useStatusFlow<StatusTarget>({
    words: { title: nouns.title, noun: nouns.noun },
    usage: (target) => (target.usage.inUse ? [`${target.usage.summary}.`] : []),
    effect: () => [nouns.inactiveEffect],
    request: async (target, active, version) => {
      try {
        return await request(target.id, { active, version });
      } finally {
        await Promise.all(
          invalidate.map((queryKey) => queryClient.invalidateQueries({ queryKey })),
        );
      }
    },
    onFailure: (error) => {
      if (error instanceof ApiError && error.kind === 'conflict') {
        show({ message: conflictMessage(error, nouns.noun, dateFormat) });
      } else if (error instanceof ApiError && error.kind === 'notFound') {
        show({ message: nouns.notFound });
      } else {
        showFailure(error);
      }
    },
  });
}

/**
 * Says why an action failed, in plain words: with Try again when the server could not be reached
 * or failed, without it when the API refused for good. An ended session is left to the sign-in
 * page, which says so.
 */
export function useFailureToast() {
  const { show } = useToast();

  return function showFailure(error: unknown, tryAgain?: () => void) {
    const failure = error instanceof ApiError ? error : new ApiError('other');
    if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
      return;
    }
    const mayWorkLater = failure.kind === 'unreachable' || failure.kind === 'other';
    show({
      message: failure.message,
      action:
        mayWorkLater && tryAgain
          ? { label: 'Try again', altText: 'Try again', onClick: tryAgain }
          : undefined,
    });
  };
}
