import { zodResolver } from '@hookform/resolvers/zod';
import { RotateCcw } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { ApiError, conflictMessage } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import { useCloseAfterSave } from '@/ui/useCloseAfterSave';
import { CONFLICT_NEXT_STEP } from '../save-failure';
import { fetchFieldHelp, HELP_MAX_LENGTH, useUpdateFieldHelp } from './api';
import type { FieldHelpEntry } from './api';
import { entryName } from './field-help-words';
import { ResetHelpDialog } from './ResetHelpDialog';
import { fieldHelpSchema, oneParagraph } from './schemas';
import type { FieldHelpValues } from './schemas';
import styles from './field-help.module.css';

const FIELDS = ['text'] as const;

export interface FieldHelpPanelProps {
  entry: FieldHelpEntry;
  onClose: () => void;
}

/**
 * Edit field help (016 FR-046) in a side panel: the text (plain, 0–300 characters, counting down
 * near the limit; empty hides the ⓘ), the default beside it and **Reset to default** with a
 * question first. A save over a newer change is refused, naming who and when.
 */
export function FieldHelpPanel({ entry, onClose }: FieldHelpPanelProps) {
  const { show } = useToast();
  const { dateFormat, dateTime } = useFormatters();
  const update = useUpdateFieldHelp();
  const [start] = useState(entry.text);
  const form = useForm<FieldHelpValues>({
    resolver: zodResolver(fieldHelpSchema),
    defaultValues: { text: entry.text },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(entry.version);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [askReset, setAskReset] = useState(false);
  // The panel is open while the address names the entry: it closes once the save is known.
  const { saved, closeAfterSave } = useCloseAfterSave(onClose);
  const name = `'${entry.label}'`;

  async function save(values: FieldHelpValues) {
    const typed = oneParagraph(values.text);
    if (typed === start.trim()) {
      // Only spaces or line breaks changed: nothing to save.
      closeAfterSave();
      return;
    }
    clearProblems();
    setSaving(true);
    try {
      await update.mutateAsync({ key: entry.key, body: { text: typed, version } });
      show({ message: `Help of ${name} saved` });
      closeAfterSave();
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
      showServerProblems(apiError.fields);
      setFailure(null);
      return;
    }
    if (apiError.kind === 'conflict') {
      setFailure({
        message: conflictMessage(apiError, 'help text', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      const latest = await fetchFieldHelp(entry.key).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      return;
    }
    setFailure({ message: apiError.message });
  }

  const submit = form.handleSubmit(save, () => {
    onInvalid();
    setFailure(null);
  });

  return (
    <>
      <FormPanel
        title="Edit field help"
        description={entryName(entry.level, entry.label)}
        isDirty={!saved && form.formState.isDirty}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={() => void submit()}
        onClose={onClose}
      >
        <div className={styles.form}>
          <FormField
            control={form.control}
            name="text"
            render={(field) => (
              <TextAreaField
                label="Help text"
                hint="Plain text, shown in the ⓘ beside the field. Leave it empty to hide the ⓘ."
                maxLength={HELP_MAX_LENGTH}
                rows={5}
                {...field}
              />
            )}
          />
          <div className={styles.defaultText}>
            <h3>Default text</h3>
            <p>{entry.defaultText}</p>
            {entry.isDefault ? (
              <p className={styles.meta}>This field uses the default text.</p>
            ) : (
              <>
                {entry.updatedBy && entry.updatedAt && (
                  <p className={styles.meta}>
                    {`Edited by ${entry.updatedBy.name}, ${dateTime(entry.updatedAt)}`}
                  </p>
                )}
                <div>
                  <Button size="sm" icon={<RotateCcw />} onClick={() => setAskReset(true)}>
                    Reset to default
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      </FormPanel>
      {askReset && (
        <ResetHelpDialog
          entry={entry}
          version={version}
          onDone={closeAfterSave}
          onCancel={() => setAskReset(false)}
        />
      )}
    </>
  );
}
