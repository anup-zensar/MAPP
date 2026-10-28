import { useState } from 'react';
import { ApiError, conflictMessage, isSessionLost } from '@/api/errors';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { useFormatters } from '@/format/useFormatters';
import type { PurchaseRequest, PurchaseRequestInput } from './api';
import { notReadyLines } from './form-problems';
import { CONFLICT_NEXT_STEP, NOT_READY_TITLE } from './form-text';

/** A refused save that is not about one field. */
export interface SaveFailure {
  message: string;
  nextStep?: string;
  /** Every gate problem of a refused submission (409 not_ready), at once. */
  details: string[];
  /** Someone else saved first: offer to load their version, keeping the entries. */
  conflict: boolean;
}

export interface SaveOptions {
  onSave: (input: PurchaseRequestInput) => Promise<PurchaseRequest>;
  setProblems: (problems: FieldProblem[]) => void;
}

/** Sends the form and turns a refusal into field problems or one failure (032 FR-016, FR-053). */
export function usePurchaseRequestSave({ onSave, setProblems }: SaveOptions) {
  const { dateFormat } = useFormatters();
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState<PurchaseRequest | null>(null);

  async function save(input: PurchaseRequestInput) {
    if (saving) {
      return;
    }
    setSaving(true);
    setFailure(null);
    try {
      setSaved(await onSave(input));
    } catch (error) {
      showFailure(error instanceof ApiError ? error : new ApiError('other'));
    } finally {
      setSaving(false);
    }
  }

  function showFailure(error: ApiError) {
    if (isSessionLost(error)) {
      return;
    }
    if (error.kind === 'validation' && error.fields.length > 0) {
      setProblems(error.fields);
      return;
    }
    setProblems([]);
    if (error.kind === 'notReady') {
      setFailure({ message: NOT_READY_TITLE, details: notReadyLines(error), conflict: false });
      return;
    }
    const conflict = error.kind === 'conflict';
    setFailure({
      message: conflict ? conflictMessage(error, 'purchase request', dateFormat) : error.message,
      nextStep: conflict ? CONFLICT_NEXT_STEP : undefined,
      details: [],
      conflict,
    });
  }

  return { failure, saving, saved, save };
}
