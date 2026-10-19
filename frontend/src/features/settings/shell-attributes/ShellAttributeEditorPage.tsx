import { useEffect, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import {
  Breadcrumb,
  Button,
  Card,
  ChoiceField,
  ConfirmDialog,
  ErrorState,
  FailureMessage,
  FormErrorSummary,
  FormFooter,
  FormSection,
  LockedNote,
  PageHeader,
  problemOf,
  Skeleton,
  TextAreaField,
  TextField,
  useToast,
} from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { useUnsavedChanges } from '@/lib/useUnsavedChanges';
import {
  isChoiceType,
  isLevel,
  LEVEL_WORDS,
  LEVELS,
  lockReason,
  MARKING_WORDS,
  markingsFor,
  TYPE_WORDS,
  typesFor,
} from './attributeWords';
import type { AttributeLevel } from './attributeWords';
import { OptionsEditor } from './OptionsEditor';
import type { OptionRow } from './OptionsEditor';
import {
  checkAttributeImpact,
  fetchAttribute,
  useAllShellAttributes,
  useCreateAttribute,
  useShellAttribute,
  useUpdateAttribute,
} from './shellAttributesApi';
import type { ShellAttribute, ShellAttributeUpdate } from './shellAttributesApi';
import { AttributeNotFound } from './ShellAttributePage';
import styles from './ShellAttributes.module.css';
import { CONFLICT_NEXT_STEP } from './useStatusChange';

const DEFAULT_SECTION = 'Shell attributes';
const YES_NO_HINT = 'Always optional; shown as a tick box that defaults to No.';
const SPLIT_HINT =
  "People split the record's cost over the chosen values; the shares must add up to 100 %.";

export interface ShellAttributeEditorPageProps {
  /** create: /settings/shell-attributes/new; edit: /settings/shell-attributes/:attributeId/edit. */
  mode: 'create' | 'edit';
}

/**
 * The Shell attribute editor (feature 002, US2, FR-016 – FR-019, FR-021), a full page for create
 * and edit: Basics, Place on the form and — on create — the first options; locked fields with
 * their reason; every problem at once; the impact check before a change of marking or type;
 * unsaved changes protected.
 */
export function ShellAttributeEditorPage({ mode }: ShellAttributeEditorPageProps) {
  return mode === 'create' ? <CreateAttribute /> : <EditAttribute />;
}

/** /settings/shell-attributes/new?level=activity */
function CreateAttribute() {
  usePageTitle('New Shell attribute');
  const [params] = useSearchParams();
  const levelParam = params.get('level');
  const level = isLevel(levelParam) ? levelParam : 'plan';

  return (
    <>
      <Breadcrumb
        items={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: `/settings/shell-attributes?level=${level}` },
          { label: 'New Shell attribute' },
        ]}
      />
      <PageHeader eyebrow="Settings" title="New Shell attribute" />
      <AttributeForm level={level} />
    </>
  );
}

/** /settings/shell-attributes/:attributeId/edit */
function EditAttribute() {
  const { attributeId = '' } = useParams();
  const attribute = useShellAttribute(attributeId);
  usePageTitle(attribute.data ? `Edit ${attribute.data.label}` : 'Edit Shell attribute');

  if (attribute.data) {
    const { id, label, level } = attribute.data;
    return (
      <>
        <Breadcrumb
          items={[
            { label: 'Settings', to: '/settings' },
            { label: 'Shell attributes', to: '/settings/shell-attributes' },
            {
              label: `${LEVEL_WORDS[level]} form`,
              to: `/settings/shell-attributes?level=${level}`,
            },
            { label, to: `/settings/shell-attributes/${id}` },
            { label: 'Edit' },
          ]}
        />
        <PageHeader eyebrow="Settings" title={`Edit Shell attribute '${label}'`} />
        {/* The edit-form rule: the attribute is copied into the form once, when it opens. */}
        <AttributeForm key={id} level={level} attribute={attribute.data} />
      </>
    );
  }

  const notFound = attribute.error instanceof ApiError && attribute.error.kind === 'notFound';
  return (
    <>
      <Breadcrumb
        items={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: '/settings/shell-attributes' },
          { label: 'Edit' },
        ]}
      />
      <PageHeader eyebrow="Settings" title="Edit Shell attribute" />
      <Card>
        {notFound ? (
          <AttributeNotFound />
        ) : attribute.isError ? (
          <ErrorState error={attribute.error} onRetry={() => attribute.refetch()} />
        ) : (
          <Skeleton lines={6} height="18px" />
        )}
      </Card>
    </>
  );
}

