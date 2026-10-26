import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { useCurrentUser } from '@/session/useSession';
import { conflictMessage } from '@/api/errors';
import { DateRangeField } from '@/ui/DateRangeField/DateRangeField';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { PersonField, usePeople } from '@/features/people';
import { useFormatters } from '@/format/useFormatters';
import { asApiError, isSessionLost } from '@/api/errors';
import { fetchLatestDelegation, useCreateDelegation, useUpdateDelegation } from './delegations-api';
import type { ApprovalDelegation, DelegationScope } from './delegations-api';
import { delegationChanges, delegationSchema, delegationValuesOf, newDelegation } from './schemas';
import type { DelegationValues } from './schemas';

/** The longest note the API takes. */
const NOTE_MAX = 500;
const FIELDS = ['personId', 'delegateId', 'startDate', 'endDate', 'note'] as const;
/** The date fields' ids: the API's keys with the field's prefix (DateRangeField's ids). */
const DATE_IDS = { startDate: 'delegation-startDate', endDate: 'delegation-endDate' } as const;
const GONE = 'This delegation no longer exists.';
const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

export interface DelegationFormProps {
  /** The delegation to change; without one, the form adds a delegation. */
  delegation?: ApprovalDelegation;
  /** The person may add delegations for someone else (workflows.manage). */
  canManageOthers: boolean;
  /** The list the delegation was opened from (its latest version is read from there). */
  scope: DelegationScope;
  /** Back to the list of delegations. */
  onClose: () => void;
}

/**
 * "Add delegation" / "Edit delegation" (017 US4, FR-022, FR-024): the delegating person (only for
 * someone who may manage others' delegations; the signed-in person otherwise), the delegate, the
 * first and last day and an optional note. The API checks everything and every problem shows at
 * once, in the summary and at its field; a change someone saved first is named with its time.
 */
export function DelegationForm({
  delegation,
  canManageOthers,
  scope,
  onClose,
}: DelegationFormProps) {
  const queryClient = useQueryClient();
  const user = useCurrentUser();
  const people = usePeople();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const create = useCreateDelegation();
  const update = useUpdateDelegation();
  const form = useForm<DelegationValues>({
    resolver: zodResolver(delegationSchema),
    defaultValues: delegationValuesOf(delegation, user.id),
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [startDate, endDate] = useWatch({ control: form.control, name: ['startDate', 'endDate'] });
  const [version, setVersion] = useState(delegation?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  /** The delegate's name for the message, from the people list (the API answers it too). */
  function delegateName(saved: ApprovalDelegation, delegateId: string): string {
    return (
      saved.delegate.name ||
      people.data?.items.find((person) => person.id === delegateId)?.name ||
      ''
    );
  }

  function saveDelegation(values: DelegationValues): Promise<ApprovalDelegation> {
    if (delegation) {
      const start = delegationValuesOf(delegation, user.id);
      const changes = delegationChanges(values, start, version);
      return update.mutateAsync({ id: delegation.id, changes });
    }
    return create.mutateAsync(newDelegation(values, user.id, canManageOthers));
  }

  async function save(values: DelegationValues) {
    setSaving(true);
    try {
      const saved = await saveDelegation(values);
      show({ message: `Delegation to ${delegateName(saved, values.delegateId)} saved` });
      onClose();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = asApiError(error);
    if (apiError.kind === 'validation') {
      // The date problems point at DateRangeField's inputs, so the summary's links reach them.
      showServerProblems(
        apiError.fields.map((problem) => ({ ...problem, field: idOf(problem.field) })),
      );
      setFailure(null);
      return;
    }
    clearProblems();
    if (isSessionLost(apiError)) {
      setFailure(null);
    } else if (apiError.kind === 'conflict' && apiError.conflict && delegation) {
      // The typed changes stay; Save again replaces the other change on purpose.
      const latest = await fetchLatestDelegation(queryClient, scope, delegation.id).catch(
        () => undefined,
      );
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({
        message: conflictMessage(apiError, 'delegation', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
    } else {
      setFailure({ message: apiError.kind === 'notFound' ? GONE : apiError.message });
    }
  }

  return (
    <FormPanel
      title={delegation ? 'Edit delegation' : 'Add delegation'}
      description="Your delegate approves for you on these days, as well as you."
      isDirty={form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void form.handleSubmit(save, onInvalid)()}
      onClose={onClose}
    >
      {canManageOthers && !delegation && (
        <FormField
          control={form.control}
          name="personId"
          render={(field) => (
            <PersonField label="Delegating person" marking="Required" {...field} />
          )}
        />
      )}
      <FormField
        control={form.control}
        name="delegateId"
        render={(field) => (
          <PersonField
            label="Delegate"
            marking="Required"
            hint="Someone with a role that approves activities."
            {...field}
          />
        )}
      />
      <DateRangeField
        label="From – To"
        idPrefix="delegation"
        marking="Required"
        hint="At most 90 days, from today or later."
        start={startDate}
        end={endDate}
        onChange={({ start, end }) => {
          form.setValue('startDate', start, { shouldDirty: true });
          form.setValue('endDate', end, { shouldDirty: true });
        }}
        errors={{
          start: problemOf(problems, DATE_IDS.startDate),
          end: problemOf(problems, DATE_IDS.endDate),
        }}
      />
      <FormField
        control={form.control}
        name="note"
        render={(field) => (
          <TextAreaField label="Note (optional)" rows={3} maxLength={NOTE_MAX} {...field} />
        )}
      />
    </FormPanel>
  );
}

function idOf(field: string): string {
  return field === 'startDate' || field === 'endDate' ? DATE_IDS[field] : field;
}
