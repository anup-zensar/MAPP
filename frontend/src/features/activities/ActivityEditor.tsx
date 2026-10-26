import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { useNavigate, useSearchParams } from 'react-router';
import { useAttributeForm } from '@/features/settings';
import { Button, LinkButton } from '@/ui/Button/Button';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { FormLayout } from '@/ui/FormRail/FormRail';
import { ClearValuesDialog } from '@/ui/ShellAttributeFields/ClearValuesDialog';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import { useSpreadEffectText } from '@/features/line-items';
import { useFormatters } from '@/format/useFormatters';
import type { Activity, ActivityFormData } from './api';
import { ActivityForm } from './ActivityForm';
import { activityRailSections } from './activity-form-progress';
import { fieldId, fromActivity, fromDefaults, planCurrencyOf } from './activity-form-state';
import type { FormState } from './activity-form-state';
import styles from './ActivityFormPage.module.css';
import { activityHref } from './activity-text';
import { ActivityFormHeader } from './ActivityFormHeader';
import { ReapprovalDialog } from './ReapprovalDialog';
import { activityFormSchema, focusFieldOf } from './schemas';
import { useActivitySave } from './useActivitySave';

const FIELDS = [
  'name',
  'externalName',
  'activityTypeId',
  'ownerId',
  'approvalWorkflowId',
  'startDate',
  'endDate',
  'ongoing',
  'plannedCost',
  'goals.inq',
  'goals.mql',
  'goals.revenue',
  'description',
] as const;

export interface ActivityEditorProps {
  data: ActivityFormData;
  /** The edited activity, copied into the form once (001's edit-form rule); none for a new one. */
  activity?: Activity;
}

export function ActivityEditor({ data, activity }: ActivityEditorProps) {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { date } = useFormatters();
  const attributeForm = useAttributeForm('activity');
  const [initial] = useState(() => (activity ? fromActivity(activity) : fromDefaults(data)));
  const form = useForm<FormState>({
    resolver: zodResolver(
      activityFormSchema({ form: data, mode: activity ? 'edit' : 'new', original: initial, date }),
    ),
    defaultValues: initial,
    ...CHECK_ON_SAVE,
  });
  const formProblems = useFormProblems(form, FIELDS);
  const value = useWatch({ control: form.control, compute: (values) => values });
  const planCurrency = planCurrencyOf(data, activity);
  const save = useActivitySave({
    data,
    activity,
    initial,
    attributeForm: attributeForm.data,
    planCurrency,
    problems: formProblems,
  });
  const { leaveTo } = save;
  const { dialog } = useUnsavedChanges(form.formState.isDirty && leaveTo === null);
  const focusField = focusFieldOf(searchParams);
  // What the typed dates do to the line items' spreads, for the re-approval question (D-150): the
  // same sentence SpreadEffectNote shows under the dates, from the same request.
  const spreadEffect = useSpreadEffectText(activity?.id ?? '', {
    startDate: value.startDate,
    endDate: value.ongoing ? '' : value.endDate,
    ongoing: value.ongoing,
  });

  // Declared after the guard: by the time it runs, the guard has let go of the page.
  useEffect(() => {
    if (leaveTo !== null) {
      void navigate(leaveTo);
    }
  }, [leaveTo, navigate]);

  // ?field=ownerId: the field a Ready to submit link leads to takes the focus once it is shown
  // (the owner field shows once the people have loaded).
  useEffect(() => {
    if (!focusField) {
      return;
    }
    let tries = 0;
    const timer = window.setInterval(() => {
      const field = document.getElementById(fieldId(focusField));
      tries += 1;
      if (field || tries > 40) {
        field?.focus();
        window.clearInterval(timer);
      }
    }, 50);
    return () => window.clearInterval(timer);
  }, [focusField]);

  const submit = (addAnother: boolean) =>
    form.handleSubmit((values) => {
      formProblems.clearProblems();
      save.save(values, addAnother);
    }, invalid);

  function invalid() {
    save.clearFailure();
    formProblems.onInvalid();
  }

  const { plan } = data.context;
  const cancelTo = activity
    ? activityHref(plan.id, activity.id)
    : `/plans/${encodeURIComponent(plan.id)}`;

  return (
    <>
      <ActivityFormHeader
        title={activity ? 'Edit activity' : 'New activity'}
        form={data}
        activity={activity}
      />
      <form
        noValidate
        aria-busy={save.saving}
        className={styles.form}
        onSubmit={(event) => void submit(false)(event)}
      >
        <FormLayout
          sections={activityRailSections(
            value,
            attributeForm.data,
            data.context.businessUnits.map((unit) => unit.id),
          )}
        >
          {save.failure && <FailureMessage failure={save.failure} />}
          <FormErrorSummary errors={formProblems.problems} fieldId={fieldId} />
          <ActivityForm
            form={form}
            data={data}
            problems={formProblems.problems}
            stored={activity}
            planCurrency={planCurrency}
            attributeForm={{
              data: attributeForm.data,
              error: attributeForm.error,
              refetch: () => void attributeForm.refetch(),
            }}
          />
        </FormLayout>
        <FormFooter>
          <Button type="submit" variant="primary">
            {save.saving ? 'Saving…' : 'Save'}
          </Button>
          {!activity && <Button onClick={() => void submit(true)()}>Save and add another</Button>}
          <LinkButton to={cancelTo}>Cancel</LinkButton>
        </FormFooter>
      </form>
      <ClearValuesDialog
        open={save.clearing !== null}
        onOpenChange={(open) => !open && save.cancelClearing()}
        cleared={save.clearing?.about ?? []}
        onConfirm={save.confirmClearing}
      />
      {activity && (
        <ReapprovalDialog
          open={save.reapproval !== null}
          onOpenChange={(open) => !open && save.cancelReapproval()}
          name={activity.name}
          changes={save.reapproval?.about ?? []}
          spreadEffect={spreadEffect}
          onConfirm={save.confirmReapproval}
          busy={save.saving}
        />
      )}
      {dialog}
    </>
  );
}