/** What the form holds while it is being filled in. */
interface FormValues {
  label: string;
  helpText: string;
  level: AttributeLevel;
  /** "" until a type is chosen. */
  type: string;
  /** "" until a marking is chosen. */
  marking: string;
  section: string;
}

/** The fields an edit can change and send (D-203: only the changed ones). */
const FIELDS = ['label', 'helpText', 'level', 'type', 'marking', 'section'] as const;

interface AttributeFormProps {
  /** The level of a new attribute, or the level of the attribute being edited. */
  level: AttributeLevel;
  /** The attribute to edit; without one, the form creates a new attribute. */
  attribute?: ShellAttribute;
}

/** The editor's form, its save flow and its dialogs. */
function AttributeForm({ level, attribute }: AttributeFormProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const attributes = useAllShellAttributes();
  const createAttribute = useCreateAttribute();
  const updateAttribute = useUpdateAttribute();

  const [start] = useState<FormValues>(() => ({
    label: attribute?.label ?? '',
    helpText: attribute?.helpText ?? '',
    level,
    type: attribute?.type ?? '',
    marking: attribute?.marking ?? 'optional',
    section: attribute?.section ?? DEFAULT_SECTION,
  }));
  const [values, setValues] = useState<FormValues>(start);
  const [rows, setRows] = useState<OptionRow[]>([]);
  const [version, setVersion] = useState(attribute?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  // The warnings of the impact check, waiting for Save changes or Keep editing (FR-021).
  const [warnings, setWarnings] = useState<string[] | null>(null);
  // Where to go once saved; leaving is then no longer "losing changes".
  const [leaveTo, setLeaveTo] = useState<string | null>(null);

  const changed = FIELDS.filter((field) => values[field] !== start[field]);
  const isDirty = leaveTo === null && (changed.length > 0 || rows.length > 0);
  const { dialog } = useUnsavedChanges(isDirty);

  useEffect(() => {
    if (leaveTo !== null) {
      navigate(leaveTo);
    }
  }, [leaveTo, navigate]);

  const levelLock = attribute && lockReason(attribute, 'level');
  const typeLock = attribute && lockReason(attribute, 'type');
  const sectionLock = attribute && lockReason(attribute, 'section');
  const sections = [
    ...new Set(
      (attributes.data?.items ?? [])
        .filter((each) => each.level === values.level)
        .map((each) => each.section),
    ),
  ];
  const backTo = attribute
    ? `/settings/shell-attributes/${attribute.id}`
    : `/settings/shell-attributes?level=${values.level}`;

  function change(next: Partial<FormValues>) {
    setValues((current) => ({ ...current, ...next }));
  }

  /** A level offers its own types and markings; a choice it doesn't offer is cleared. */
  function changeLevel(next: string) {
    if (!isLevel(next)) {
      return;
    }
    const types: string[] = typesFor(next);
    const markings: string[] = markingsFor(next);
    const type = types.includes(values.type) ? values.type : '';
    const marking = type === 'yesNo' || markings.includes(values.marking) ? values.marking : '';
    change({ level: next, type, marking });
  }

  /** Yes/no attributes are always optional (FR-017). */
  function changeType(type: string) {
    change(type === 'yesNo' ? { type, marking: 'optional' } : { type });
  }

  /** Instant feedback for what can be checked here; the API checks everything (all at once). */
  function localProblems(): FieldProblem[] {
    const found: FieldProblem[] = [];
    if (values.label.trim() === '') {
      found.push({ field: 'label', message: 'Enter a label.' });
    }
    if (values.type === '') {
      found.push({ field: 'type', message: 'Choose a type.' });
    }
    if (values.marking === '') {
      found.push({ field: 'marking', message: 'Choose a marking.' });
    }
    if (!attribute && isChoiceType(values.type)) {
      found.push(...optionProblems(rows));
    }
    return found;
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (saving) {
      return;
    }
    const found = localProblems();
    setProblems(found);
    setFailure(null);
    if (found.length > 0) {
      return;
    }
    if (!attribute) {
      await create();
      return;
    }
    if (changed.length === 0) {
      setLeaveTo(backTo);
      return;
    }
    // A new marking or type can make records incomplete or remove options: say so first.
    if (changed.includes('marking') || changed.includes('type')) {
      setSaving(true);
      try {
        const impact = await checkAttributeImpact({ id: attribute.id, body: changes() });
        if (impact.warnings.length > 0) {
          setWarnings(impact.warnings.map((warning) => warning.message));
          setSaving(false);
          return;
        }
      } catch (error) {
        await showFailure(error);
        setSaving(false);
        return;
      }
    }
    await save();
  }

  /** The changed fields and the version the person started from. */
  function changes(): ShellAttributeUpdate {
    const body: ShellAttributeUpdate = { version };
    for (const field of changed) {
      body[field] = values[field];
    }
    return body;
  }

  async function create() {
    setSaving(true);
    try {
      const created = await createAttribute.mutateAsync({
        label: values.label,
        helpText: values.helpText,
        level: values.level,
        type: values.type,
        marking: values.marking,
        section: values.section,
        options: isChoiceType(values.type) ? rows.map((row) => ({ label: row.label })) : [],
      });
      show({ message: `Shell attribute '${created.label}' created` });
      setLeaveTo(`/settings/shell-attributes/${created.id}`);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function save() {
    if (!attribute) {
      return;
    }
    setWarnings(null);
    setSaving(true);
    try {
      const saved = await updateAttribute.mutateAsync({ id: attribute.id, body: changes() });
      show({ message: `Shell attribute '${saved.label}' saved` });
      setLeaveTo(`/settings/shell-attributes/${saved.id}`);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'signInRequired' || apiError.kind === 'sessionExpired') {
      return;
    }
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      return;
    }
    if (apiError.kind === 'conflict' && attribute) {
      // The typed values stay; the form takes the version with the other person's change, so
      // Save again replaces it on purpose (feature 001's edit-form rule).
      const latest = await fetchAttribute(attribute.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({
        message: conflictMessage(apiError, 'Shell attribute', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      return;
    }
    setFailure({ message: apiError.message });
  }

  return (
    <>
      <form noValidate aria-busy={saving} className={styles.editor} onSubmit={submit}>
        {failure && <FailureMessage failure={failure} />}
        <FormErrorSummary errors={problems} />

        <FormSection title="Basics">
          <TextField
            label="Label"
            name="label"
            marking="Required"
            maxLength={80}
            value={values.label}
            onChange={(label) => change({ label })}
            error={problemOf(problems, 'label')}
          />
          <TextAreaField
            label="Help text"
            name="helpText"
            hint="Shown as an ⓘ tip next to the field on forms."
            maxLength={300}
            rows={3}
            value={values.helpText}
            onChange={(helpText) => change({ helpText })}
            error={problemOf(problems, 'helpText')}
          />
          {levelLock ? (
            <FixedField label="Level" value={LEVEL_WORDS[values.level]}>
              <LockedNote>{levelLock}</LockedNote>
            </FixedField>
          ) : (
            <ChoiceField
              label="Level"
              name="level"
              marking="Required"
              options={LEVELS.map((each) => ({ id: each, label: LEVEL_WORDS[each], active: true }))}
              value={values.level}
              onChange={changeLevel}
              error={problemOf(problems, 'level')}
            />
          )}
          {typeLock && attribute ? (
            <FixedField label="Type" value={TYPE_WORDS[attribute.type]}>
              <LockedNote>{typeLock}</LockedNote>
            </FixedField>
          ) : (
            <ChoiceField
              label="Type"
              name="type"
              marking="Required"
              hint={values.type === 'multipleChoiceSplit' ? SPLIT_HINT : undefined}
              options={typesFor(values.level).map((each) => ({
                id: each,
                label: TYPE_WORDS[each],
                active: true,
              }))}
              value={values.type}
              onChange={changeType}
              error={problemOf(problems, 'type')}
            />
          )}
          {values.type === 'yesNo' ? (
            <FixedField label="Marking" value={MARKING_WORDS.optional}>
              <p className={styles.hint}>{YES_NO_HINT}</p>
            </FixedField>
          ) : (
            <ChoiceField
              label="Marking"
              name="marking"
              marking="Required"
              options={markingsFor(values.level).map((each) => ({
                id: each,
                label: MARKING_WORDS[each],
                active: true,
              }))}
              value={values.marking}
              onChange={(marking) => change({ marking })}
              error={problemOf(problems, 'marking')}
            />
          )}
        </FormSection>

        <FormSection title="Place on the form">
          {sectionLock ? (
            <FixedField label="Section" value={values.section}>
              <LockedNote>{sectionLock}</LockedNote>
            </FixedField>
          ) : (
            <TextField
              label="Section"
              name="section"
              hint="Attributes of the same section are shown together; sections appear in the order of their first attribute."
              maxLength={60}
              suggestions={sections}
              value={values.section}
              onChange={(section) => change({ section })}
              error={problemOf(problems, 'section')}
            />
          )}
        </FormSection>

        {!attribute && isChoiceType(values.type) && (
          <FormSection title="Options">
            <OptionsEditor rows={rows} onChange={setRows} problems={problems} />
          </FormSection>
        )}

        <FormFooter>
          <Button type="submit" variant="primary">
            Save
          </Button>
          <Button onClick={() => navigate(backTo)}>Cancel</Button>
        </FormFooter>
      </form>
      {dialog}
      {warnings && attribute && (
        <ConfirmDialog
          open
          onOpenChange={(open) => {
            if (!open) {
              setWarnings(null);
            }
          }}
          title={`Save changes to Shell attribute '${attribute.label}'?`}
          confirmLabel="Save changes"
          cancelLabel="Keep editing"
          busy={saving}
          onConfirm={save}
        >
          <ul className={styles.impactList}>
            {warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </ConfirmDialog>
      )}
    </>
  );
}

/** The problems of the first options that can be seen here (FR-018); the API checks again. */
function optionProblems(rows: OptionRow[]): FieldProblem[] {
  if (rows.length === 0) {
    return [{ field: 'options', message: 'Add at least one option.' }];
  }
  const found: FieldProblem[] = [];
  const seen = new Set<string>();
  const reported = new Set<string>();
  rows.forEach((row, index) => {
    const label = row.label.trim();
    const field = `options[${index}].label`;
    if (label === '') {
      found.push({ field, message: 'Enter a label for every option.' });
      return;
    }
    const key = label.toLowerCase();
    if (seen.has(key) && !reported.has(key)) {
      found.push({ field, message: `'${label}' appears twice.` });
      reported.add(key);
    }
    seen.add(key);
  });
  return found;
}

interface FixedFieldProps {
  label: string;
  value: string;
  /** Why it can't change here: a LockedNote, or a hint. */
  children: ReactNode;
}

/** A field that can't be changed here, shown as its value with the reason (design system §6.6). */
function FixedField({ label, value, children }: FixedFieldProps) {
  return (
    <div className={styles.lockedField}>
      <span className={styles.lockedLabel}>{label}</span>
      <span className={styles.lockedValue}>{value}</span>
      {children}
    </div>
  );
}
