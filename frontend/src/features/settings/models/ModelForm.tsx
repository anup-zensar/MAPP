import { useState } from 'react';
import { useActivityTypes } from '@/api/settingsQueries';
import {
  ChoiceField,
  DecimalField,
  ErrorState,
  FormPanel,
  MoneyField,
  problemOf,
  Skeleton,
  TextField,
  useToast,
} from '@/components';
import type { ChoiceOption, FieldProblem, SaveFailure } from '@/components';
import {
  formatAmount,
  parseDecimalInput,
  parseMoneyInput,
  toCents,
  toHundredths,
} from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { ImpactDialog } from '../shared/ImpactDialog';
import { saveFailureOf } from '../shared/saveFailure';
import styles from '../shared/SettingsTable.module.css';
import { checkModelImpact, fetchModel, useCreateModel, useUpdateModel } from './modelsApi';
import type { ChangeImpact, Model, ModelCreate, ModelUpdate } from './modelsApi';
import { WaterfallExample } from './WaterfallExample';

/** What the form does: create a new model (blank or a copy of another) or change one. */
export type ModelFormMode =
  { kind: 'new' } | { kind: 'duplicate'; model: Model } | { kind: 'edit'; model: Model };

export interface ModelFormProps {
  mode: ModelFormMode;
  /** Opens a model's details: the View of the "created" message. */
  onView: (id: string) => void;
  onClose: () => void;
}

/** The typed values, as text, exactly as they are in the fields. */
interface ModelValues {
  name: string;
  inqToMqlRate: string;
  mqlToDealRate: string;
  averageDealSize: string;
  activityTypeIds: string[];
}

/** A save that waits for "Save changes" in the impact dialog. */
interface PendingImpact {
  body: ModelUpdate;
  warnings: ChangeImpact['warnings'];
}

/**
 * New model, Edit model and Duplicate (FR-009 – FR-012) in a side panel: name, both rates, the
 * average deal size, the allowed activity types and the live worked example. Every problem shows at
 * once; a change that affects plans or activities is confirmed first; a conflict keeps what was
 * typed and names who saved first.
 */
