import { useEffect, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { useAttributeForm } from '@/api/settingsQueries';
import {
  Button,
  Card,
  ChoiceField,
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
import type { ChoiceOption, FieldProblem, SaveFailure } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { useUnsavedChanges } from '@/lib/useUnsavedChanges';
import { ImpactDialog } from '../shared/ImpactDialog';
import { saveFailureOf } from '../shared/saveFailure';
import {
  conditionText,
  DEFAULT_SECTION,
  findFormAttribute,
  isChoiceType,
  isLevel,
  LEVEL_WORDS,
  LEVELS,
  linkOptionText,
  lockReason,
  MARKING_WORDS,
  markingsFor,
  parentCandidates,
  TYPE_WORDS,
  typesFor,
} from './attributeWords';
import type { AttributeLevel, FormAttribute } from './attributeWords';
import { OptionLinksEditor, OptionsEditor } from './OptionsEditor';
import type { OptionRow } from './OptionsEditor';
import {
  checkAttributeImpact,
  fetchAttribute,
  useAllShellAttributes,
  useCreateAttribute,
  useShellAttribute,
  useUpdateAttribute,
} from './shellAttributesApi';
import type { ChangeImpact, ShellAttribute, ShellAttributeUpdate } from './shellAttributesApi';
import { AttributeNotFound } from './ShellAttributePage';
import styles from './ShellAttributes.module.css';

const YES_NO_HINT = 'Always optional; shown as a tick box that defaults to No.';
const SPLIT_HINT =
  "People split the record's cost over the chosen values; the shares must add up to 100 %.";
const CONDITION_LABEL = "Shown when the plan's business unit includes";
const NO_BUSINESS_UNIT = "There is no Business unit attribute yet, so conditions can't be set.";
/** The "None" entry of Depends on; the API takes "" for it. */
const NONE = 'none';

export interface ShellAttributeEditorPageProps {
  /** create: /settings/shell-attributes/new; edit: /settings/shell-attributes/:attributeId/edit. */
  mode: 'create' | 'edit';
}

/**
 * The Shell attribute editor (feature 002, US2, US3, FR-016 – FR-023), a full page for create and
 * edit: Basics, Place on the form, Depends on and condition and — on create, or when an attribute
 * starts depending on another — its options with their links; locked fields with their reason;
 * every problem at once; the impact check before a change of marking or type; unsaved changes
 * protected.
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
      <PageHeader
        breadcrumb={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: `/settings/shell-attributes?level=${level}` },
          { label: 'New Shell attribute' },
        ]}
        title="New Shell attribute"
      />
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
        <PageHeader
          breadcrumb={[
            { label: 'Settings', to: '/settings' },
            { label: 'Shell attributes', to: '/settings/shell-attributes' },
            {
              label: `${LEVEL_WORDS[level]} form`,
              to: `/settings/shell-attributes?level=${level}`,
            },
            { label, to: `/settings/shell-attributes/${id}` },
            { label: 'Edit' },
          ]}
          title={`Edit Shell attribute '${label}'`}
        />
        {/* The edit-form rule: the attribute is copied into the form once, when it opens. */}
        <AttributeForm key={id} level={level} attribute={attribute.data} />
      </>
    );
  }

  const notFound = attribute.error instanceof ApiError && attribute.error.kind === 'notFound';
  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: '/settings/shell-attributes' },
          { label: 'Edit' },
        ]}
        title="Edit Shell attribute"
      />
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
  /** Depends on: "" for none (FR-022). */
  parentAttributeId: string;
  /** The Business unit options of the condition; none means always shown (FR-023). */
  conditionOptionIds: string[];
}

/** The text fields an edit can change and send (D-203: only the changed ones). */
const TEXT_FIELDS = [
  'label',
  'helpText',
  'level',
  'type',
  'marking',
  'section',
  'parentAttributeId',
] as const;

