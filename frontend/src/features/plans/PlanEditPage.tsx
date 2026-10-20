import { useQueryClient } from '@tanstack/react-query';
import { useRef, useState } from 'react';
import { useParams, useSearchParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { ErrorState, PageHeader, useToast } from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { ChangeEffectsDialog } from './dialogs/ChangeEffectsDialog';
import { PlanForm, PlanFormSkeleton } from './PlanForm';
import { toUpdateBody, valuesFromPlan } from './planFormValues';
import type { PlanFormValues } from './planFormValues';
import { PlanNotFound } from './PlanNotFound';
import { fetchLatestPlan, useChangeEffects, usePlan, useUpdatePlan } from './plansApi';
import type { Plan, PlanChangeEffects } from './plansApi';

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

/**
 * Edit plan (US4, FR-018 – FR-023), at /plans/:planId/edit (?focus=<field> focuses a field): the
 * plan form with the year locked. A new model or business unit is confirmed first; a save after
 * someone else's is refused, naming them, with the typed values kept.
 */
export function PlanEditPage() {
  const { planId = '' } = useParams();
  const plan = usePlan(planId);
  usePageTitle(plan.data ? `Edit ${plan.data.name}` : 'Edit plan');

  if (plan.isError && plan.error instanceof ApiError && plan.error.kind === 'notFound') {
    return <PlanNotFound />;
  }

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Plans', to: '/plans' },
          { label: plan.data?.name ?? planId, to: `/plans/${encodeURIComponent(planId)}` },
          { label: 'Edit' },
        ]}
        eyebrow="Plans"
        title="Edit plan"
        description={plan.data?.name}
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

/** The form, with the plan as it was when the page opened (the edit-form rule). */
function EditForm({ plan: loaded }: { plan: Plan }) {
  const [plan] = useState(loaded);
  const [initial] = useState(() => valuesFromPlan(loaded));
  const [focusField] = useSearchParams();
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const updatePlan = useUpdatePlan(plan.id);
  const changeEffects = useChangeEffects(plan.id);
  const [version, setVersion] = useState(plan.version);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
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

  async function save(values: PlanFormValues): Promise<string | null> {
    try {
      if (!(await confirmEffects(values))) {
        return null;
      }
      const saved = await updatePlan.mutateAsync(toUpdateBody(values, version));
      show({ message: `Plan '${saved.name}' saved` });
      return `/plans/${encodeURIComponent(saved.id)}`;
    } catch (error) {
      await showFailure(error);
      return null;
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      setFailure(null);
      return;
    }
    setProblems([]);
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
        problems={problems}
        failure={failure}
        onSave={save}
        cancelTo={`/plans/${encodeURIComponent(plan.id)}`}
        focusField={focusField.get('focus')}
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
