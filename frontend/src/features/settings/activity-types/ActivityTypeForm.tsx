import { useState } from 'react';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { formatAmount, parseMoneyInput, toCents } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import styles from '@/ui/settings-table.module.css';
import { ImpactDialog } from '../ImpactDialog';
import { saveFailureOf } from '../save-failure';
import { useModels } from '../api';
import {
  checkActivityTypeImpact,
  fetchActivityType,
  useCreateActivityType,
  useUpdateActivityType,
} from './api';
import type { ActivityType, ActivityTypeCreate, ActivityTypeUpdate, ChangeImpact } from './api';

export interface ActivityTypeFormProps {
  /** The activity type to change; without one, the form adds a new activity type. */
  type?: ActivityType;
  onClose: () => void;
}

/** The typed values, as text, exactly as they are in the fields. */
interface ActivityTypeValues {
  name: string;
  defaultPlannedCost: string;
  modelIds: string[];
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
  const [start] = useState<ActivityTypeValues>(() => ({
    name: type?.name ?? '',
    defaultPlannedCost: type?.defaultPlannedCost
      ? formatAmount(type.defaultPlannedCost.amount)
      : '',
    modelIds: type?.models.map((model) => model.id) ?? [],
  }));
  const [values, setValues] = useState(start);
  const [version, setVersion] = useState(type?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [pendingImpact, setPendingImpact] = useState<PendingImpact | null>(null);
  const costChanged =
    type !== undefined && !sameCost(values.defaultPlannedCost, type.defaultPlannedCost);

  function set<K extends keyof ActivityTypeValues>(field: K, value: ActivityTypeValues[K]) {
    setValues((previous) => ({ ...previous, [field]: value }));
  }

  async function save() {
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
    setProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && type) {
      // Typed values stay; the next Save replaces the other person's change on purpose.
      const latest = await fetchActivityType(type.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  return (
    <>
      <FormPanel
        title={type ? 'Edit activity type' : 'New activity type'}
        isDirty={!sameValues(values, start)}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={() => void save()}
        onClose={onClose}
      >
        <div className={styles.formFields}>
          <TextField
            label="Name"
            name="name"
            marking="Required"
            maxLength={120}
            value={values.name}
            onChange={(name) => set('name', name)}
            error={problemOf(problems, 'name')}
          />
          <MoneyField
            label="Default planned cost"
            name="defaultPlannedCost"
            currency="USD"
            hint="Offered when someone picks this type for an activity."
            note={costChanged ? 'Applies when someone picks this type from now on.' : undefined}
            value={values.defaultPlannedCost}
            onChange={(cost) => set('defaultPlannedCost', cost)}
            error={problemOf(problems, 'defaultPlannedCost')}
          />
          <ModelsField
            type={type}
            value={values.modelIds}
            onChange={(ids) => set('modelIds', ids)}
            error={problemOf(problems, 'modelIds')}
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

interface ModelsFieldProps {
  /** The type being edited: the models allowing it stay offered even when inactive. */
  type?: ActivityType;
  value: string[];
  onChange: (ids: string[]) => void;
  error?: string;
}

/** Allowed in models: the active models, plus those that already allow the type. */
function ModelsField({ type, value, onChange, error }: ModelsFieldProps) {
  const models = useModels();

  if (!models.data) {
    return models.isError ? (
      <ErrorState error={models.error} onRetry={() => void models.refetch()} />
    ) : (
      <Skeleton lines={3} />
    );
  }

  const options: ChoiceOption[] = models.data.items.map((model) => ({
    id: model.id,
    label: model.name,
    active: model.active,
  }));
  for (const allowing of type?.models ?? []) {
    if (!options.some((option) => option.id === allowing.id)) {
      options.push({ id: allowing.id, label: allowing.name, active: allowing.active });
    }
  }

  return (
    <ChoiceField
      label="Allowed in models"
      name="modelIds"
      hint="Plans using these models can choose this type."
      multiple
      options={options}
      value={value}
      onChange={onChange}
      error={error}
    />
  );
}

function sameValues(a: ActivityTypeValues, b: ActivityTypeValues): boolean {
  return (
    a.name === b.name &&
    a.defaultPlannedCost === b.defaultPlannedCost &&
    sameIds(a.modelIds, b.modelIds)
  );
}

function sameIds(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
}

/** Whether a model that allowed the type is left out of the new list. */
function removesModels(before: string[], after: string[]): boolean {
  return before.some((id) => !after.includes(id));
}

/** A typed cost as the API takes it; null when nothing is typed (no default). */
function costInput(text: string): ActivityTypeCreate['defaultPlannedCost'] {
  if (text.trim() === '') {
    return null;
  }
  return { amount: parseMoneyInput(text)?.amount ?? text.trim(), currency: 'USD' };
}

/** "5,000" and 5000.00 USD are the same cost; empty is the same as no default. */
function sameCost(typed: string, saved: ActivityType['defaultPlannedCost']): boolean {
  if (typed.trim() === '' || saved === null) {
    return typed.trim() === '' && saved === null;
  }
  const amount = parseMoneyInput(typed);
  return amount !== null && toCents(amount.amount) === toCents(saved.amount);
}

function createBody(values: ActivityTypeValues): ActivityTypeCreate {
  return {
    name: values.name,
    defaultPlannedCost: costInput(values.defaultPlannedCost),
    modelIds: values.modelIds,
  };
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
function changedFields(
  type: ActivityType,
  start: ActivityTypeValues,
  values: ActivityTypeValues,
  version: number,
): ActivityTypeUpdate | null {
  const body: ActivityTypeUpdate = { version };
  if (values.name !== start.name) {
    body.name = values.name;
  }
  if (!sameCost(values.defaultPlannedCost, type.defaultPlannedCost)) {
    body.defaultPlannedCost = costInput(values.defaultPlannedCost);
  }
  if (!sameIds(values.modelIds, start.modelIds)) {
    // The complete new list of models allowing the type.
    body.modelIds = values.modelIds;
  }
  return Object.keys(body).length > 1 ? body : null;
}
