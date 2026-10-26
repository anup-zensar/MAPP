import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useCurrentUser } from '@/session/useSession';
import { conflictMessage } from '@/api/errors';
import { DateRangeField } from '@/ui/DateRangeField/DateRangeField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import { PersonField, usePeople } from '@/features/people';
import { useFormatters } from '@/format/useFormatters';
import { fetchLatestDelegation, useCreateDelegation, useUpdateDelegation } from './delegationsApi';
import type { ApprovalDelegation, DelegationScope, DelegationUpdate } from './delegationsApi';
import { asApiError, isSessionLost } from './messages';

/** The longest note the API takes. */
const NOTE_MAX = 500;
/** The date fields' ids: the API's keys with the field's prefix (DateRangeField's ids). */
const DATE_IDS: Record<string, string> = {
  startDate: 'delegation-startDate',
  endDate: 'delegation-endDate',
};
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
  const [start] = useState(() => ({
    personId: delegation?.person.id ?? user.id,
    delegateId: delegation?.delegate.id ?? '',
    startDate: delegation?.startDate ?? '',
    endDate: delegation?.endDate ?? '',
    note: delegation?.note ?? '',
  }));
  const [values, setValues] = useState(start);
  const [version, setVersion] = useState(delegation?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const isDirty = (Object.keys(start) as (keyof typeof start)[]).some(
    (key) => values[key] !== start[key],
  );

  function set<K extends keyof typeof values>(key: K, value: (typeof values)[K]) {
    setValues((current) => ({ ...current, [key]: value }));
  }

  /** The delegate's name for the message, from the people list (the API answers it too). */
  function delegateName(saved: ApprovalDelegation): string {
    return (
      saved.delegate.name ||
      people.data?.items.find((person) => person.id === values.delegateId)?.name ||
      ''
    );
  }

  async function save() {
    setSaving(true);
    try {
      let saved: ApprovalDelegation;
      if (delegation) {
        const changes: DelegationUpdate = { version };
        if (values.delegateId !== start.delegateId) {
          changes.delegateId = values.delegateId;
        }
        if (values.startDate !== start.startDate) {
          changes.startDate = values.startDate;
        }
        if (values.endDate !== start.endDate) {
          changes.endDate = values.endDate;
        }
        if (values.note !== start.note) {
          changes.note = values.note.trim();
        }
        saved = await update.mutateAsync({ id: delegation.id, changes });
      } else {
        saved = await create.mutateAsync({
          ...(canManageOthers && values.personId !== user.id ? { personId: values.personId } : {}),
          delegateId: values.delegateId,
          startDate: values.startDate,
          endDate: values.endDate,
          ...(values.note.trim() ? { note: values.note.trim() } : {}),
        });
      }
      show({ message: `Delegation to ${delegateName(saved)} saved` });
      onClose();
    } catch (error) {
      const apiError = asApiError(error);
      if (apiError.kind === 'validation') {
        // The date problems point at DateRangeField's inputs, so the summary's links reach them.
        setProblems(
          apiError.fields.map((problem) => ({
            ...problem,
            field: DATE_IDS[problem.field] ?? problem.field,
          })),
        );
        setFailure(null);
      } else if (isSessionLost(apiError)) {
        setProblems([]);
        setFailure(null);
      } else if (apiError.kind === 'conflict' && apiError.conflict && delegation) {
        setProblems([]);
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
        setProblems([]);
        setFailure({ message: apiError.kind === 'notFound' ? GONE : apiError.message });
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <FormPanel
      title={delegation ? 'Edit delegation' : 'Add delegation'}
      description="Your delegate approves for you on these days, as well as you."
      isDirty={isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void save()}
      onClose={onClose}
    >
      {canManageOthers && !delegation && (
        <PersonField
          label="Delegating person"
          name="personId"
          marking="Required"
          value={values.personId}
          onChange={(personId) => set('personId', personId)}
          error={problemOf(problems, 'personId')}
        />
      )}
      <PersonField
        label="Delegate"
        name="delegateId"
        marking="Required"
        hint="Someone with a role that approves activities."
        value={values.delegateId}
        onChange={(delegateId) => set('delegateId', delegateId)}
        error={problemOf(problems, 'delegateId')}
      />
      <DateRangeField
        label="From – To"
        idPrefix="delegation"
        marking="Required"
        hint="At most 90 days, from today or later."
        start={values.startDate}
        end={values.endDate}
        onChange={({ start: startDate, end: endDate }) =>
          setValues((current) => ({ ...current, startDate, endDate }))
        }
        errors={{
          start: problemOf(problems, DATE_IDS.startDate),
          end: problemOf(problems, DATE_IDS.endDate),
        }}
      />
      <TextAreaField
        label="Note (optional)"
        name="note"
        rows={3}
        maxLength={NOTE_MAX}
        value={values.note}
        onChange={(note) => set('note', note)}
        error={problemOf(problems, 'note')}
      />
    </FormPanel>
  );
}
