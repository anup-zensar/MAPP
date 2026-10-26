import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { useAttributeForm } from '@/features/settings/api';
import { Button, LinkButton } from '@/ui/Button/Button';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { ClearValuesDialog } from '@/ui/ShellAttributeFields/ClearValuesDialog';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import { useToast } from '@/ui/Toast/useToast';
import type { BreadcrumbItem } from '@/ui/Breadcrumb/Breadcrumb';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import type { ClearedValue } from '@/ui/ShellAttributeFields/rules';
import { useSpreadEffectText } from '@/features/line-items/useSpreadEffectText';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { usePageTitle } from '@/ui/usePageTitle';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import {
  fetchLatestActivity,
  useActivity,
  useCreateActivity,
  useEditActivityForm,
  useNewActivityForm,
  useUpdateActivity,
} from './activitiesApi';
import type { Activity, ActivityFormData } from './activitiesApi';
import { ActivityForm } from './ActivityForm';
import { activityRailSections } from './activityFormProgress';
import {
  fieldId,
  fromActivity,
  fromDefaults,
  isDirty,
  planCurrencyOf,
  reapprovalChanges,
  toCreateBody,
  toUpdateBody,
  validate,
} from './activityFormState';
import type { ReapprovalChange } from './activityFormState';
import styles from './ActivityFormPage.module.css';
import { FormSkeleton, LoadFailed } from './ActivityLoadStates';
import { activityHref } from './activityText';
import { FormLayout } from './FormRail';
import { ReapprovalDialog } from './ReapprovalDialog';

export interface ActivityFormPageProps {
  /**
   * new: /plans/:planId/initiatives/:initiativeId/activities/new;
   * edit: /plans/:planId/activities/:activityId/edit (?field=<field> focuses a field).
   */
  mode: 'new' | 'edit';
}

const INITIATIVE_GONE = 'This initiative no longer exists — someone may have deleted it.';
const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/**
 * The activity form, a full page for a new and an edited activity (feature 006, US1, US3, US4;
 * 006 plan, "Activity form page").
 */
export function ActivityFormPage({ mode }: ActivityFormPageProps) {
  return mode === 'new' ? <NewActivityPage /> : <EditActivityPage />;
}

/** The new-activity page: loads the form of the initiative, then the editor. */
function NewActivityPage() {
  const { initiativeId = '' } = useParams();
  const location = useLocation();
  usePageTitle('New activity');
  const query = useNewActivityForm(initiativeId);

  if (query.isPending || query.isError) {
    return (
      <>
        <FormHeader title="New activity" />
        {query.isPending ? (
          <FormSkeleton />
        ) : (
          <LoadFailed
            error={query.error}
            notFoundText={INITIATIVE_GONE}
            onRetry={() => void query.refetch()}
          />
        )}
      </>
    );
  }
  // A new location (Save and add another) starts a fresh, empty form.
  return <ActivityEditor key={location.key} form={query.data} />;
}

/** The edit page: loads the activity and its form, then the editor with the activity's values. */
function EditActivityPage() {
  const { activityId = '' } = useParams();
  const activity = useActivity(activityId);
  const form = useEditActivityForm(activityId);
  usePageTitle(activity.data ? `Edit ${activity.data.name}` : 'Edit activity');

  if (activity.isSuccess && form.isSuccess) {
    return <ActivityEditor key={activity.data.id} form={form.data} activity={activity.data} />;
  }
  const failed = activity.isError ? activity : form.isError ? form : null;
  return (
    <>
      <FormHeader title="Edit activity" />
      {failed ? (
        <LoadFailed
          error={failed.error}
          notFoundText={ACTIVITY_GONE}
          onRetry={() => void failed.refetch()}
        />
      ) : (
        <FormSkeleton />
      )}
    </>
  );
}

interface ActivityEditorProps {
  form: ActivityFormData;
  /** The edited activity, copied into the form once (001's edit-form rule); none for a new one. */
  activity?: Activity;
}

/**
 * The form with its checks, Save (and for a new activity Save and add another), the questions
 * before re-approval, and the protection of typed values.
 */
