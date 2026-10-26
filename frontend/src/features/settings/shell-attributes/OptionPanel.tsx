import { useState } from 'react';
import { useAttributeForm } from '../api';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import { useFormatters } from '@/format/useFormatters';
import { saveFailureOf } from '../save-failure';
import { findFormAttribute, linkOptionText } from './attributeWords';
import { LinkedToField } from './LinkedToField';
import { fetchOption, useCreateOption, useUpdateOption } from './shellAttributesApi';
import type { OptionChange, ShellAttribute, ShellAttributeOption } from './shellAttributesApi';
import styles from './ShellAttributes.module.css';

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
  const startLabel = option?.label ?? '';
  const [startLinks] = useState(() => option?.parentOptions.map((each) => each.id) ?? []);
  const [label, setLabel] = useState(startLabel);
  const [links, setLinks] = useState(startLinks);
  const [version, setVersion] = useState(option?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const linksChanged = !sameIds(links, startLinks);

  /** Instant feedback for what can be checked here, with the API's words; the API checks again. */
  function localProblems(): FieldProblem[] {
    const found: FieldProblem[] = [];
    if (label.trim() === '') {
      found.push({ field: 'label', message: 'Enter a label.' });
    }
    if (parentRef && links.length === 0) {
      found.push({ field: 'parentOptionIds', message: linkOptionText(label, parentRef.name) });
    }
    return found;
  }

  async function save() {
    const found = localProblems();
    if (found.length > 0) {
      setProblems(found);
      setFailure(null);
      return;
    }
    setSaving(true);
    try {
      if (option) {
        const body: OptionChange = { id: option.id, version };
        if (label !== startLabel) {
          body.label = label;
        }
        if (dependent && linksChanged) {
          body.parentOptionIds = links;
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
          parentOptionIds: dependent ? links : undefined,
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
    setProblems(view.problems);
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

  return (
    <FormPanel
      title={option ? `Edit option '${option.label}'` : 'Add option'}
      description={attribute.label}
      isDirty={label !== startLabel || linksChanged}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void save()}
      onClose={onClose}
    >
      <div className={styles.panelFields}>
        <TextField
          label="Label"
          name="label"
          marking="Required"
          maxLength={120}
          value={label}
          onChange={setLabel}
          error={problemOf(problems, 'label')}
        />
        {parentRef && (
          <ParentLinks
            level={attribute.level}
            parentId={parentRef.id}
            value={links}
            onChange={setLinks}
            error={problemOf(problems, 'parentOptionIds')}
          />
        )}
      </div>
    </FormPanel>
  );
}

interface ParentLinksProps {
  level: ShellAttribute['level'];
  /** The attribute the option's attribute depends on. */
  parentId: string;
  value: string[];
  onChange: (ids: string[]) => void;
  error?: string;
}

/**
 * Linked to, once the level's form has arrived: it holds every option of the parent, also beyond
 * the first 100 the options list shows.
 */
function ParentLinks({ level, parentId, value, onChange, error }: ParentLinksProps) {
  const form = useAttributeForm(level);
  const parent = form.data ? findFormAttribute(form.data, parentId) : undefined;
  if (parent) {
    return (
      <LinkedToField
        name="parentOptionIds"
        parent={parent}
        value={value}
        onChange={onChange}
        error={error}
      />
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
