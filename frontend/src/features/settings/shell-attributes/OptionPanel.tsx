import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormPanel, problemOf, TextField, useToast } from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { fetchOptions, useCreateOption, useUpdateOption } from './shellAttributesApi';
import type { ShellAttribute, ShellAttributeOption } from './shellAttributesApi';
import { CONFLICT_NEXT_STEP } from './useStatusChange';

export interface OptionPanelProps {
  attribute: ShellAttribute;
  /** The option to rename; without one, the panel adds a new option at the end of the list. */
  option?: ShellAttributeOption;
  onClose: () => void;
}

/**
 * "Add option" or "Edit option '<label>'" (feature 002, FR-018): the label, every problem at
 * once, the conflict message when someone else saved first, and unsaved changes protected.
 */
export function OptionPanel({ attribute, option, onClose }: OptionPanelProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createOption = useCreateOption();
  const updateOption = useUpdateOption();
  // The edit-form rule: the option, its version included, is copied into the form once.
  const startLabel = option?.label ?? '';
  const [label, setLabel] = useState(startLabel);
  const [version, setVersion] = useState(option?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save() {
    // Instant feedback for the one required field; the API checks everything else.
    if (label.trim() === '') {
      setProblems([{ field: 'label', message: 'Enter a label.' }]);
      setFailure(null);
      return;
    }
    setSaving(true);
    try {
      if (option) {
        const saved = await updateOption.mutateAsync({ id: option.id, label, version });
        show({ message: `Option '${saved.label}' saved` });
      } else {
        const created = await createOption.mutateAsync({ attributeId: attribute.id, label });
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
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      setFailure(null);
      return;
    }
    setProblems([]);
    if (apiError.kind === 'conflict' && option) {
      // The typed label stays; the form takes the version with the other person's change, so
      // Save again replaces it on purpose.
      const latest = await fetchOptions(attribute.id, { q: option.label, status: 'all' })
        .then((page) => page.items.find((each) => each.id === option.id))
        .catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({
        message: conflictMessage(apiError, 'option', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      return;
    }
    setFailure({ message: apiError.message });
  }

  return (
    <FormPanel
      title={option ? `Edit option '${option.label}'` : 'Add option'}
      description={attribute.label}
      isDirty={label !== startLabel}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={save}
      onClose={onClose}
    >
      <TextField
        label="Label"
        name="label"
        marking="Required"
        maxLength={120}
        value={label}
        onChange={setLabel}
        error={problemOf(problems, 'label')}
      />
    </FormPanel>
  );
}
