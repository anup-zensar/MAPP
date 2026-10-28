import { useQueryClient } from '@tanstack/react-query';
import { useRef, useState } from 'react';
import { useParams, useSearchParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { allows } from '@/session/gate-action';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { useToast } from '@/ui/Toast/useToast';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import { usePageTitle } from '@/ui/usePageTitle';
import { ChangeEffectsDialog } from './dialogs/ChangeEffectsDialog';
import { PlanForm } from './PlanForm';
import type { PlanSaveAnswer } from './PlanForm';
import { PlanFormSkeleton } from './PlanFormSkeleton';
import { answers016, toUpdateBody, valuesFromPlan } from './plan-form-values';
import { PlanNotFound } from './PlanNotFound';
import { fetchLatestPlan, useChangeEffects, usePlan, useUpdatePlan } from './api';
import type { Plan, PlanChangeEffects } from './api';
import { focusOf } from './schemas';
import type { PlanFormValues } from './schemas';

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

// The plan form with the year locked; ?focus=<field> focuses a field. A new model or business unit
// is confirmed first; a save after someone else's is refused with the typed values kept.
export function PlanEditPage() {
  const { planId = '' } = useParams();
  const plan = usePlan(planId);
  usePageTitle(plan.data ? `Edit ${plan.data.name}` : 'Edit plan');

  if (plan.isError && plan.error instanceof ApiError && plan.error.kind === 'notFound') {
    return <PlanNotFound />;
  }
  const edit = plan.data?.access?.edit;
  if (!allows(edit)) {
    return <NoAccess message={edit?.reason || undefined} />;
  }

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Plans', to: '/plans' },
          { label: plan.data?.name ?? planId, to: `/plans/${encodeURIComponent(planId)}` },
          { label: 'Edit' },
        ]}
        title="Edit plan"
      />
      {plan.isError ? (
        <ErrorState error={plan.error} onRetry={() => void plan.refetch()} />
      ) : plan.isPending ? (
        <PlanFormSkeleton />
      ) : (
        <EditForm plan={plan.data} />
      )}
    </>
  );
}

function EditForm({ plan: loaded }: { plan: Plan }) {
  const [plan] = useState(loaded);
  const initial = valuesFromPlan(plan);
  const [search] = useSearchParams();
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const updatePlan = useUpdatePlan(plan.id);
  const changeEffects = useChangeEffects(plan.id);
  const [version, setVersion] = useState(plan.version);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [effects, setEffects] = useState<{ data: PlanChangeEffects; modelOnly: boolean } | null>(
    null,
  );
  // The answer to the consequences dialog: true saves, false goes back to the form.
  const answer = useRef<((confirmed: boolean) => void) | null>(null);

  function answerEffects(confirmed: boolean) {
    setEffects(null);
    answer.current?.(confirmed);
    answer.current = null;
  }

  /** Asks the API what a new model or business unit changes, and the person whether to go on. */
  async function confirmEffects(values: PlanFormValues): Promise<boolean> {
    const modelChanged = values.modelId !== initial.modelId;
    const attributesChanged =
      JSON.stringify(values.attributes) !== JSON.stringify(initial.attributes);
    if (!modelChanged && !attributesChanged) {
      return true;
    }
    const data = await changeEffects.mutateAsync({
      modelId: values.modelId || null,
      attributes: values.attributes,
    });
    if (
      data.disallowedActivities.length === 0 &&
      data.hiddenGroups.length === 0 &&
      data.shownGroups.length === 0
    ) {
      return true;
    }
    const modelOnly = data.hiddenGroups.length === 0 && data.shownGroups.length === 0;
    return new Promise((resolve) => {
      answer.current = resolve;
      setEffects({ data, modelOnly });
    });
  }

  async function save(values: PlanFormValues): Promise<PlanSaveAnswer> {
    try {
      if (!(await confirmEffects(values))) {
        return null;
      }
      const saved = await updatePlan.mutateAsync(toUpdateBody(values, version, answers016(plan)));
      show({ message: `Plan '${saved.name}' saved` });
      return { to: `/plans/${encodeURIComponent(saved.id)}` };
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      await showFailure(apiError);
      return { problems: apiError.kind === 'validation' ? apiError.fields : [] };
    }
  }

  async function showFailure(apiError: ApiError) {
    if (apiError.kind === 'validation') {
      setFailure(null);
      return;
    }
    if (apiError.kind === 'conflict') {
      // The typed values stay; the form takes the version with the other person's change, so
      // Save again replaces it on purpose (FR-023).
      const latest = await fetchLatestPlan(queryClient, plan.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({
        message: conflictMessage(apiError, 'plan', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      return;
    }
    setFailure({ message: apiError.message });
  }

  return (
    <>
      <PlanForm
        mode="edit"
        initial={initial}
        plan={plan}
        failure={failure}
        onSave={save}
        cancelTo={`/plans/${encodeURIComponent(plan.id)}`}
        focusField={focusOf(search)}
      />
      <ChangeEffectsDialog
        effects={effects?.data ?? null}
        planId={plan.id}
        modelOnly={effects?.modelOnly ?? false}
        onConfirm={() => answerEffects(true)}
        onKeepEditing={() => answerEffects(false)}
      />
    </>
  );
}
