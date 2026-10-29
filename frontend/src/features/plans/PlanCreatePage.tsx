import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { forbiddenMessage } from '@/session/access';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { useToast } from '@/ui/Toast/useToast';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { usePageTitle } from '@/ui/usePageTitle';
import { PlanForm } from './PlanForm';
import type { PlanSaveAnswer } from './PlanForm';
import { PlanFormSkeleton } from './PlanFormSkeleton';
import { answers016, toCreateBody, valuesFromDefaults } from './plan-form-values';
import { useCreatePlan, usePlanDefaults } from './api';
import type { PlanFormValues } from './schemas';

// The plan form filled in with the person's defaults; saving opens the new Draft plan's page.
export function PlanCreatePage() {
  usePageTitle('New plan');
  const defaults = usePlanDefaults();
  const createPlan = useCreatePlan();
  const { show } = useToast();
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save(values: PlanFormValues): Promise<PlanSaveAnswer> {
    try {
      const plan = await createPlan.mutateAsync(toCreateBody(values, answers016(defaults.data)));
      show({ message: `Plan '${plan.name}' created — add campaigns next.` });
      return { to: `/plans/${encodeURIComponent(plan.id)}` };
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      setFailure(apiError.kind === 'validation' ? null : { message: apiError.message });
      return { problems: apiError.kind === 'validation' ? apiError.fields : [] };
    }
  }

  const refused = forbiddenMessage(defaults.error);
  if (refused) {
    return <NoAccess message={refused} />;
  }

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: 'New plan' }]}
        title="New plan"
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
          failure={failure}
          onSave={save}
          cancelTo="/plans"
        />
      )}
    </>
  );
}
