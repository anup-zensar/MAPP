import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import styles from '@/ui/settings-table.module.css';
import { ImpactDialog } from '../ImpactDialog';
import { saveFailureOf } from '../save-failure';
import {
  changedFields,
  createBody,
  removesModels,
  sameCost,
  startValues,
} from './activity-type-values';
import {
  checkActivityTypeImpact,
  fetchActivityType,
  useCreateActivityType,
  useUpdateActivityType,
} from './api';
import type { ActivityType, ActivityTypeUpdate, ChangeImpact } from './api';
import { ModelsField } from './ModelsField';
import { activityTypeSchema } from './schemas';
import type { ActivityTypeValues } from './schemas';

const FIELDS = ['name', 'defaultPlannedCost', 'modelIds'] as const;

export interface ActivityTypeFormProps {
  /** The activity type to change; without one, the form adds a new activity type. */
  type?: ActivityType;
  onClose: () => void;
}

/** A save that waits for "Save changes" in the impact dialog. */
interface PendingImpact {
  body: ActivityTypeUpdate;
  warnings: ChangeImpact['warnings'];
}

/**
 * New activity type and Edit activity type (FR-012, FR-013) in a side panel: the name, the optional
 * default planned cost in USD and the models allowing it. Every problem shows at once; removing
 * models whose plans have activities of this type is confirmed first; a conflict keeps what was
 * typed and names who saved first.
 */
export function ActivityTypeForm({ type, onClose }: ActivityTypeFormProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createType = useCreateActivityType();
  const updateType = useUpdateActivityType();
  // The edit-form rule: the type is copied into the form once, when the panel opens.
  const [start] = useState(() => startValues(type));
  const form = useForm<ActivityTypeValues>({
    resolver: zodResolver(activityTypeSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(type?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [pendingImpact, setPendingImpact] = useState<PendingImpact | null>(null);
  const cost = useWatch({ control: form.control, name: 'defaultPlannedCost' });
  const costChanged = type !== undefined && !sameCost(cost, type.defaultPlannedCost);

  async function save(values: ActivityTypeValues) {
    clearProblems();
    setSaving(true);
    try {
      if (!type) {
        const created = await createType.mutateAsync(createBody(values));
        show({ message: `Activity type '${created.name}' created` });
        onClose();
        return;
      }
      const body = changedFields(type, start, values, version);
      if (!body) {
        onClose();
        return;
      }
      // Removing models limits what their plans' new activities can choose (FR-012).
      if (body.modelIds && removesModels(start.modelIds, body.modelIds)) {
        const impact = await checkActivityTypeImpact(type.id, body);
        if (impact.warnings.length > 0) {
          setPendingImpact({ body, warnings: impact.warnings });
          return;
        }
      }
      await saveChanges(type, body);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function saveChanges(saved: ActivityType, body: ActivityTypeUpdate) {
    const updated = await updateType.mutateAsync({ id: saved.id, body });
    show({ message: `Activity type '${updated.name}' saved` });
    onClose();
  }

  async function confirmImpact() {
    if (!type || !pendingImpact) {
      return;
    }
    setSaving(true);
    try {
      await saveChanges(type, pendingImpact.body);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
      setPendingImpact(null);
    }
  }

  async function showFailure(error: unknown) {
    const view = saveFailureOf(error, 'activityType', dateFormat);
    showServerProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && type) {
      // Typed values stay; the next Save replaces the other person's change on purpose.
      const latest = await fetchActivityType(type.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  const submit = form.handleSubmit(save, onInvalid);

  return (
    <>
      <FormPanel
        title={type ? 'Edit activity type' : 'New activity type'}
        isDirty={form.formState.isDirty}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={() => void submit()}
        onClose={onClose}
      >
        <div className={styles.formFields}>
          <FormField
            control={form.control}
            name="name"
            render={(field) => (
              <TextField label="Name" marking="Required" maxLength={120} {...field} />
            )}
          />
          <FormField
            control={form.control}
            name="defaultPlannedCost"
            render={(field) => (
              <MoneyField
                label="Default planned cost"
                currency="USD"
                hint="Offered when someone picks this type for an activity."
                note={costChanged ? 'Applies when someone picks this type from now on.' : undefined}
                {...field}
              />
            )}
          />
          <FormField
            control={form.control}
            name="modelIds"
            render={(field) => <ModelsField type={type} {...field} />}
          />
        </div>
      </FormPanel>
      {pendingImpact && type && (
        <ImpactDialog
          kind="activityType"
          name={type.name}
          warnings={pendingImpact.warnings}
          busy={saving}
          onConfirm={() => void confirmImpact()}
          onCancel={() => setPendingImpact(null)}
        />
      )}
    </>
  );
}
