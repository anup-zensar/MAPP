import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import type { FormProblems } from '@/ui/useFormProblems';
import type { DateFormat } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import type { LineItemActivity, LineItemCreate } from './api';
import type { LineItemFormState } from './line-item-form-state';
import {
  formProblems,
  isRestartRefusal,
  needsRestartConfirmation,
  restartReasons,
  toCreateBody,
  withoutCleared,
} from './line-item-requests';
import { ACTIVITY_GONE } from './line-item-text';

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

/** A failed save that is not about one field; `gone` says what was deleted meanwhile. */
export type LineItemFailure = SaveFailure & { gone?: 'activity' | 'lineItem' };

export interface LineItemSaveAttempt {
  body: LineItemCreate;
  addAnother: boolean;
  reasons: string[];
}

export interface LineItemSaveOptions {
  activity: LineItemActivity;
  attributeForm: AttributeForm;
  /** The values the edit form opened with and the stored attributes; null on the create form. */
  stored: { state: LineItemFormState; attributes: LineItemFormState['attributes'] } | null;
  problems: FormProblems;
  onSave: (body: LineItemCreate, addAnother: boolean) => Promise<() => void>;
  onConflict?: () => Promise<void>;
}

/** Saving the line-item form: the restart question first (FR-031), then the request and its refusal. */
export function useLineItemSave(options: LineItemSaveOptions) {
  const { activity, attributeForm, stored, problems, onSave } = options;
  const { dateFormat } = useFormatters();
  const [failure, setFailure] = useState<LineItemFailure | null>(null);
  const [saving, setSaving] = useState(false);
  const [pending, setPending] = useState<LineItemSaveAttempt | null>(null);
  const [afterSave, setAfterSave] = useState<(() => void) | null>(null);

  function save(state: LineItemFormState, addAnother: boolean) {
    if (saving) {
      return;
    }
    const cleared = valuesToClear(attributeForm, state.attributes, stored?.attributes ?? {});
    const attempt = {
      body: toCreateBody(state, withoutCleared(state.attributes, cleared)),
      addAnother,
      reasons: restartReasons(stored?.state ?? null, state),
    };
    if (needsRestartConfirmation(activity, stored?.state ?? null, state)) {
      setPending(attempt);
      return;
    }
    void send(attempt);
  }

  async function send(attempt: LineItemSaveAttempt) {
    setSaving(true);
    try {
      const after = await onSave(attempt.body, attempt.addAnother);
      setPending(null);
      setAfterSave(() => after);
    } catch (error) {
      await showFailure(error instanceof ApiError ? error : new ApiError('other'), attempt);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: ApiError, attempt: LineItemSaveAttempt) {
    // Someone submitted the activity meanwhile: the person confirms first (D-146).
    const restart = isRestartRefusal(error);
    setPending(restart ? attempt : null);
    if (restart || error.kind === 'validation') {
      problems.showServerProblems(formProblems(error, attempt.body));
      setFailure(null);
      return;
    }
    problems.clearProblems();
    if (error.kind === 'conflict') {
      await options.onConflict?.().catch(() => undefined);
    }
    setFailure(failureOf(error, dateFormat));
  }

  return {
    failure,
    saving,
    pending,
    /** What to do once a save went through (leave, or open a new form); null until then. */
    afterSave,
    save,
    confirmRestart: () => {
      if (pending) {
        void send({ ...pending, body: { ...pending.body, restartApproval: true } });
      }
    },
    cancelRestart: () => setPending(null),
  };
}

function failureOf(error: ApiError, dateFormat: DateFormat): LineItemFailure {
  if (error.kind === 'conflict') {
    return {
      message: conflictMessage(error, 'line item', dateFormat),
      nextStep: CONFLICT_NEXT_STEP,
    };
  }
  if (error.kind === 'notFound') {
    return {
      message: error.message,
      gone: error.message === ACTIVITY_GONE ? 'activity' : 'lineItem',
    };
  }
  return { message: error.message };
}
