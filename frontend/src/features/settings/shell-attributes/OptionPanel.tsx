import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormField } from '@/ui/FormField/FormField';
import type { FieldBinding } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import { useAttributeForm } from '../api';
import { saveFailureOf } from '../save-failure';
import { findFormAttribute } from './attribute-words';
import type { ShellAttribute } from './api';
import { LinkedToField } from './LinkedToField';
import { fetchOption, useCreateOption, useUpdateOption } from './options-api';
import type { OptionChange, ShellAttributeOption } from './options-api';
import { optionSchema } from './schemas';
import type { OptionValues } from './schemas';
import styles from './ShellAttributes.module.css';

const FIELDS = ['label', 'parentOptionIds'] as const;

export interface OptionPanelProps {
  attribute: ShellAttribute;
  /** The option to change; without one, the panel adds a new option at the end of the list. */
  option?: ShellAttributeOption;
  onClose: () => void;
}

/**
 * "Add option" or "Edit option '<label>'" (feature 002, FR-018, FR-022): the label and — for an
 * attribute that depends on another — the parent options it is linked to; every problem at once,
 * the conflict message when someone else saved first, and unsaved changes protected.
 */
export function OptionPanel({ attribute, option, onClose }: OptionPanelProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createOption = useCreateOption();
  const updateOption = useUpdateOption();
  const parentRef = attribute.parentAttribute;
  const dependent = parentRef !== null;
  // The edit-form rule: the option, its version included, is copied into the form once.
  const [start] = useState<OptionValues>(() => ({
    label: option?.label ?? '',
    parentOptionIds: option?.parentOptions.map((each) => each.id) ?? [],
  }));
  const form = useForm<OptionValues>({
    resolver: zodResolver(optionSchema(parentRef?.name ?? null)),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(option?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save({ label, parentOptionIds }: OptionValues) {
    clearProblems();
    setSaving(true);
    try {
      if (option) {
        const body: OptionChange = { id: option.id, version };
        if (label !== start.label) {
          body.label = label;
        }
        if (dependent && !sameIds(parentOptionIds, start.parentOptionIds)) {
          body.parentOptionIds = parentOptionIds;
        }
        if (body.label === undefined && body.parentOptionIds === undefined) {
          onClose();
          return;
        }
        const saved = await updateOption.mutateAsync(body);
        show({ message: `Option '${saved.label}' saved` });
      } else {
        const created = await createOption.mutateAsync({
          attributeId: attribute.id,
          label,
          parentOptionIds: dependent ? parentOptionIds : undefined,
        });
        show({ message: `Option '${created.label}' added` });
      }
      onClose();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const view = saveFailureOf(error, 'option', dateFormat);
    showServerProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && option) {
      // What was typed stays; the form takes the version with the other person's change, so
      // Save again replaces it on purpose.
      const latest = await fetchOption(option.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  const submit = form.handleSubmit(save, () => {
    onInvalid();
    setFailure(null);
  });

  return (
    <FormPanel
      title={option ? `Edit option '${option.label}'` : 'Add option'}
      description={attribute.label}
      isDirty={form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void submit()}
      onClose={onClose}
    >
      <div className={styles.panelFields}>
        <FormField
          control={form.control}
          name="label"
          render={(field) => (
            <TextField label="Label" marking="Required" maxLength={120} {...field} />
          )}
        />
        {parentRef && (
          <FormField
            control={form.control}
            name="parentOptionIds"
            render={(field) => (
              <ParentLinks level={attribute.level} parentId={parentRef.id} {...field} />
            )}
          />
        )}
      </div>
    </FormPanel>
  );
}

interface ParentLinksProps extends FieldBinding<string[]> {
  level: ShellAttribute['level'];
  /** The attribute the option's attribute depends on. */
  parentId: string;
}

/**
 * Linked to, once the level's form has arrived: it holds every option of the parent, also beyond
 * the first 100 the options list shows.
 */
function ParentLinks({ level, parentId, name, value, onChange, error }: ParentLinksProps) {
  const form = useAttributeForm(level);
  const parent = form.data ? findFormAttribute(form.data, parentId) : undefined;
  if (parent) {
    return (
      <LinkedToField name={name} parent={parent} value={value} onChange={onChange} error={error} />
    );
  }
  if (form.isError) {
    return <ErrorState error={form.error} onRetry={() => void form.refetch()} />;
  }
  return <Skeleton lines={3} />;
}

function sameIds(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
}