export function ModelForm({ mode, onView, onClose }: ModelFormProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createModel = useCreateModel();
  const updateModel = useUpdateModel();
  const editing = mode.kind === 'edit' ? mode.model : null;
  // The edit-form rule: the model is copied into the form once, when the panel opens.
  const [start] = useState(() => startValues(mode));
  const [values, setValues] = useState(start);
  const [version, setVersion] = useState(editing?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [pendingImpact, setPendingImpact] = useState<PendingImpact | null>(null);

  function set<K extends keyof ModelValues>(field: K, value: ModelValues[K]) {
    setValues((previous) => ({ ...previous, [field]: value }));
  }

  async function save() {
    setSaving(true);
    try {
      if (!editing) {
        const created = await createModel.mutateAsync(createBody(values));
        show({
          message: `Model '${created.name}' created`,
          action: {
            label: 'View',
            altText: 'Find it in the list of models',
            onClick: () => onView(created.id),
          },
        });
        onClose();
        return;
      }
      const body = changedFields(editing, start, values, version);
      if (!body) {
        onClose();
        return;
      }
      if (affectsOthers(body)) {
        const impact = await checkModelImpact(editing.id, body);
        if (impact.warnings.length > 0) {
          setPendingImpact({ body, warnings: impact.warnings });
          return;
        }
      }
      await saveChanges(editing, body);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function saveChanges(model: Model, body: ModelUpdate) {
    const saved = await updateModel.mutateAsync({ id: model.id, body });
    show({ message: `Model '${saved.name}' saved` });
    onClose();
  }

  async function confirmImpact() {
    if (!editing || !pendingImpact) {
      return;
    }
    setSaving(true);
    try {
      await saveChanges(editing, pendingImpact.body);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
      setPendingImpact(null);
    }
  }

  async function showFailure(error: unknown) {
    const view = saveFailureOf(error, 'model', dateFormat);
    setProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && editing) {
      // Typed values stay; the next Save replaces the other person's change on purpose.
      const latest = await fetchModel(editing.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  const dealSize = parseMoneyInput(values.averageDealSize)?.amount ?? null;

  return (
    <>
      <FormPanel
        title={editing ? 'Edit model' : 'New model'}
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
            value={values.name}
            onChange={(name) => set('name', name)}
            error={problemOf(problems, 'name')}
          />
          <DecimalField
            label="INQ→MQL rate"
            name="inqToMqlRate"
            marking="Required"
            decimals={2}
            suffix="%"
            value={values.inqToMqlRate}
            onChange={(rate) => set('inqToMqlRate', rate)}
            error={problemOf(problems, 'inqToMqlRate')}
          />
          <DecimalField
            label="MQL→deal rate"
            name="mqlToDealRate"
            marking="Required"
            decimals={2}
            suffix="%"
            value={values.mqlToDealRate}
            onChange={(rate) => set('mqlToDealRate', rate)}
            error={problemOf(problems, 'mqlToDealRate')}
          />
          <MoneyField
            label="Average deal size"
            name="averageDealSize"
            marking="Required"
            currency="USD"
            value={values.averageDealSize}
            onChange={(amount) => set('averageDealSize', amount)}
            error={problemOf(problems, 'averageDealSize')}
          />
          <ActivityTypesField
            model={editing}
            value={values.activityTypeIds}
            onChange={(ids) => set('activityTypeIds', ids)}
            error={problemOf(problems, 'activityTypeIds')}
          />
          <WaterfallExample
            inqToMqlRate={parseDecimalInput(values.inqToMqlRate, 2)}
            mqlToDealRate={parseDecimalInput(values.mqlToDealRate, 2)}
            averageDealSize={dealSize}
          />
        </div>
      </FormPanel>
      {pendingImpact && editing && (
        <ImpactDialog
          kind="model"
          name={editing.name}
          warnings={pendingImpact.warnings}
          busy={saving}
          onConfirm={() => void confirmImpact()}
          onCancel={() => setPendingImpact(null)}
        />
      )}
    </>
  );
}

interface ActivityTypesFieldProps {
  /** The model being edited: its allowed types stay offered even when inactive. */
  model: Model | null;
  value: string[];
  onChange: (ids: string[]) => void;
  error?: string;
}

/** Allowed activity types: the active types, plus those the model already allows. */
function ActivityTypesField({ model, value, onChange, error }: ActivityTypesFieldProps) {
  const activityTypes = useActivityTypes();

  if (!activityTypes.data) {
    return activityTypes.isError ? (
      <ErrorState error={activityTypes.error} onRetry={() => void activityTypes.refetch()} />
    ) : (
      <Skeleton lines={3} />
    );
  }

  const options: ChoiceOption[] = activityTypes.data.items.map((type) => ({
    id: type.id,
    label: type.name,
    active: type.active,
  }));
  for (const allowed of model?.activityTypes ?? []) {
    if (!options.some((option) => option.id === allowed.id)) {
      options.push({ id: allowed.id, label: allowed.name, active: allowed.active });
    }
  }

  return (
    <ChoiceField
      label="Allowed activity types"
      name="activityTypeIds"
      marking="Required"
      hint="Activities in plans using this model can be of these types."
      multiple
      options={options}
      value={value}
      onChange={onChange}
      error={error}
    />
  );
}

function startValues(mode: ModelFormMode): ModelValues {
  if (mode.kind === 'new') {
    return {
      name: '',
      inqToMqlRate: '',
      mqlToDealRate: '',
      averageDealSize: '',
      activityTypeIds: [],
    };
  }
  const { model } = mode;
  return {
    name: mode.kind === 'duplicate' ? `${model.name} (copy)` : model.name,
    inqToMqlRate: model.inqToMqlRate,
    mqlToDealRate: model.mqlToDealRate,
    averageDealSize: formatAmount(model.averageDealSize.amount),
    // A new model can allow active types only.
    activityTypeIds: model.activityTypes
      .filter((type) => mode.kind === 'edit' || type.active)
      .map((type) => type.id),
  };
}

function sameValues(a: ModelValues, b: ModelValues): boolean {
  return (
    a.name === b.name &&
    a.inqToMqlRate === b.inqToMqlRate &&
    a.mqlToDealRate === b.mqlToDealRate &&
    a.averageDealSize === b.averageDealSize &&
    sameIds(a.activityTypeIds, b.activityTypeIds)
  );
}

function sameIds(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
}

/** A typed rate as the API takes it: "2,5" stays as typed, so the API says what is wrong. */
function rateText(text: string): string {
  return parseDecimalInput(text, 2) ?? text.trim();
}

/** A typed amount as the API takes it; null when nothing is typed. */
function dealSizeInput(text: string): ModelCreate['averageDealSize'] {
  if (text.trim() === '') {
    return null;
  }
  return { amount: parseMoneyInput(text)?.amount ?? text.trim(), currency: 'USD' };
}

function createBody(values: ModelValues): ModelCreate {
  return {
    name: values.name,
    inqToMqlRate: rateText(values.inqToMqlRate),
    mqlToDealRate: rateText(values.mqlToDealRate),
    averageDealSize: dealSizeInput(values.averageDealSize),
    activityTypeIds: values.activityTypeIds,
  };
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
function changedFields(
  model: Model,
  start: ModelValues,
  values: ModelValues,
  version: number,
): ModelUpdate | null {
  const body: ModelUpdate = { version };
  if (values.name !== start.name) {
    body.name = values.name;
  }
  if (!sameRate(values.inqToMqlRate, model.inqToMqlRate)) {
    body.inqToMqlRate = rateText(values.inqToMqlRate);
  }
  if (!sameRate(values.mqlToDealRate, model.mqlToDealRate)) {
    body.mqlToDealRate = rateText(values.mqlToDealRate);
  }
  if (!sameAmount(values.averageDealSize, model.averageDealSize.amount)) {
    body.averageDealSize = dealSizeInput(values.averageDealSize);
  }
  if (!sameIds(values.activityTypeIds, start.activityTypeIds)) {
    body.activityTypeIds = values.activityTypeIds;
  }
  return Object.keys(body).length > 1 ? body : null;
}

/** Changes that may affect plans and activities, checked with …/impact first (FR-011, FR-012). */
function affectsOthers(body: ModelUpdate): boolean {
  return (
    body.inqToMqlRate !== undefined ||
    body.mqlToDealRate !== undefined ||
    body.averageDealSize !== undefined ||
    body.activityTypeIds !== undefined
  );
}

function sameRate(typed: string, saved: string): boolean {
  const rate = parseDecimalInput(typed, 2);
  return rate !== null && toHundredths(rate) === toHundredths(saved);
}

function sameAmount(typed: string, saved: string): boolean {
  const amount = parseMoneyInput(typed);
  return amount !== null && toCents(amount.amount) === toCents(saved);
}
