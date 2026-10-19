import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { ErrorState, PageHeader, useToast } from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { PlanForm, PlanFormSkeleton } from './PlanForm';
import { toCreateBody, valuesFromDefaults } from './planFormValues';
import type { PlanFormValues } from './planFormValues';
import { useCreatePlan, usePlanDefaults } from './plansApi';

/**
 * New plan (US1, FR-005 – FR-010): the plan form filled in with the person's defaults. Saving
 * creates a Draft plan and opens its page with "Plan '<name>' created — add campaigns next."
 */
export function PlanCreatePage() {
  usePageTitle('New plan');
  const defaults = usePlanDefaults();
  const createPlan = useCreatePlan();
  const { show } = useToast();
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save(values: PlanFormValues): Promise<string | null> {
    try {
      const plan = await createPlan.mutateAsync(toCreateBody(values));
      show({ message: `Plan '${plan.name}' created — add campaigns next.` });
      return `/plans/${encodeURIComponent(plan.id)}`;
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      setProblems(apiError.kind === 'validation' ? apiError.fields : []);
      setFailure(apiError.kind === 'validation' ? null : { message: apiError.message });
      return null;
    }
  }

  return (
    <>
      <PageHeader
        eyebrow="Plans"
        title="New plan"
        description="A name, a model and a business unit are enough to start — everything else can follow."
      />
      {defaults.isError ? (
        <ErrorState error={defaults.error} onRetry={() => void defaults.refetch()} />
      ) : defaults.isPending ? (
        <PlanFormSkeleton />
      ) : (
        <PlanForm
          mode="create"
          initial={valuesFromDefaults(defaults.data)}
          defaults={defaults.data}
          problems={problems}
          failure={failure}
          onSave={save}
          cancelTo="/plans"
        />
      )}
    </>
  );
}
