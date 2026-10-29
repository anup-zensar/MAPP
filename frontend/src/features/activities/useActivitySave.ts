import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { overLimitMessages } from '@/features/overspend';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import type { AttributeForm, ClearedValue } from '@/ui/ShellAttributeFields/rules';
import { useToast } from '@/ui/Toast/useToast';
import type { FormProblems } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { fetchLatestActivity, useCreateActivity, useUpdateActivity } from './api';
import type { Activity, ActivityFormData } from './api';
import { reapprovalChanges, toCreateBody, toUpdateBody } from './activity-form-state';
import type { FormState, ReapprovalChange } from './activity-form-state';
import { activityHref } from './activity-text';

export interface ActivitySaveOptions {
  data: ActivityFormData;
  activity?: Activity;
  initial: FormState;
  attributeForm: AttributeForm | undefined;
  planCurrency: string;
  problems: FormProblems;
}

interface Question<T> {
  about: T;
  values: FormState;
  addAnother: boolean;
}

/**
 * Saving the activity form: values it would clear and changes that restart approval are asked
 * about first; after a conflict the typed values stay with the newer version, so Save replaces it.
 */
export function useActivitySave(options: ActivitySaveOptions) {
  const { data, activity, initial, attributeForm, planCurrency, problems } = options;
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { date, dateFormat } = useFormatters();
  const create = useCreateActivity();
  const update = useUpdateActivity();
  const [version, setVersion] = useState(activity?.version ?? 0);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState<Question<ClearedValue[]> | null>(null);
  const [reapproval, setReapproval] = useState<Question<ReapprovalChange[]> | null>(null);
  // Where to go once the activity is saved: the typed values are no longer unsaved then.
  const [leaveTo, setLeaveTo] = useState<string | null>(null);

  function save(values: FormState, addAnother: boolean) {
    if (saving) {
      return;
    }
    const overLimits = overLimitMessages(
      data.overspendLimits,
      activity?.plannedCost?.amount ?? null,
      values.plannedCost,
    );
    if (overLimits.length > 0) {
      setFailure({ message: overLimits.join(' ') });
      return;
    }
    setFailure(null);
    const cleared = attributeForm
      ? valuesToClear(attributeForm, values.attributes, activity?.attributes ?? {})
      : [];
    if (cleared.length > 0) {
      setClearing({ about: cleared, values, addAnother });
      return;
    }
    void confirmApproval(values, addAnother);
  }

  async function confirmApproval(values: FormState, addAnother: boolean) {
    const underApproval = activity?.status === 'submitted' || activity?.status === 'approved';
    const changes = underApproval ? reapprovalChanges(initial, values, date, planCurrency) : [];
    if (changes.length > 0) {
      setReapproval({ about: changes, values, addAnother });
      return;
    }
    await send(values, addAnother, false);
  }

  async function send(values: FormState, addAnother: boolean, restartApproval: boolean) {
    setSaving(true);
    try {
      setLeaveTo(await saved(values, addAnother, restartApproval));
      setReapproval(null);
    } catch (error) {
      setReapproval(null);
      await showFailure(error instanceof ApiError ? error : new ApiError('other'));
    } finally {
      setSaving(false);
    }
  }

  async function saved(values: FormState, addAnother: boolean, restartApproval: boolean) {
    if (activity) {
      const body = toUpdateBody(values, version, restartApproval, planCurrency);
      const result = await update.mutateAsync({ id: activity.id, body });
      show({ message: `Activity '${result.name}' saved` });
      return activityHref(data.context.plan.id, result.id);
    }
    const body = toCreateBody(values, planCurrency);
    const created = await create.mutateAsync({ initiativeId: data.context.initiative.id, body });
    const message = `Activity '${created.name}' created`;
    const href = activityHref(data.context.plan.id, created.id);
    if (!addAnother) {
      show({ message });
      return href;
    }
    show({
      message,
      action: {
        label: 'View',
        altText: 'Open it from the plan',
        onClick: () => void navigate(href),
      },
    });
    return location.pathname;
  }

  async function showFailure(error: ApiError) {
    if (error.kind === 'validation') {
      problems.showServerProblems(error.fields);
      return;
    }
    if (error.kind === 'conflict' && activity) {
      // When the activity cannot be read now, the next Save meets the conflict again.
      const latest = await fetchLatestActivity(queryClient, activity.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({ message: conflictMessage(error, 'activity', dateFormat) });
      return;
    }
    setFailure({ message: error.message });
  }

  return {
    failure,
    saving,
    clearing,
    reapproval,
    leaveTo,
    save,
    confirmClearing: () => {
      setClearing(null);
      if (clearing) {
        void confirmApproval(clearing.values, clearing.addAnother);
      }
    },
    cancelClearing: () => setClearing(null),
    clearFailure: () => setFailure(null),
    confirmReapproval: () => {
      if (reapproval) {
        void send(reapproval.values, false, true);
      }
    },
    cancelReapproval: () => setReapproval(null),
  };
}
