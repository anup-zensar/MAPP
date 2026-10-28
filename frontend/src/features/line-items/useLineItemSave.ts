import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import type { FormProblems } from '@/ui/useFormProblems';
import type { DateFormat } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import type { LineItemCreate } from './api';
import type { LineItemFormState } from './line-item-form-state';
import { formProblems, toCreateBody, withoutCleared } from './line-item-requests';
import { ACTIVITY_GONE } from './line-item-text';

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

/** A failed save that is not about one field; `gone` says what was deleted meanwhile. */
export type LineItemFailure = SaveFailure & { gone?: 'activity' | 'lineItem' };

export interface LineItemSaveAttempt {
  body: LineItemCreate;
  addAnother: boolean;
}

export interface LineItemSaveOptions {
  attributeForm: AttributeForm;
  /** The values the edit form opened with and the stored attributes; null on the create form. */
  stored: { state: LineItemFormState; attributes: LineItemFormState['attributes'] } | null;
  problems: FormProblems;
  onSave: (body: LineItemCreate, addAnother: boolean) => Promise<() => void>;
  onConflict?: () => Promise<void>;
}

/** Saving the line-item form: the request and its refusal; it never restarts approval (D-322). */
export function useLineItemSave(options: LineItemSaveOptions) {
  const { attributeForm, stored, problems, onSave } = options;
  const { dateFormat } = useFormatters();
  const [failure, setFailure] = useState<LineItemFailure | null>(null);
  const [saving, setSaving] = useState(false);
  const [afterSave, setAfterSave] = useState<(() => void) | null>(null);

  function save(state: LineItemFormState, addAnother: boolean) {
    if (saving) {
      return;
    }
    const cleared = valuesToClear(attributeForm, state.attributes, stored?.attributes ?? {});
    void send({ body: toCreateBody(state, withoutCleared(state.attributes, cleared)), addAnother });
  }

  async function send(attempt: LineItemSaveAttempt) {
    setSaving(true);
    try {
      const after = await onSave(attempt.body, attempt.addAnother);
      setAfterSave(() => after);
    } catch (error) {
      await showFailure(error instanceof ApiError ? error : new ApiError('other'), attempt);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: ApiError, attempt: LineItemSaveAttempt) {
    if (error.kind === 'validation') {
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
    /** What to do once a save went through (leave, or open a new form); null until then. */
    afterSave,
    save,
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