type ChangedField = (typeof TEXT_FIELDS)[number] | 'conditionOptionIds';

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
    parentAttributeId: attribute?.parentAttribute?.id ?? '',
    conditionOptionIds: attribute?.condition.map((option) => option.id) ?? [],
  }));
  const [values, setValues] = useState<FormValues>(start);
  const [rows, setRows] = useState<OptionRow[]>([]);
  // When an attribute starts depending on another: its options' new links, by option id.
  const [links, setLinks] = useState<Record<string, string[]>>({});
  const [version, setVersion] = useState(attribute?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  // The warnings of the impact check, waiting for Save changes or Keep editing (FR-021).
  const [warnings, setWarnings] = useState<ChangeImpact['warnings'] | null>(null);
  // Where to go once saved; leaving is then no longer "losing changes".
  const [leaveTo, setLeaveTo] = useState<string | null>(null);

  // The level's form: the Business unit options and the options of the attribute it depends on.
  const levelForm = useAttributeForm(values.level);
  // The attribute's own options (edit), which need links when it starts depending on another.
  const ownForm = useAttributeForm(attribute?.level ?? values.level);

  const changed: ChangedField[] = [
    ...TEXT_FIELDS.filter((field) => values[field] !== start[field]),
    ...(sameIds(values.conditionOptionIds, start.conditionOptionIds)
      ? []
      : (['conditionOptionIds'] as const)),
  ];
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
  const parentLock = attribute && lockReason(attribute, 'parentAttributeId');
  const conditionLock = attribute && lockReason(attribute, 'condition');
  const allAttributes = attributes.data?.items ?? [];
  const sections = [
    ...new Set(
      allAttributes.filter((each) => each.level === values.level).map((each) => each.section),
    ),
  ];
  const isChoice = isChoiceType(values.type);
  const parent =
    levelForm.data && values.parentAttributeId
      ? findFormAttribute(levelForm.data, values.parentAttributeId)
      : undefined;
  // Edit: an attribute that starts depending on another links each active option to it.
  const needsLinks =
    attribute !== undefined &&
    isChoice &&
    values.parentAttributeId !== '' &&
    values.parentAttributeId !== start.parentAttributeId;
  const ownOptions =
    (attribute && ownForm.data && findFormAttribute(ownForm.data, attribute.id)?.options) || [];
  const activeOwnOptions = ownOptions.filter((option) => option.active);
  const backTo = attribute
    ? `/settings/shell-attributes/${attribute.id}`
    : `/settings/shell-attributes?level=${values.level}`;

  function change(next: Partial<FormValues>) {
    setValues((current) => ({ ...current, ...next }));
  }

  /** Links belong to the parent's options: another parent starts them afresh. */
  function changeParent(next: string) {
    const parentAttributeId = next === NONE ? '' : next;
    change({ parentAttributeId });
    setLinks({});
    setRows((current) => current.map((row) => ({ ...row, parentOptionIds: [] })));
  }

  /** A level offers its own types, markings and parents; a choice it doesn't offer is cleared. */
  function changeLevel(next: string) {
    if (!isLevel(next)) {
      return;
    }
    const types: string[] = typesFor(next);
    const markings: string[] = markingsFor(next);
    const type = types.includes(values.type) ? values.type : '';
    const marking = type === 'yesNo' || markings.includes(values.marking) ? values.marking : '';
    change({ level: next, type, marking });
    changeParent(NONE);
  }

  /** Yes/no attributes are always optional (FR-017); only choice attributes depend on others. */
  function changeType(type: string) {
    change(type === 'yesNo' ? { type, marking: 'optional' } : { type });
    if (!isChoiceType(type) && values.parentAttributeId !== '') {
      changeParent(NONE);
    }
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
    if (!attribute && isChoice) {
      found.push(...optionProblems(rows, parent));
    }
    if (needsLinks && parent) {
      for (const option of activeOwnOptions) {
        if ((links[option.id] ?? []).length === 0) {
          found.push({
            field: `optionLinks.${option.id}`,
            message: linkOptionText(option.label, parent.label),
          });
        }
      }
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
          setWarnings(impact.warnings);
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

  /** The changed fields, the links a new parent needs, and the version the person started from. */
  function changes(): ShellAttributeUpdate {
    const body: ShellAttributeUpdate = { version };
    for (const field of changed) {
      if (field === 'conditionOptionIds') {
        body.conditionOptionIds = values.conditionOptionIds;
      } else {
        body[field] = values[field];
      }
    }
    if (needsLinks) {
      body.optionLinks = activeOwnOptions.map((option) => ({
        optionId: option.id,
        parentOptionIds: links[option.id] ?? [],
      }));
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
        parentAttributeId: isChoice ? values.parentAttributeId : '',
        conditionOptionIds: values.conditionOptionIds,
        options: isChoice
          ? rows.map((row) =>
              parent
                ? { label: row.label, parentOptionIds: row.parentOptionIds }
                : { label: row.label },
            )
          : [],
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
    const view = saveFailureOf(error, 'shellAttribute', dateFormat);
    setProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && attribute) {
      // The typed values stay; the form takes the version with the other person's change, so
      // Save again replaces it on purpose (feature 001's edit-form rule).
      const latest = await fetchAttribute(attribute.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  // Depends on: the choice attributes placed before it, and the one it depends on now.
  const candidates = parentCandidates(allAttributes, values.level, values.section, attribute);
  const chosenParent = allAttributes.find((each) => each.id === values.parentAttributeId);
  if (chosenParent && !candidates.includes(chosenParent)) {
    candidates.push(chosenParent);
  }
  const parentOptions: ChoiceOption[] = [
    { id: NONE, label: 'None', active: true },
    ...candidates.map((each) => ({ id: each.id, label: each.label, active: each.active })),
  ];

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

        <FormSection title="Depends on and condition">
          {!levelForm.data ? (
            levelForm.isError ? (
              <ErrorState error={levelForm.error} onRetry={() => void levelForm.refetch()} />
            ) : (
              <Skeleton lines={3} height="18px" />
            )
          ) : (
            <>
              {isChoice &&
                (parentLock && attribute ? (
                  <FixedField label="Depends on" value={attribute.parentAttribute?.name ?? 'None'}>
                    <LockedNote>{parentLock}</LockedNote>
                  </FixedField>
                ) : (
                  <ChoiceField
                    label="Depends on"
                    name="parentAttributeId"
                    hint="Its options are filtered by the value chosen for that attribute."
                    options={parentOptions}
                    value={values.parentAttributeId || NONE}
                    onChange={(next) => changeParent(next || NONE)}
                    error={problemOf(problems, 'parentAttributeId')}
                  />
                ))}
              {conditionLock && attribute ? (
                <FixedField
                  label={CONDITION_LABEL}
                  value={conditionText(attribute) ?? 'Always shown'}
                >
                  <LockedNote>{conditionLock}</LockedNote>
                </FixedField>
              ) : levelForm.data.businessUnitAttributeId === null ? (
                <FixedField label={CONDITION_LABEL} value="Always shown">
                  <p className={styles.hint}>{NO_BUSINESS_UNIT}</p>
                </FixedField>
              ) : (
                <ChoiceField
                  label={CONDITION_LABEL}
                  name="conditionOptionIds"
                  hint="Leave all unticked to show it always."
                  multiple
                  options={levelForm.data.businessUnitOptions.map((option) => ({
                    id: option.id,
                    label: option.label,
                    active: option.active,
                  }))}
                  value={values.conditionOptionIds}
                  onChange={(conditionOptionIds) => change({ conditionOptionIds })}
                  error={problemOf(problems, 'conditionOptionIds')}
                />
              )}
            </>
          )}
        </FormSection>

        {!attribute && isChoice && (
          <FormSection title="Options">
            <OptionsEditor rows={rows} onChange={setRows} problems={problems} parent={parent} />
          </FormSection>
        )}
        {needsLinks && parent && (
          <FormSection title="Options">
            {ownForm.data ? (
              <OptionLinksEditor
                options={activeOwnOptions}
                parent={parent}
                links={links}
                onChange={setLinks}
                problems={problems}
              />
            ) : ownForm.isError ? (
              <ErrorState error={ownForm.error} onRetry={() => void ownForm.refetch()} />
            ) : (
              <Skeleton lines={3} height="18px" />
            )}
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
        <ImpactDialog
          kind="shellAttribute"
          name={attribute.label}
          warnings={warnings}
          busy={saving}
          onConfirm={() => void save()}
          onCancel={() => setWarnings(null)}
        />
      )}
    </>
  );
}

/**
 * The problems of the first options that can be seen here (FR-018, FR-022), with the API's words;
 * the API checks again.
 */
function optionProblems(rows: OptionRow[], parent: FormAttribute | undefined): FieldProblem[] {
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
    } else {
      const key = label.toLowerCase();
      if (seen.has(key) && !reported.has(key)) {
        found.push({ field, message: `'${label}' appears twice.` });
        reported.add(key);
      }
      seen.add(key);
    }
    if (parent && row.parentOptionIds.length === 0) {
      found.push({
        field: `options[${index}].parentOptionIds`,
        message: linkOptionText(label, parent.label),
      });
    }
  });
  return found;
}

function sameIds(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
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