function ActivityEditor({ form, activity }: ActivityEditorProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const { show } = useToast();
  const { date, dateFormat } = useFormatters();
  const create = useCreateActivity();
  const update = useUpdateActivity();
  const attributeForm = useAttributeForm('activity');
  const [initial] = useState(() => (activity ? fromActivity(activity) : fromDefaults(form)));
  const [version, setVersion] = useState(activity?.version ?? 0);
  const [value, setValue] = useState(initial);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [saving, setSaving] = useState(false);
  // The dependent values saving would clear, while their question is open.
  const [clearing, setClearing] = useState<{ cleared: ClearedValue[]; addAnother: boolean } | null>(
    null,
  );
  // The changes that restart approval, while their question is open.
  const [reapproval, setReapproval] = useState<ReapprovalChange[] | null>(null);
  // Where to go once the activity is saved: the typed values are no longer unsaved then.
  const [leaveTo, setLeaveTo] = useState<string | null>(null);
  const { dialog } = useUnsavedChanges(leaveTo === null && isDirty(value, initial));
  const { plan, initiative } = form.context;
  // The planned cost is typed and sent in the plan currency (016 FR-011).
  const planCurrency = planCurrencyOf(form, activity);
  const focusField = searchParams.get('field');
  // What the typed dates do to the line items' spreads, for the re-approval question (D-150): the
  // same sentence SpreadEffectNote shows under the dates, from the same request.
  const spreadEffect = useSpreadEffectText(activity?.id ?? '', {
    startDate: value.startDate,
    endDate: value.ongoing ? '' : value.endDate,
    ongoing: value.ongoing,
  });

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

  async function save(addAnother = false) {
    if (saving) {
      return;
    }
    const found = validate(value, {
      form,
      mode: activity ? 'edit' : 'new',
      original: initial,
      date,
    });
    setFailure(null);
    setProblems(found);
    if (found.length > 0) {
      return;
    }
    const cleared = attributeForm.data
      ? valuesToClear(attributeForm.data, value.attributes, activity?.attributes ?? {})
      : [];
    if (cleared.length > 0) {
      setClearing({ cleared, addAnother });
      return;
    }
    await confirmApproval(addAnother);
  }

  /** Asks before a change sends a submitted or approved activity back for approval, then saves. */
  async function confirmApproval(addAnother: boolean) {
    const underApproval = activity?.status === 'submitted' || activity?.status === 'approved';
    const changes = underApproval ? reapprovalChanges(initial, value, date, planCurrency) : [];
    if (changes.length > 0) {
      setReapproval(changes);
      return;
    }
    await send(addAnother, false);
  }

  async function send(addAnother: boolean, restartApproval: boolean) {
    setSaving(true);
    try {
      if (activity) {
        const saved = await update.mutateAsync({
          id: activity.id,
          body: toUpdateBody(value, version, restartApproval, planCurrency),
        });
        setReapproval(null);
        show({ message: `Activity '${saved.name}' saved` });
        setLeaveTo(activityHref(plan.id, saved.id));
        return;
      }
      const created = await create.mutateAsync({
        initiativeId: initiative.id,
        body: toCreateBody(value, planCurrency),
      });
      const message = `Activity '${created.name}' created`;
      const href = activityHref(plan.id, created.id);
      if (addAnother) {
        show({
          message,
          action: {
            label: 'View',
            altText: 'Open it from the plan',
            onClick: () => void navigate(href),
          },
        });
        setLeaveTo(location.pathname);
      } else {
        show({ message });
        setLeaveTo(href);
      }
    } catch (error) {
      setReapproval(null);
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      return;
    }
    if (apiError.kind === 'conflict' && activity) {
      // The typed values stay; the form takes the version with the other person's change, so
      // Save again replaces it on purpose. When the activity cannot be read now, the next Save
      // meets the conflict again.
      const latest = await fetchLatestActivity(queryClient, activity.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({ message: conflictMessage(apiError, 'activity', dateFormat) });
      return;
    }
    setFailure({ message: apiError.message });
  }

  const cancelTo = activity
    ? activityHref(plan.id, activity.id)
    : `/plans/${encodeURIComponent(plan.id)}`;
  const railSections = activityRailSections(
    value,
    attributeForm.data,
    form.context.businessUnits.map((unit) => unit.id),
  );

  return (
    <>
      <FormHeader
        title={activity ? 'Edit activity' : 'New activity'}
        form={form}
        activity={activity}
      />
      <form
        noValidate
        aria-busy={saving}
        className={styles.form}
        onSubmit={(event) => {
          event.preventDefault();
          void save();
        }}
      >
        <FormLayout sections={railSections}>
          {failure && <FailureMessage failure={failure} />}
          <FormErrorSummary errors={problems} fieldId={fieldId} />
          <ActivityForm
            form={form}
            value={value}
            onChange={setValue}
            problems={problems}
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
            {saving ? 'Saving…' : 'Save'}
          </Button>
          {!activity && <Button onClick={() => void save(true)}>Save and add another</Button>}
          <LinkButton to={cancelTo}>Cancel</LinkButton>
        </FormFooter>
      </form>
      <ClearValuesDialog
        open={clearing !== null}
        onOpenChange={(open) => !open && setClearing(null)}
        cleared={clearing?.cleared ?? []}
        onConfirm={() => {
          const addAnother = clearing?.addAnother ?? false;
          setClearing(null);
          void confirmApproval(addAnother);
        }}
      />
      {activity && (
        <ReapprovalDialog
          open={reapproval !== null}
          onOpenChange={(open) => !open && setReapproval(null)}
          name={activity.name}
          changes={reapproval ?? []}
          spreadEffect={spreadEffect}
          onConfirm={() => void send(false, true)}
          busy={saving}
        />
      )}
      {dialog}
    </>
  );
}

interface FormHeaderProps {
  title: string;
  /** The form's context, once loaded: the breadcrumb and the description. */
  form?: ActivityFormData;
  /** The edited activity, linked in the breadcrumb. */
  activity?: { id: string; name: string };
}

/**
 * The form's breadcrumb and title (design system §2.3): "Plans › <plan> › <campaign> ›
 * <initiative> › <activity> › Edit activity" and "New activity" or "Edit activity" — the breadcrumb
 * names where it lives, so no eyebrow or description line repeats it (UI audit X-03).
 */
function FormHeader({ title, form, activity }: FormHeaderProps) {
  const items: BreadcrumbItem[] = [{ label: 'Plans', to: '/plans' }];
  if (form) {
    const { plan, campaign, initiative } = form.context;
    items.push(
      { label: plan.name, to: recordHref('plan', plan.id) },
      { label: campaign.name, to: recordHref('campaign', campaign.id, plan.id) },
      { label: initiative.name, to: recordHref('initiative', initiative.id, plan.id) },
    );
    if (activity) {
      items.push({ label: activity.name, to: activityHref(plan.id, activity.id) });
    }
  }
  items.push({ label: activity || title.startsWith('Edit') ? 'Edit' : title });
  return <PageHeader breadcrumb={items} title={title} />;
}
