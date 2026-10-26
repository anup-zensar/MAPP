import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useRef, useState } from 'react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import type { Resolver } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormField } from '@/ui/FormField/FormField';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { FormSection } from '@/ui/FormSection/FormSection';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import { useAttributeForm } from '../api';
import { ImpactDialog } from '../ImpactDialog';
import { saveFailureOf } from '../save-failure';
import { changedFields, createBody, startValues, updateBody } from './attribute-values';
import { findFormAttribute, isChoiceType } from './attribute-words';
import type { AttributeLevel } from './attribute-words';
import {
  checkAttributeImpact,
  fetchAttribute,
  useAllShellAttributes,
  useCreateAttribute,
  useUpdateAttribute,
} from './api';
import type { ChangeImpact, ShellAttribute } from './api';
import { BasicsSection } from './BasicsSection';
import { DependsOnSection } from './DependsOnSection';
import { OptionLinksEditor, OptionsEditor } from './OptionsEditor';
import { PlaceSection } from './PlaceSection';
import { attributeProblems, attributeSchema } from './schemas';
import type { AttributeRules, AttributeValues } from './schemas';
import styles from './ShellAttributes.module.css';

const FIELDS = [
  'label',
  'helpText',
  'level',
  'type',
  'marking',
  'section',
  'parentAttributeId',
  'conditionOptionIds',
] as const;

export interface AttributeFormProps {
  /** The level of a new attribute, or the level of the attribute being edited. */
  level: AttributeLevel;
  /** The attribute to edit; without one, the form creates a new attribute. */
  attribute?: ShellAttribute;
}

/** The Shell attribute editor's form, its save flow and its dialogs. */
export function AttributeForm({ level, attribute }: AttributeFormProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const attributes = useAllShellAttributes();
  const createAttribute = useCreateAttribute();
  const updateAttribute = useUpdateAttribute();
  const [start] = useState(() => startValues(attribute, level));
  // The checks need the level's form, which arrives after the form exists: they read the latest.
  const latestRules = useRef<AttributeRules>({ isNew: !attribute, optionsToLink: null });
  const resolver: Resolver<AttributeValues> = (values, context, options) =>
    zodResolver(attributeSchema(latestRules.current))(values, context, options);
  const form = useForm<AttributeValues>({ resolver, defaultValues: start, ...CHECK_ON_SAVE });
  // The options' problems carry the API's names, so they are shown as the API's would be.
  const {
    problems,
    showServerProblems: showProblems,
    clearProblems,
  } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(attribute?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  // The warnings of the impact check, waiting for Save changes or Keep editing (FR-021).
  const [warnings, setWarnings] = useState<ChangeImpact['warnings'] | null>(null);
  // Where to go once saved; leaving is then no longer "losing changes".
  const [leaveTo, setLeaveTo] = useState<string | null>(null);
  const { dialog } = useUnsavedChanges(leaveTo === null && form.formState.isDirty);
  const [chosenLevel, type, parentAttributeId] = useWatch({
    control: form.control,
    name: ['level', 'type', 'parentAttributeId'],
  });

  // The level's form: the Business unit options and the options of the attribute it depends on.
  const levelForm = useAttributeForm(chosenLevel);
  // The attribute's own options (edit), which need links when it starts depending on another.
  const ownForm = useAttributeForm(attribute?.level ?? chosenLevel);
  const isChoice = isChoiceType(type);
  const parent =
    levelForm.data && parentAttributeId
      ? findFormAttribute(levelForm.data, parentAttributeId)
      : undefined;
  // Edit: an attribute that starts depending on another links each active option to it.
  const needsLinks =
    attribute !== undefined &&
    isChoice &&
    parentAttributeId !== '' &&
    parentAttributeId !== start.parentAttributeId;
  const ownOptions =
    (attribute && ownForm.data && findFormAttribute(ownForm.data, attribute.id)?.options) || [];
  const activeOwnOptions = ownOptions.filter((option) => option.active);
  const rules: AttributeRules = {
    isNew: !attribute,
    parent,
    optionsToLink: needsLinks ? activeOwnOptions : null,
  };
  const backTo = attribute
    ? `/settings/shell-attributes/${attribute.id}`
    : `/settings/shell-attributes?level=${chosenLevel}`;

  useEffect(() => {
    latestRules.current = rules;
  });

  useEffect(() => {
    if (leaveTo !== null) {
      navigate(leaveTo);
    }
  }, [leaveTo, navigate]);

  async function check(values: AttributeValues) {
    clearProblems();
    setFailure(null);
    if (!attribute) {
      await create(values);
      return;
    }
    const changed = changedFields(values, start);
    if (changed.length === 0) {
      setLeaveTo(backTo);
      return;
    }
    // A new marking or type can make records incomplete or remove options: say so first.
    if (changed.includes('marking') || changed.includes('type')) {
      setSaving(true);
      try {
        const body = updateBody(values, changed, version, rules.optionsToLink);
        const impact = await checkAttributeImpact({ id: attribute.id, body });
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
    await save(values);
  }

  async function create(values: AttributeValues) {
    setSaving(true);
    try {
      const created = await createAttribute.mutateAsync(createBody(values, parent));
      show({ message: `Shell attribute '${created.label}' created` });
      setLeaveTo(`/settings/shell-attributes/${created.id}`);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function save(values: AttributeValues) {
    if (!attribute) {
      return;
    }
    setWarnings(null);
    setSaving(true);
    try {
      const body = updateBody(values, changedFields(values, start), version, rules.optionsToLink);
      const saved = await updateAttribute.mutateAsync({ id: attribute.id, body });
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
    showProblems(view.problems);
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

  const submit = form.handleSubmit(check, () => {
    showProblems(attributeProblems(form.getValues(), rules));
    setFailure(null);
  });
  const allAttributes = attributes.data?.items ?? [];

  return (
    <FormProvider {...form}>
      <form
        noValidate
        aria-busy={saving}
        className={styles.editor}
        onSubmit={(event) => {
          event.preventDefault();
          if (!saving) {
            void submit(event);
          }
        }}
      >
        {failure && <FailureMessage failure={failure} />}
        <FormErrorSummary errors={problems} />
        <BasicsSection attribute={attribute} />
        <PlaceSection attribute={attribute} allAttributes={allAttributes} />
        <DependsOnSection
          attribute={attribute}
          allAttributes={allAttributes}
          levelForm={levelForm}
        />
        {!attribute && isChoice && (
          <FormSection title="Options">
            <FormField
              control={form.control}
              name="options"
              render={({ value, onChange }) => (
                <OptionsEditor
                  rows={value}
                  onChange={onChange}
                  problems={problems}
                  parent={parent}
                />
              )}
            />
          </FormSection>
        )}
        {needsLinks && parent && (
          <FormSection title="Options">
            {ownForm.data ? (
              <FormField
                control={form.control}
                name="optionLinks"
                render={({ value, onChange }) => (
                  <OptionLinksEditor
                    options={activeOwnOptions}
                    parent={parent}
                    links={value}
                    onChange={onChange}
                    problems={problems}
                  />
                )}
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
          onConfirm={() => void save(form.getValues())}
          onCancel={() => setWarnings(null)}
        />
      )}
    </FormProvider>
  );
}
